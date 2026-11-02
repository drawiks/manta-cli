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
| Wall time, 80–182 MiB | **1.9–3.8 с** | **3.5–7.7 с** |
| CPU time | **2.9–5.7 с** | **5.5–11.9 с** |
| Peak RSS | **38–45 MiB** | **~815 MiB** (warm JVM) |
| Пропускная способность | **41–48 MiB/s** | **22–24 MiB/s** |
| Отношение wall | — | manta ≈ **0.45–0.56×** от odota e2e |
| Отношение CPU | — | manta ≈ **0.45–0.54×** от odota e2e |
| Отношение RSS | — | manta ≈ **5%** от odota e2e |

Иными словами, на этом хосте `manta-cli` примерно в **1.8–2.2 раза быстрее**
по wall time, в **~2 раза** меньше по CPU и в **~20 раз** меньше по памяти.

### Где уходит время у связки

- `parser_only` и `odota_e2e` очень близки по wall time: Go-агрегация NDJSON
  дешёвая, доминирует Java-парсер и HTTP/JSON.
- `odota_offline` (только Go-агрегация) — **0.58–1.32 с** и **13–14% CPU**
  от `parser_only`. То есть основной расход — не агрегация, а разбор и
  передача NDJSON.
- Размер NDJSON — **48–111 MiB**, примерно 59% от размера `.dem`.

### Холодный JVM

| Реплей | контейнер до healthy | первый parse | CPU первого parse | peak RSS |
|---|---:|---:|---:|---:|
| 9064911131 (80 MiB) | 0.71 с | 6.14 с | 11.58 с | 422 MiB |
| 9048749977 (182 MiB) | 0.50 с | 10.05 с | 14.90 с | 596 MiB |

Первый parse заметно дороже тёплого: JIT, загрузка классов и прогрев кучи.
Для повторных разборов JVM остаётся тёплым, но и RSS остаётся высоким.

### Ограничение ресурсов (`--cpus=1 --memory=2g`)

На реплее 9064911131:

| Режим | wall | CPU | CPU % | peak RSS |
|---|---:|---:|---:|---:|
| `parser_only_constrained` | 3.83 с | 4.21 с | 110% | 224 MiB |
| `odota_e2e_constrained` | 3.61 с | 5.28 с | 146% | 224 MiB |

Ограничение памяти до 2 GiB сильно снижает RSS JVM (815 → 224 MiB) и на этом
реплее почти не ухудшает wall time. Это важно для деплоя: JVM можно держать в
жёстком memory limit, но cold/JIT-поведение всё равно остаётся дорогим.

### Сжатый вход

`manta-cli` умеет читать `.dem.bz2` напрямую. На трёх реплеях:

| Реплей | raw `.dem` | `.dem.bz2` |
|---|---:|---:|
| 9064911131 | 1.93 с | 1.96 с |
| 9064082159 | 2.21 с | 2.30 с |
| 9048749977 | 3.80 с | 4.02 с |

Декомпрессия bzip2 добавляет немного: основной расход — сам разбор manta.
`odota-cli` в текущем виде отправляет в парсер сырой `.dem`, поэтому сжатый
вход для него — отдельная распаковка снаружи.

## Выводы

- **Скорость:** `manta-cli` быстрее в ~2 раза по wall и CPU на всех реплеях.
- **Память:** разница радикальная: десятки MiB против сотен MiB у JVM.
- **Основной расход odota-связки** — Java-парсер и NDJSON, а не Go-агрегация.
- **Цена JVM** — высокий RSS и заметный cold-start/JIT; constrained-режим
  снижает память, но не убирает JVM-оверхед.
- **Функционально** `manta-cli` уже отдаёт тот же JSON; замер показывает, что
  замена связки даёт кратный выигрыш по ресурсам на этом сценарии.

## Воспроизведение

```bash
# бинарники
go build -trimpath -ldflags="-s -w" -o /tmp/bench/manta_cli .
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
