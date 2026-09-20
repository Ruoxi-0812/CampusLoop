# Measured results

Timestamp: 2026-09-20T03:40:40Z. Runtime: go1.25.0, darwin/arm64; 11 logical CPUs; GOMAXPROCS=11; 9 products.

| Operation | Workers | Median requests/s | Median p95 (ms) | Observed errors |
|---|---:|---:|---:|---:|
| GetProduct | 1 | 20,979 | 0.058 | 0 / 316,931 |
| GetProduct | 20 | 168,773 | 0.245 | 0 / 2,484,242 |
| ListProducts | 1 | 16,558 | 0.070 | 0 / 248,452 |
| ListProducts | 20 | 121,246 | 0.329 | 0 / 1,799,294 |

Total measured requests: 4,848,919. Errors: 0. Warmup requests excluded.

See README.md for scope and limitations. These are baseline measurements, not percentage improvements.
