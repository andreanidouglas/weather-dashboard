# Cache strategy benchmark

Run from the repository root:

```sh
go run ./benchmark
```

The executable generates reproducible random workloads of 10,000 requests
across 1,000 cities and reuses each workload for both cache strategies. It
compares repeated city names, aliases sharing 100 coordinate buckets, and
coordinate requests with sub-cent jitter. Pass `-seed` to generate different
request sequences.

It calls the `/api/weather` and `/api/pos` handlers directly to avoid the
production rate limiter and uses a local mock OpenWeatherMap endpoint, so it
needs no API key or external network access. Cache hits are counted as requests
that did not reach the mock upstream; cache-entry counts show coordinate-key
deduplication separately from hit rate.
