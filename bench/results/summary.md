# manta-cli vs odota-cli + odota/parser

## Environment

- Host: 4 logical CPUs, `model name	: 12th Gen Intel(R) Core(TM) i3-1215U`
- Replays: 5; measured runs per mode: 3 (plus 1 warm-up)
- Warm page cache; parser container `odota/parser:latest`, `PARSER_WORKERS=1`
- CPU/RSS for the parser are measured from the JVM process (`/proc/<pid>`), not Docker overhead

## Per-replay medians

| replay | size MiB | mode | wall s | CPU s | CPU % | peak RSS MiB | MiB/s |
|---|---|---|---|---|---|---|---|
| 9048749977 | 182.0 | manta_bz2 | 5.84 | 8.74 | 149.60 | 46.72 | 31.15 |
| 9048749977 | 182.0 | manta_dem | 5.90 | 9.29 | 157.38 | 40.62 | 30.84 |
| 9048749977 | 182.0 | odota_e2e | 9.96 | 15.60 | 155.20 | 859.18 | 18.27 |
| 9048749977 | 182.0 | odota_offline | 1.57 | 1.61 | 102.04 | 16.75 | 115.62 |
| 9048749977 | 182.0 | parser_only | 10.36 | 11.88 | 114.61 | 859.18 | 17.56 |
| 9053590215 | 133.5 | manta_dem | 4.41 | 6.93 | 157.23 | 40.77 | 30.30 |
| 9053590215 | 133.5 | odota_e2e | 7.43 | 11.35 | 153.35 | 859.18 | 17.97 |
| 9053590215 | 133.5 | odota_offline | 1.05 | 1.07 | 102.87 | 16.75 | 127.44 |
| 9053590215 | 133.5 | parser_only | 7.08 | 8.08 | 114.16 | 859.18 | 18.86 |
| 9064082159 | 101.3 | manta_bz2 | 3.35 | 4.99 | 148.92 | 46.79 | 30.21 |
| 9064082159 | 101.3 | manta_dem | 3.20 | 5.04 | 156.75 | 40.71 | 31.63 |
| 9064082159 | 101.3 | odota_e2e | 5.60 | 8.96 | 157.56 | 859.18 | 18.08 |
| 9064082159 | 101.3 | odota_offline | 0.89 | 0.91 | 102.17 | 16.75 | 113.97 |
| 9064082159 | 101.3 | parser_only | 5.52 | 6.47 | 116.90 | 859.18 | 18.34 |
| 9064911131 | 80.0 | manta_bz2 | 3.08 | 4.33 | 148.17 | 46.86 | 25.94 |
| 9064911131 | 80.0 | manta_dem | 2.90 | 4.64 | 157.52 | 40.47 | 27.55 |
| 9064911131 | 80.0 | odota_e2e | 4.45 | 7.00 | 157.14 | 859.18 | 17.97 |
| 9064911131 | 80.0 | odota_offline | 1.18 | 1.20 | 102.09 | 16.75 | 67.75 |
| 9064911131 | 80.0 | parser_only | 4.97 | 5.49 | 116.40 | 859.18 | 16.11 |
| 9065268582 | 119.6 | manta_dem | 3.98 | 6.22 | 155.48 | 39.86 | 30.04 |
| 9065268582 | 119.6 | odota_e2e | 6.91 | 10.69 | 154.67 | 859.18 | 17.29 |
| 9065268582 | 119.6 | odota_offline | 1.36 | 1.40 | 102.50 | 16.75 | 87.85 |
| 9065268582 | 119.6 | parser_only | 8.08 | 9.16 | 115.00 | 859.18 | 14.80 |

## Ratios vs odota end-to-end

| replay | manta wall / odota | manta CPU / odota | manta RSS / odota | manta vs parser wall | offline agg / parser CPU |
|---|---|---|---|---|---|
| 9048749977 | 0.59 | 0.60 | 0.05 | 0.57 | 0.14 |
| 9053590215 | 0.59 | 0.61 | 0.05 | 0.62 | 0.13 |
| 9064082159 | 0.57 | 0.56 | 0.05 | 0.58 | 0.14 |
| 9064911131 | 0.65 | 0.66 | 0.05 | 0.58 | 0.22 |
| 9065268582 | 0.58 | 0.58 | 0.05 | 0.49 | 0.15 |

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
| 9064911131 | 1.59 | 6.56 | 12.86 | 434.22 |
| 9048749977 | 0.50 | 11.93 | 16.92 | 570.97 |

## Constrained parser (`--cpus=1 --memory=2g`)

| replay | mode | wall s | CPU s | CPU % | peak RSS MiB |
|---|---|---|---|---|---|
| 9064911131 | odota_e2e_constrained | 3.78 | 5.59 | 146.24 | 220.75 |
| 9064911131 | parser_only_constrained | 4.77 | 5.32 | 111.45 | 220.75 |

## Notes

- Warm page cache is used for all measured runs; the first cold-cache read is not isolated.
- JVM RSS includes the Java heap and persists between warm runs; peak is sampled during each request.
- `odota_e2e` includes odota-cli + HTTP + Java parser; `parser_only` measures the Java side.
- `odota_offline` is Go aggregation from the pre-generated NDJSON, no HTTP.

