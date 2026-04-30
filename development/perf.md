# Performance notes

## Parallel resolver execution

**Commit**: parallel resolver execution via bounded goroutine pool.

### Design

- Threshold: `len(rules) >= 8 && runtime.NumCPU() > 1` → parallel
- Concurrency: bounded by `runtime.NumCPU()` via semaphore channel
- Determinism: results collected into per-rule slots, merged in rule-ID order
- Panic recovery: per-goroutine, same as serial path

### Benchmark results (Apple M1 Max, 10 cores)

```
BenchmarkEvaluate_Serial-10        1000   ~1.7 µs/op   (4 trivial rules)
BenchmarkEvaluate_Parallel-10      1000  ~32.0 µs/op   (16 trivial rules)
```

**Interpretation**: For trivial resolvers (no-op functions), goroutine overhead
dominates and serial is ~20x faster. The parallel path benefits repos with:

- 8+ rules
- Resolvers that do real file scanning (ms-scale work)
- Large repos (50k+ files where each resolver walks the file list)

For archfit's own scan (17 rules, ~200 files), the total evaluation time is
<5ms either way — dominated by collector I/O, not resolver execution.

### Race safety

`TestEvaluate_ParallelDeterminism` runs 100 iterations with `-race` and
asserts byte-identical JSON output. This is enforced in CI.

### When to revisit

The threshold (8 rules) and concurrency limit (`NumCPU()`) are conservative.
If archfit grows to 50+ rules or resolvers begin doing heavier analysis
(AST parsing, content scanning), lower the threshold or increase the pool.
