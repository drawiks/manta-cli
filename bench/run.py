#!/usr/bin/env python3
"""Resource benchmark: manta-cli vs odota-cli + odota/parser.

Measures wall time, CPU time and peak RSS for both variants on the same
replays. Results are written to bench/results/ (CSV, Markdown, SVG charts).

This script is intentionally dependency-free (stdlib only) and does not
modify the production code.
"""
from __future__ import annotations

import csv
import json
import math
import os
import random
import shlex
import statistics
import subprocess
import sys
import threading
import time
from dataclasses import dataclass, asdict
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
BENCH = ROOT / "bench"
RESULTS = BENCH / "results"
CHARTS = RESULTS / "charts"
BIN = Path("/tmp/bench")
MANTA = BIN / "manta_cli"
ODOTA = BIN / "odota_cli"
CONTAINER = "parser-bench"
CONTAINER_CONSTRAINED = "parser-bench-1c"
PORT = 5600
PORT_CONSTRAINED = 5601
CLK_TCK = os.sysconf("SC_CLK_TCK")

REPLAYS = [
    ("9064911131", Path("/home/draw/replays/9064911131.dem"), Path("/home/draw/replays/9064911131.dem.bz2")),
    ("9064082159", Path("/home/draw/replays/9064082159.dem"), Path("/home/draw/replays/9064082159.dem.bz2")),
    ("9048749977", Path("/home/draw/replays/9048749977.dem"), Path("/home/draw/replays/9048749977.dem.bz2")),
    ("9053590215", Path("/home/draw/9053590215.dem"), None),
    ("9065268582", Path("/home/draw/9065268582.dem"), None),
]

MODES = ["manta_dem", "manta_bz2", "odota_e2e", "odota_offline", "parser_only"]
RUNS = 3
WARMUP = 1


@dataclass
class Run:
    replay: str
    size_mib: float
    file_mib: float
    mode: str
    run: int
    wall_s: float
    child_user_s: float
    child_sys_s: float
    server_user_s: float
    server_sys_s: float
    cpu_total_s: float
    cpu_pct: float
    child_rss_mib: float
    server_peak_rss_mib: float
    server_steady_rss_mib: float
    throughput_mib_s: float
    http_code: str = ""
    output_bytes: int = 0
    note: str = ""


def sh(cmd: list[str], timeout: int = 600) -> subprocess.CompletedProcess:
    return subprocess.run(cmd, text=True, capture_output=True, timeout=timeout)


def proc_stat(pid: int) -> tuple[float, float]:
    try:
        data = Path(f"/proc/{pid}/stat").read_text()
        # comm may contain spaces/parentheses: split after the last ')'
        rest = data[data.rindex(")") + 2:].split()
        utime = int(rest[11]) / CLK_TCK
        stime = int(rest[12]) / CLK_TCK
        return utime, stime
    except Exception:
        return 0.0, 0.0


def proc_status(pid: int) -> tuple[float, float]:
    rss = hwm = 0.0
    try:
        for line in Path(f"/proc/{pid}/status").read_text().splitlines():
            if line.startswith("VmRSS:"):
                rss = int(line.split()[1]) / 1024.0
            elif line.startswith("VmHWM:"):
                hwm = int(line.split()[1]) / 1024.0
    except Exception:
        pass
    return rss, hwm


def proc_io(pid: int) -> tuple[int, int]:
    try:
        vals = {}
        for line in Path(f"/proc/{pid}/io").read_text().splitlines():
            k, v = line.split(":", 1)
            vals[k] = int(v.strip())
        return vals.get("read_bytes", 0), vals.get("syscr", 0)
    except Exception:
        return 0, 0


def server_pid(name: str = CONTAINER) -> int:
    p = sh(["docker", "inspect", "-f", "{{.State.Pid}}", name])
    try:
        return int(p.stdout.strip())
    except Exception:
        return 0


class Sampler:
    def __init__(self, pid: int, interval: float = 0.05):
        self.pid = pid
        self.interval = interval
        self.stop = threading.Event()
        self.peak = 0.0
        self.thread = threading.Thread(target=self._run, daemon=True)

    def _run(self) -> None:
        while not self.stop.is_set():
            rss, _ = proc_status(self.pid)
            if rss > self.peak:
                self.peak = rss
            time.sleep(self.interval)

    def __enter__(self):
        self.thread.start()
        return self

    def __exit__(self, *exc):
        self.stop.set()
        self.thread.join(timeout=1)


def wait_health(port: int, timeout: int = 90) -> float:
    start = time.perf_counter()
    while time.perf_counter() - start < timeout:
        p = sh(["curl", "-fsS", "-m", "1", f"http://localhost:{port}/healthz"])
        if p.returncode == 0:
            return time.perf_counter() - start
        time.sleep(0.2)
    raise RuntimeError(f"parser on :{port} did not become healthy")


def start_container(name: str, port: int, extra: list[str] | None = None,
                    container_port: int = 5600) -> float:
    sh(["docker", "rm", "-f", name])
    cmd = ["docker", "run", "-d", "--rm", "--name", name,
           "-p", f"{port}:{container_port}", "-e", "PARSER_WORKERS=1"]
    if extra:
        cmd += extra
    cmd += ["odota/parser:latest"]
    p = sh(cmd)
    if p.returncode != 0:
        raise RuntimeError(p.stderr)
    return wait_health(port)


def stop_container(name: str) -> None:
    sh(["docker", "rm", "-f", name])


def run_child(cmd: list[str], stdout_path: str | None = None,
              sample_server: bool = False, server_name: str = CONTAINER) -> Run:
    """Run one command, return wall/CPU/RSS. stdout_path captures stdout."""
    spid = server_pid(server_name) if sample_server else 0
    before_cpu = proc_stat(spid) if spid else (0.0, 0.0)
    before_io = proc_io(spid) if spid else (0, 0)
    sampler = Sampler(spid) if spid else None

    out_path = stdout_path or os.devnull
    flags = os.O_WRONLY | os.O_CREAT | os.O_TRUNC
    actions = [
        (os.POSIX_SPAWN_OPEN, 0, os.devnull, os.O_RDONLY, 0),
        (os.POSIX_SPAWN_OPEN, 1, out_path, flags, 0o644),
        (os.POSIX_SPAWN_OPEN, 2, os.devnull, os.O_WRONLY, 0),
    ]
    if sampler:
        sampler.__enter__()
    start = time.perf_counter()
    pid = os.posix_spawnp(cmd[0], cmd, os.environ, file_actions=actions)
    _, status, ru = os.wait4(pid, 0)
    wall = time.perf_counter() - start
    if sampler:
        sampler.__exit__()
    after_cpu = proc_stat(spid) if spid else (0.0, 0.0)
    after_io = proc_io(spid) if spid else (0, 0)
    steady_rss = proc_status(spid)[0] if spid else 0.0
    _, hwm = proc_status(spid) if spid else (0.0, 0.0)
    peak = max(sampler.peak if sampler else 0.0, hwm)

    child_user = ru.ru_utime
    child_sys = ru.ru_stime
    server_user = max(0.0, after_cpu[0] - before_cpu[0])
    server_sys = max(0.0, after_cpu[1] - before_cpu[1])
    cpu = child_user + child_sys + server_user + server_sys
    child_rss = ru.ru_maxrss / 1024.0
    return Run(
        replay="", size_mib=0, file_mib=0, mode="", run=0,
        wall_s=wall,
        child_user_s=child_user, child_sys_s=child_sys,
        server_user_s=server_user, server_sys_s=server_sys,
        cpu_total_s=cpu, cpu_pct=(cpu / wall * 100.0) if wall > 0 else 0.0,
        child_rss_mib=child_rss, server_peak_rss_mib=peak,
        server_steady_rss_mib=steady_rss,
        throughput_mib_s=0.0,
        output_bytes=0,
        note=f"exit={os.waitstatus_to_exitcode(status)} io_read_delta={after_io[0]-before_io[0]}",
    )


def generate_ndjson(replay: str, dem: Path) -> Path:
    out = BIN / f"{replay}.ndjson"
    if out.exists() and out.stat().st_size > 0:
        return out
    print(f"[gen] {replay} NDJSON", flush=True)
    r = run_child(["curl", "-sS", "-X", "POST", "--data-binary", f"@{dem}",
                   f"http://localhost:{PORT}/"], stdout_path=str(out), sample_server=True)
    if r.wall_s <= 0:
        raise RuntimeError("empty parser response")
    return out


def measure(replay: str, dem: Path, bz2: Path | None, ndjson: Path,
            mode: str, run_idx: int, port: int = PORT,
            server_name: str = CONTAINER) -> Run:
    size_mib = dem.stat().st_size / 1048576.0
    if mode == "manta_dem":
        r = run_child([str(MANTA), str(dem)])
        file_mib = size_mib
    elif mode == "manta_bz2":
        assert bz2 is not None
        r = run_child([str(MANTA), str(bz2)])
        file_mib = bz2.stat().st_size / 1048576.0
    elif mode == "odota_e2e":
        r = run_child([str(ODOTA), "--url", f"http://localhost:{port}", str(dem)],
                      sample_server=True, server_name=server_name)
        file_mib = size_mib
    elif mode == "odota_offline":
        r = run_child([str(ODOTA), str(ndjson)])
        file_mib = ndjson.stat().st_size / 1048576.0
    elif mode == "parser_only":
        tmp = BIN / "curl.out"
        r = run_child(["curl", "-sS", "-o", os.devnull, "-w", "%{http_code} %{size_download}",
                       "-X", "POST", "--data-binary", f"@{dem}",
                       f"http://localhost:{port}/"], stdout_path=str(tmp),
                      sample_server=True, server_name=server_name)
        try:
            code, sz = tmp.read_text().split()
            r.http_code, r.output_bytes = code, int(sz)
        except Exception:
            pass
        file_mib = size_mib
    else:
        raise ValueError(mode)
    r.replay = replay
    r.size_mib = size_mib
    r.file_mib = file_mib
    r.mode = mode
    r.run = run_idx
    r.throughput_mib_s = size_mib / r.wall_s if r.wall_s > 0 else 0.0
    return r


def median_row(rows: list[Run]) -> dict:
    walls = [r.wall_s for r in rows]
    cpus = [r.cpu_total_s for r in rows]
    rss = [max(r.child_rss_mib, r.server_peak_rss_mib) for r in rows]
    thr = [r.throughput_mib_s for r in rows]
    return {
        "replay": rows[0].replay,
        "size_mib": rows[0].size_mib,
        "mode": rows[0].mode,
        "runs": len(rows),
        "wall_med": statistics.median(walls),
        "wall_min": min(walls),
        "wall_max": max(walls),
        "cpu_med": statistics.median(cpus),
        "cpu_pct_med": statistics.median([r.cpu_pct for r in rows]),
        "rss_med": statistics.median(rss),
        "rss_max_run": max(rss),
        "throughput_med": statistics.median(thr),
        "child_rss_med": statistics.median([r.child_rss_mib for r in rows]),
        "server_peak_rss_med": statistics.median([r.server_peak_rss_mib for r in rows]),
        "server_steady_rss_med": statistics.median([r.server_steady_rss_mib for r in rows]),
    }


def write_csv(rows: list[Run]) -> None:
    cols = list(asdict(rows[0]).keys())
    with (RESULTS / "raw.csv").open("w", newline="") as f:
        w = csv.DictWriter(f, fieldnames=cols)
        w.writeheader()
        for r in rows:
            w.writerow(asdict(r))


def fmt(v: float, nd: int = 2) -> str:
    return f"{v:.{nd}f}"


def table(headers: list[str], rows: list[list[str]]) -> str:
    out = ["| " + " | ".join(headers) + " |", "|" + "|".join(["---"] * len(headers)) + "|"]
    for row in rows:
        out.append("| " + " | ".join(row) + " |")
    return "\n".join(out)


def write_summary(rows: list[Run], cold: list[dict], constrained: list[Run],
                  ndjson_sizes: dict[str, int]) -> None:
    med = {}
    for r in rows:
        med.setdefault((r.replay, r.mode), []).append(r)
    med = {k: median_row(v) for k, v in med.items()}

    lines = ["# manta-cli vs odota-cli + odota/parser", ""]
    lines += ["## Environment", "",
              f"- Host: {os.cpu_count()} logical CPUs, `{Path('/proc/cpuinfo').read_text().splitlines()[4]}`",
              f"- Replays: {len(REPLAYS)}; measured runs per mode: {RUNS} (plus {WARMUP} warm-up)",
              "- Warm page cache; parser container `odota/parser:latest`, `PARSER_WORKERS=1`",
              "- CPU/RSS for the parser are measured from the JVM process (`/proc/<pid>`), not Docker overhead",
              ""]

    lines += ["## Per-replay medians", "",
              table(
                  ["replay", "size MiB", "mode", "wall s", "CPU s", "CPU %", "peak RSS MiB", "MiB/s"],
                  [[m["replay"], fmt(m["size_mib"], 1), m["mode"], fmt(m["wall_med"]),
                    fmt(m["cpu_med"]), fmt(m["cpu_pct_med"]), fmt(m["rss_med"]),
                    fmt(m["throughput_med"])]
                   for m in sorted(med.values(), key=lambda x: (x["replay"], x["mode"]))]),
              ""]

    lines += ["## Ratios vs odota end-to-end", "",
              table(
                  ["replay", "manta wall / odota", "manta CPU / odota", "manta RSS / odota",
                   "manta vs parser wall", "offline agg / parser CPU"],
                  [[m["replay"],
                    fmt(med[(m["replay"], "manta_dem")]["wall_med"] / med[(m["replay"], "odota_e2e")]["wall_med"]),
                    fmt(med[(m["replay"], "manta_dem")]["cpu_med"] / med[(m["replay"], "odota_e2e")]["cpu_med"]),
                    fmt(med[(m["replay"], "manta_dem")]["rss_med"] / med[(m["replay"], "odota_e2e")]["rss_med"]),
                    fmt(med[(m["replay"], "manta_dem")]["wall_med"] / med[(m["replay"], "parser_only")]["wall_med"]),
                    fmt(med[(m["replay"], "odota_offline")]["cpu_med"] / med[(m["replay"], "parser_only")]["cpu_med"]),
                    ]
                   for m in sorted(med.values(), key=lambda x: x["replay"]) if m["mode"] == "manta_dem"]),
              ""]

    lines += ["## NDJSON / output sizes", "",
              table(["replay", "raw .dem MiB", "NDJSON MiB"],
                    [[rid, fmt(next(m["size_mib"] for m in med.values() if m["replay"] == rid), 1),
                      fmt(ndjson_sizes.get(rid, 0) / 1048576.0, 1)]
                     for rid, _, _ in REPLAYS]),
              ""]

    if cold:
        lines += ["## JVM cold start", "",
                  table(["replay", "container start to healthy s", "first parse wall s", "first parse CPU s", "peak RSS MiB"],
                        [[c["replay"], fmt(c["startup_s"]), fmt(c["parse_wall_s"]),
                          fmt(c["parse_cpu_s"]), fmt(c["peak_rss_mib"])] for c in cold]),
                  ""]

    if constrained:
        cm = {}
        for r in constrained:
            cm.setdefault((r.replay, r.mode), []).append(r)
        cm = {k: median_row(v) for k, v in cm.items()}
        lines += ["## Constrained parser (`--cpus=1 --memory=2g`)", "",
                  table(["replay", "mode", "wall s", "CPU s", "CPU %", "peak RSS MiB"],
                        [[m["replay"], m["mode"], fmt(m["wall_med"]), fmt(m["cpu_med"]),
                          fmt(m["cpu_pct_med"]), fmt(m["rss_med"])]
                         for m in sorted(cm.values(), key=lambda x: x["mode"])]),
                  ""]

    lines += ["## Notes", "",
              "- Warm page cache is used for all measured runs; the first cold-cache read is not isolated.",
              "- JVM RSS includes the Java heap and persists between warm runs; peak is sampled during each request.",
              "- `odota_e2e` includes odota-cli + HTTP + Java parser; `parser_only` measures the Java side.",
              "- `odota_offline` is Go aggregation from the pre-generated NDJSON, no HTTP.",
              ""]
    (RESULTS / "summary.md").write_text("\n".join(lines) + "\n")


def svg_chart(path: Path, title: str, ylabel: str, data: dict[str, dict[str, float]],
              modes: list[str], colors: dict[str, str]) -> None:
    width, height = 1100, 520
    left, right, top, bottom = 70, 30, 50, 120
    plot_w = width - left - right
    plot_h = height - top - bottom
    replays = list(data)
    maxv = max((data[r].get(m, 0) for r in replays for m in modes), default=1)
    maxv = maxv * 1.15 or 1
    group_w = plot_w / len(replays)
    bar_w = group_w * 0.8 / len(modes)
    parts = [f'<svg xmlns="http://www.w3.org/2000/svg" width="{width}" height="{height}" font-family="sans-serif">',
             f'<text x="{left}" y="28" font-size="20" font-weight="bold">{title}</text>',
             f'<text x="18" y="{top + plot_h/2}" transform="rotate(-90 18 {top + plot_h/2})" font-size="12">{ylabel}</text>']
    # grid + y labels
    for i in range(6):
        v = maxv * i / 5
        y = top + plot_h - plot_h * i / 5
        parts.append(f'<line x1="{left}" y1="{y:.1f}" x2="{left+plot_w}" y2="{y:.1f}" stroke="#ddd"/>')
        parts.append(f'<text x="{left-8}" y="{y+4:.1f}" text-anchor="end" font-size="11">{v:.1f}</text>')
    for gi, r in enumerate(replays):
        gx = left + gi * group_w
        for mi, m in enumerate(modes):
            v = data[r].get(m, 0)
            bh = plot_h * v / maxv if maxv else 0
            x = gx + group_w * 0.1 + mi * bar_w
            y = top + plot_h - bh
            parts.append(f'<rect x="{x:.1f}" y="{y:.1f}" width="{bar_w*0.9:.1f}" height="{bh:.1f}" fill="{colors.get(m, "#888")}"/>')
            parts.append(f'<text x="{x+bar_w*0.45:.1f}" y="{y-4:.1f}" text-anchor="middle" font-size="9">{v:.1f}</text>')
        parts.append(f'<text x="{gx+group_w/2:.1f}" y="{top+plot_h+18}" text-anchor="middle" font-size="11">{r}</text>')
    # legend
    lx = left
    for m in modes:
        parts.append(f'<rect x="{lx}" y="{height-50}" width="12" height="12" fill="{colors.get(m, "#888")}"/>')
        parts.append(f'<text x="{lx+18}" y="{height-39}" font-size="12">{m}</text>')
        lx += 18 + len(m) * 7 + 20
    parts.append("</svg>")
    path.write_text("\n".join(parts))


def write_charts(med: dict) -> None:
    modes = ["manta_dem", "manta_bz2", "manta_offline", "parser_only", "odota_e2e"]
    colors = {"manta_dem": "#2e7d32", "manta_bz2": "#81c784", "manta_offline": "#1565c0",
              "parser_only": "#ef6c00", "odota_e2e": "#c62828", "odota_e2e_constrained": "#ad1457",
              "parser_only_constrained": "#f9a825"}
    replays = [r[0] for r in REPLAYS]
    for metric, title, ylabel in [
        ("wall_med", "Wall time", "seconds"),
        ("cpu_med", "CPU time", "seconds"),
        ("rss_med", "Peak RSS", "MiB"),
    ]:
        data = {r: {} for r in replays}
        for (rid, mode), m in med.items():
            if mode in modes:
                data[rid][mode] = m[metric]
        svg_chart(CHARTS / f"{metric}.svg", title, ylabel, data, modes, colors)


def main() -> int:
    RESULTS.mkdir(parents=True, exist_ok=True)
    CHARTS.mkdir(parents=True, exist_ok=True)
    for p in (MANTA, ODOTA):
        if not p.exists():
            print(f"missing binary: {p}", file=sys.stderr)
            return 1

    print("[1/6] generating NDJSON", flush=True)
    nd = {}
    for rid, dem, _ in REPLAYS:
        nd[rid] = generate_ndjson(rid, dem)
    nd_sizes = {rid: p.stat().st_size for rid, p in nd.items()}

    print("[2/6] warming up", flush=True)
    for rid, dem, bz2 in REPLAYS:
        modes = ["manta_dem", "odota_e2e", "odota_offline", "parser_only"]
        if bz2:
            modes.append("manta_bz2")
        for mode in modes:
            measure(rid, dem, bz2, nd[rid], mode, 0)
            print(f"  warm {rid} {mode}", flush=True)

    print("[3/6] measuring", flush=True)
    rows: list[Run] = []
    for rep in range(1, RUNS + 1):
        for rid, dem, bz2 in REPLAYS:
            modes = ["manta_dem", "odota_e2e", "odota_offline", "parser_only"]
            if bz2:
                modes.append("manta_bz2")
            random.shuffle(modes)
            for mode in modes:
                r = measure(rid, dem, bz2, nd[rid], mode, rep)
                rows.append(r)
                print(f"  {rid} {mode} run {rep}: {r.wall_s:.2f}s cpu {r.cpu_total_s:.2f}s rss {max(r.child_rss_mib,r.server_peak_rss_mib):.0f}MiB", flush=True)

    write_csv(rows)

    print("[4/6] cold JVM runs", flush=True)
    cold = []
    for rid, dem, _ in [REPLAYS[0], REPLAYS[2]]:
        stop_container(CONTAINER)
        startup = start_container(CONTAINER, PORT)
        r = measure(rid, dem, None, nd[rid], "parser_only", 0)
        cold.append({"replay": rid, "startup_s": startup, "parse_wall_s": r.wall_s,
                     "parse_cpu_s": r.server_user_s + r.server_sys_s,
                     "peak_rss_mib": r.server_peak_rss_mib})
        print(f"  cold {rid}: start {startup:.2f}s parse {r.wall_s:.2f}s", flush=True)

    print("[5/6] constrained parser (1 CPU / 2 GB)", flush=True)
    constrained = []
    try:
        startup = start_container(CONTAINER_CONSTRAINED, PORT_CONSTRAINED,
                                  ["--cpus=1", "--memory=2g"])
        rid, dem, _ = REPLAYS[0]
        for i in range(1, 3):
            for mode in ["parser_only", "odota_e2e"]:
                r = measure(rid, dem, None, nd[rid], mode, i, port=PORT_CONSTRAINED)
                r.mode += "_constrained"
                constrained.append(r)
                print(f"  constrained {mode} run {i}: {r.wall_s:.2f}s", flush=True)
    except Exception as e:
        print(f"  constrained run failed: {e}", flush=True)
    finally:
        stop_container(CONTAINER_CONSTRAINED)

    print("[6/6] writing summary and charts", flush=True)
    med = {}
    for r in rows:
        med.setdefault((r.replay, r.mode), []).append(r)
    med = {k: median_row(v) for k, v in med.items()}
    write_summary(rows, cold, constrained, nd_sizes)
    write_charts(med)
    (RESULTS / "summary.json").write_text(json.dumps({
        "medians": {f"{k[0]}/{k[1]}": v for k, v in med.items()},
        "cold": cold,
        "constrained": [asdict(r) for r in constrained],
        "ndjson_bytes": nd_sizes,
    }, indent=2))
    print("done", flush=True)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
