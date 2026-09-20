# Catalog service performance measurements

## Index optimization comparison

See [COMPARISON.md](COMPARISON.md) for the measured before/after results. Reproduce from the repository root with `python3 docs/performance/compare.py`. The script builds the original catalog/server files from the baseline commit and the current implementation using the same load-test harness (only the reload flag initialization differs to accommodate its atomic type). It runs both versions with synthetic catalogs of 9, 1,000, and 10,000 products at 20 concurrent workers. The current implementation publishes an immutable product snapshot with an ID index; reloads rebuild and atomically replace the snapshot. Index memory grows with catalog size.

## Original baseline

This is a local service baseline, not an end-to-end website or Kubernetes capacity test. Docker was unavailable and kubectl had no current context during this measurement. No production implementation was changed or optimized for the original baseline; the later comparison above measures the ID-index optimization.

## Reproduce

From `src/productcatalogservice`:

```sh
CATALOG_PERF_OUTPUT=../../docs/performance/catalog-baseline.json go test -run '^TestPerformanceBaseline$' -count=1 -v -timeout=3m .
```

The opt-in test starts the existing product catalog implementation on a temporary loopback TCP port. It loads the actual `products.json`, uses a generated gRPC client, and validates each response against the expected product or full catalog. The test is skipped in normal test runs.

- Two operations: GetProduct (round-robin across existing IDs) and ListProducts.
- One and twenty concurrent workers, closed-loop with no think time.
- Three repetitions per operation/concurrency combination; five seconds each, after 100 sequential warmup requests per case.
- A two-second request timeout. Errors include RPC failures and incorrect responses.
- Latency includes RPC, decoding, and response validation. Percentiles use nearest-rank over all attempts, including errors.
- Client and server share a process, CPU resources, and one HTTP/2 connection. Tracing interceptors, profiling, injected latency, and catalog reload are disabled.
- No frontend, Redis, database, cluster, external network, or real user traffic is included.
- Short runs measure a baseline only, not sustained reliability or maximum capacity. Local scheduling and machine load affect results.

The JSON stores every run's request count, duration, error count, throughput, and latency percentiles, plus runtime metadata. The summary uses the median of three per-run values; its p95 is not a pooled percentile.

## Source provenance

- Baseline commit: `069c0c1bb3dd7b7adc2de005a323af1b218078df`
- products.json SHA-256: `1d15e14a04aacc92ccdcdec9a50070a3e58bdabb6f12715373fa5b0739dd8bc6`
- product_catalog.go SHA-256: `c91fa7b4620356534a06aae1534ce631aa7dd1813187778e7c44141fffa52510`

## Resume interpretation

These results support a scoped statement about building and running a reproducible local gRPC load test. They do not support claims of performance improvement, production uptime, actual user counts, or authorship of the inherited service implementation. A percentage improvement requires a measured code change under the same conditions. Zero observed failures is a test result, not a reliability guarantee.
