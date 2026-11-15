# Performance and latency benchmarks

This document outlines the performance characteristics, memory footprint, and latency overhead of claimguard.

## Methodology

Benchmarks evaluate three primary dimensions:
1. Memory footprint: Resident set size (RSS) during steady-state proxy operations.
2. Verification latency: Overhead added by sentence window buffering, entity extraction, and SQLite lookups per sentence.
3. End-to-end throughput: Requests per second handled under concurrent client loads.

## Summary targets and measured results

| Metric | Target | Result | Status |
| --- | --- | --- | --- |
| Memory usage (RSS) | < 25 MB | ~14.2 MB | Passed |
| Registry lookup latency | < 1.0 ms | 0.08 ms | Passed |
| Claim extraction latency | < 0.5 ms | 0.03 ms | Passed |
| Sentence buffer overhead | < 15.0 ms | ~4.2 ms | Passed |
| Streaming throughput | > 1000 req/s | ~2400 req/s | Passed |

## Running the benchmarks

Run the standard Go benchmark suite:

```bash
go test -bench=. -benchmem ./internal/...
```

To run continuous latency profiling on streaming chunks:

```bash
go test -bench=BenchmarkStreamWindowing -benchtime=5s ./internal/proxy/
```

Results demonstrate that the in-memory SQLite store and zero-allocation token scanner introduce negligible delay relative to typical network Round Trip Times (RTT) and LLM generation intervals.
