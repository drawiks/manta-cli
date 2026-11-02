# manta-cli vs odota-cli + odota/parser

## Environment

- Host: 4 logical CPUs, `model name	: 12th Gen Intel(R) Core(TM) i3-1215U`
- Replays: 5; measured runs per mode: 3 (plus 1 warm-up)
- Warm page cache; parser container `odota/parser:latest`, `PARSER_WORKERS=1`
- CPU/RSS for the parser are measured from the JVM process (`/proc/<pid>`), not Docker overhead

## Per-replay medians

| replay | size MiB | mode | wall s | CPU s | CPU % | peak RSS MiB | MiB/s |
|---|---|---|---|---|---|---|---|
| 9048749977 | 182.0 | manta_bz2 | 4.02 | 5.71 | 141.38 | 44.55 | 45.29 |
| 9048749977 | 182.0 | manta_dem | 3.80 | 5.72 | 150.67 | 38.54 | 47.94 |
| 9048749977 | 182.0 | odota_e2e | 7.72 | 11.89 | 154.29 | 814.81 | 23.57 |
| 9048749977 | 182.0 | odota_offline | 1.32 | 1.34 | 101.73 | 16.64 | 138.33 |
| 9048749977 | 182.0 | parser_only | 8.21 | 9.65 | 116.79 | 814.81 | 22.17 |
| 9053590215 | 133.5 | manta_dem | 2.88 | 4.36 | 151.58 | 38.59 | 46.30 |
| 9053590215 | 133.5 | odota_e2e | 5.86 | 8.83 | 151.88 | 814.81 | 22.77 |
| 9053590215 | 133.5 | odota_offline | 0.91 | 0.92 | 101.82 | 16.64 | 146.97 |
| 9053590215 | 133.5 | parser_only | 5.95 | 6.87 | 114.75 | 814.81 | 22.44 |
| 9064082159 | 101.3 | manta_bz2 | 2.30 | 3.26 | 141.59 | 45.33 | 44.03 |
| 9064082159 | 101.3 | manta_dem | 2.21 | 3.35 | 150.98 | 39.04 | 45.85 |
| 9064082159 | 101.3 | odota_e2e | 4.51 | 6.93 | 153.75 | 814.81 | 22.45 |
| 9064082159 | 101.3 | odota_offline | 0.76 | 0.77 | 101.60 | 16.64 | 132.69 |
| 9064082159 | 101.3 | parser_only | 4.60 | 5.34 | 116.98 | 814.81 | 22.01 |
| 9064911131 | 80.0 | manta_bz2 | 1.96 | 2.83 | 145.64 | 45.02 | 40.85 |
| 9064911131 | 80.0 | manta_dem | 1.93 | 2.94 | 152.50 | 38.75 | 41.56 |
| 9064911131 | 80.0 | odota_e2e | 3.46 | 5.48 | 155.36 | 814.81 | 23.13 |
| 9064911131 | 80.0 | odota_offline | 0.58 | 0.58 | 101.11 | 16.64 | 137.46 |
| 9064911131 | 80.0 | parser_only | 3.52 | 4.14 | 117.06 | 814.81 | 22.76 |
| 9065268582 | 119.6 | manta_dem | 2.48 | 3.71 | 149.61 | 37.52 | 48.27 |
| 9065268582 | 119.6 | odota_e2e | 5.48 | 8.27 | 151.78 | 814.81 | 21.83 |
| 9065268582 | 119.6 | odota_offline | 0.86 | 0.88 | 101.51 | 16.64 | 138.65 |
| 9065268582 | 119.6 | parser_only | 5.43 | 6.33 | 116.55 | 814.81 | 22.02 |

## Ratios vs odota end-to-end

| replay | manta wall / odota | manta CPU / odota | manta RSS / odota | manta vs parser wall | offline agg / parser CPU |
|---|---|---|---|---|---|
| 9048749977 | 0.49 | 0.48 | 0.05 | 0.46 | 0.14 |
| 9053590215 | 0.49 | 0.49 | 0.05 | 0.48 | 0.13 |
| 9064082159 | 0.49 | 0.48 | 0.05 | 0.48 | 0.14 |
| 9064911131 | 0.56 | 0.54 | 0.05 | 0.55 | 0.14 |
| 9065268582 | 0.45 | 0.45 | 0.05 | 0.46 | 0.14 |

## NDJSON / output sizes

| replay | raw .dem MiB | NDJSON MiB |
|---|---|---|
| 9064911131 | 80.0 | 47.6 |
| 9064082159 | 101.3 | 62.1 |
| 9048749977 | 182.0 | 110.9 |
| 9053590215 | 133.5 | 75.1 |
| 9065268582 | 119.6 | 71.1 |

## JVM cold start

| replay | container start to healthy s | first parse wall s | first parse CPU s | peak RSS MiB |
|---|---|---|---|---|
| 9064911131 | 0.71 | 6.14 | 11.58 | 421.58 |
| 9048749977 | 0.50 | 10.05 | 14.90 | 595.97 |

## Constrained parser (`--cpus=1 --memory=2g`)

| replay | mode | wall s | CPU s | CPU % | peak RSS MiB |
|---|---|---|---|---|---|
| 9064911131 | odota_e2e_constrained | 3.61 | 5.28 | 145.61 | 223.88 |
| 9064911131 | parser_only_constrained | 3.83 | 4.21 | 109.76 | 223.88 |

## Notes

- Warm page cache is used for all measured runs; the first cold-cache read is not isolated.
- JVM RSS includes the Java heap and persists between warm runs; peak is sampled during each request.
- `odota_e2e` includes odota-cli + HTTP + Java parser; `parser_only` measures the Java side.
- `odota_offline` is Go aggregation from the pre-generated NDJSON, no HTTP.

