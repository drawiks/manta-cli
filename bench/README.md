# Замер ресурсов: manta-cli vs odota-cli + odota/parser

Замер сравнивает два пути разбора одного и того же реплея:

- `manta-cli` — один Go-бинарник (manta 1.5.2);
- `odota-cli` + `odota/parser` — Go CLI, HTTP и Java-парсер в Docker
  (`odota/parser:latest`, JVM 21, clarity).

## Что и как измерялось

- 5 современных реплеев (80–182 MiB) с эталонными `.out.json`.
- 1 прогревочный прогон на режим/реплей, затем 3 измеряемых; в таблицах медианы.
- Тёплый page cache; парсер запущен с `PARSER_WORKERS=1`.
- Wall time и CPU — из `rusage` дочернего процесса и `/proc/<pid>` JVM.
- Peak RSS:
  - `manta-cli` — `ru_maxrss`;
  - JVM — максимум `VmRSS` во время запроса + `VmHWM` (пик процесса).
- CPU парсера — дельта `utime+stime` JVM-процесса; для `odota_e2e` суммируется
  CPU CLI и сервера.

Режимы:

| режим | что означает |
|---|---|
| `manta_dem` | manta-cli, сырой `.dem` |
| `manta_bz2` | manta-cli, `.dem.bz2` (3 реплея) |
| `parser_only` | Java-парсер через HTTP, без Go-агрегации |
| `odota_offline` | только Go-агрегация готового NDJSON |
| `odota_e2e` | odota-cli + HTTP + Java-парсер |
| `*_constrained` | парсер с `--cpus=1 --memory=2g` |

## Ключевые результаты

| Метрика | manta-cli | odota-cli + parser |
|---|---|---|
| Wall time, 80–182 MiB | **2.9–5.9 с** | **4.5–10.0 с** |
| CPU time | **4.6–9.3 с** | **7.0–15.6 с** |
| Peak RSS | **40–47 MiB** | **~859 MiB** (warm JVM) |
| Пропускная способность | **26–32 MiB/s** | **17–18 MiB/s** |
| Отношение wall | — | manta ≈ **0.57–0.65×** от odota e2e |
| Отношение CPU | — | manta ≈ **0.56–0.66×** от odota e2e |
| Отношение RSS | — | manta ≈ **5%** от odota e2e |

Иными словами, на этом хосте `manta-cli` примерно в **1.5–1.8 раза быстрее**
по wall time, в **~1.7 раза** меньше по CPU и в **~20 раз** меньше по памяти.

### Где уходит время у связки

- `parser_only` и `odota_e2e` очень близки по wall time: Go-агрегация NDJSON
  дешёвая, доминирует Java-парсер и HTTP/JSON.
- `odota_offline` (только Go-агрегация) — **0.89–1.57 с** и **13–22% CPU**
  от `parser_only`. То есть основной расход — не агрегация, а разбор и
  передача NDJSON.
- Размер NDJSON — **48–111 MiB**, примерно 59% от размера `.dem`.

### Холодный JVM

| Реплей | контейнер до healthy | первый parse | CPU первого parse | peak RSS |
|---|---:|---:|---:|---:|
| 9064911131 (80 MiB) | 1.59 с | 6.56 с | 12.86 с | 434 MiB |
| 9048749977 (182 MiB) | 0.50 с | 11.93 с | 16.92 с | 571 MiB |

Первый parse заметно дороже тёплого: JIT, загрузка классов и прогрев кучи.
Для повторных разборов JVM остаётся тёплым, но и RSS остаётся высоким.

### Ограничение ресурсов (`--cpus=1 --memory=2g`)

На реплее 9064911131:

| Режим | wall | CPU | CPU % | peak RSS |
|---|---:|---:|---:|---:|
| `parser_only_constrained` | 4.77 с | 5.32 с | 111% | 221 MiB |
| `odota_e2e_constrained` | 3.78 с | 5.59 с | 146% | 221 MiB |

Ограничение памяти до 2 GiB сильно снижает RSS JVM (859 → 221 MiB) и на этом
реплее почти не ухудшает wall time. Это важно для деплоя: JVM можно держать в
жёстком memory limit, но cold/JIT-поведение всё равно остаётся дорогим.

### Сжатый вход

`manta-cli` умеет читать `.dem.bz2` напрямую. На трёх реплеях:

| Реплей | raw `.dem` | `.dem.bz2` |
|---|---:|---:|
| 9064911131 | 2.90 с | 3.08 с |
| 9064082159 | 3.20 с | 3.35 с |
| 9048749977 | 5.90 с | 5.84 с |

Декомпрессия bzip2 добавляет немного: основной расход — сам разбор manta.
`odota-cli` в текущем виде отправляет в парсер сырой `.dem`, поэтому сжатый
вход для него — отдельная распаковка снаружи.

## Выводы

- **Скорость:** `manta-cli` быстрее в ~1.5–1.8 раза по wall и CPU на всех реплеях.
- **Память:** разница радикальная: десятки MiB против сотен MiB у JVM.
- **Основной расход odota-связки** — Java-парсер и NDJSON, а не Go-агрегация.
- **Цена JVM** — высокий RSS и заметный cold-start/JIT; constrained-режим
  снижает память, но не убирает JVM-оверхед.
- **Функционально** `manta-cli` уже отдаёт тот же JSON; замер показывает, что
  замена связки даёт кратный выигрыш по ресурсам на этом сценарии.

## Воспроизведение

```bash
# бинарники
go build -trimpath -ldflags="-s -w" -o /tmp/bench/manta_cli ./cmd/manta-cli
(cd ~/odota-cli && GOTOOLCHAIN=auto go build -trimpath -ldflags="-s -w" -o /tmp/bench/odota_cli .)

# парсер
docker run -d --rm --name parser-bench -p 5600:5600 \
  -e PARSER_WORKERS=1 odota/parser:latest
curl --retry 30 --retry-delay 1 -f http://localhost:5600/healthz

# замер
python3 bench/run.py
```

Сырые данные: `bench/results/raw.csv`, `bench/results/summary.json`,
`bench/results/summary.md`, графики — `bench/results/charts/*.svg`.

## Оговорки

- Page cache тёплый; первый холодный доступ к файлу отдельно не изолировался.
- JVM RSS — это пик процесса, а не «полезная» память; GC может не возвращать
  память ОС.
- Абсолютные секунды зависят от хоста (4 ядра, 5.8 GiB RAM); важны отношения
  между вариантами.
- Образ `odota/parser:latest` проверен на одном реплее: NDJSON и итоговый JSON
  совпали с эталоном `odota-cli`.
