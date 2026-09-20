# Product lookup optimization results

Local loopback gRPC; 20 concurrent workers; synthetic catalogs cloned from actual product fields. Three 5-second measured repetitions per variant/size, 100 warmup requests per repetition. Values below are medians of per-run values.

| Products | Before p95 ms | After p95 ms | p95 reduction | Before requests/s | After requests/s | Throughput increase |
|---:|---:|---:|---:|---:|---:|---:|
| 9 | 0.253709 | 0.252375 | 0.53% | 163,073 | 163,856 | 0.48% |
| 1,000 | 0.300458 | 0.259334 | 13.69% | 134,024 | 160,114 | 19.47% |
| 10,000 | 0.576167 | 0.300458 | 47.85% | 63,956 | 151,880 | 137.48% |

Reductions are (before − after) / before; throughput increases are (after − before) / before. Negative values mean regression. Raw per-run results and execution order are saved alongside this report.

Scope: single process hosting both client and server, one shared HTTP/2 connection, in-memory catalog, no tracing interceptors, database, frontend, Kubernetes, or external network. Each request validates the returned product. Index construction is included in warmup rather than steady-state latency. This tests GetProduct only and does not establish whole-site speed or production capacity. Each variant is run as a block of three repetitions; time/order effects and local scheduling remain possible.

The comparison completed 12,528,859 measured requests with no observed RPC or response-validation errors. All sizes are reported; the 9-item result is effectively unchanged and should not be represented as a meaningful improvement.

The resume bullet in `resume-bullet.tex` rounds 47.85% and 137.48% to whole percentages and explicitly scopes these results to local load tests with synthetic data. The added index uses O(n) memory and O(n) construction time to provide expected O(1) steady-state ID lookup.

Validation: `go test -race -count=1 ./...` passed for the product catalog service, including duplicate/missing IDs, concurrent initialization and reads, concurrent reload, stale-ID removal, and recovery after invalid catalog JSON.
