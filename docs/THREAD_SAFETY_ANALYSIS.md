# Thread Safety Analysis for Dynamic Labels Design

## Overview

This document analyzes the thread safety of the dynamic labels implementation, focusing on the registry pattern that maps label value combinations to metric instances.

## Current Design Components

### 1. LabelRegistry Structure

```go
type LabelRegistry[T any] struct {
    labelKeys []string
    registry  sync.Map  // map[string]*T - key is label values joined
    factory   func(labelValues []string) T
}
```

### 2. Metric Instance Thread Safety

- **Counter**: Uses `atomic.AddInt64` and `atomic.LoadInt64` - ✅ Thread-safe
- **Gauge**: Uses `atomic.StoreInt64` and `atomic.LoadInt64` - ✅ Thread-safe
- **Distribution**: Uses `sync.Mutex` for all operations - ✅ Thread-safe

## Thread Safety Analysis

### ✅ Safe: sync.Map Operations

`sync.Map` is designed for concurrent access:
- `Load()` - Safe for concurrent reads
- `Store()` - Safe for concurrent writes
- `LoadOrStore()` - Atomic operation, safe for concurrent access
- `Range()` - Safe for concurrent iteration (snapshot-like behavior)

### ⚠️ Critical: Registry Get() Implementation

The design shows:
```go
func (dc *DynamicCounter) Inc(n int64, labelValues ...string) {
    counter := dc.registry.Get(labelValues)
    counter.Add(n)
}
```

**Issue**: The `Get()` method needs to be implemented correctly to avoid race conditions. The Java library uses `ConcurrentHashMap.computeIfAbsent()` which is atomic.

**Required Implementation**:
```go
func (lr *LabelRegistry[T]) Get(labelValues []string) T {
    key := labelValuesKey(labelValues)

    // Load existing value
    if value, ok := lr.registry.Load(key); ok {
        return value.(T)
    }

    // Create new value using LoadOrStore for atomicity
    newValue := lr.factory(labelValues)
    actual, _ := lr.registry.LoadOrStore(key, newValue)
    return actual.(T)
}
```

**Potential Race Condition**: If `Get()` is implemented incorrectly (e.g., using separate `Load()` and `Store()` calls), multiple goroutines could create duplicate metric instances for the same label combination.

### ✅ Safe: Factory Function Invocation

The factory function is only called when creating a new metric instance. Even if called multiple times due to a race condition, `LoadOrStore` ensures only one instance is stored and returned.

**Note**: The factory function should be pure (no side effects) since it may be called multiple times but only one result will be used.

### ✅ Safe: Key Generation

```go
func labelValuesKey(values []string) string {
    return strings.Join(values, "\x00")  // Use null byte as separator
}
```

This is thread-safe as it:
- Only reads from the input slice
- Creates a new string (immutable)
- No shared state

### ⚠️ Consideration: Label Keys Slice

```go
type LabelRegistry[T any] struct {
    labelKeys []string  // Read-only after initialization
    ...
}
```

**Status**: ✅ Safe if `labelKeys` is never modified after `LabelRegistry` creation. The slice is only read during `Get()` operations.

**Recommendation**: Consider making it immutable or documenting that it must not be modified after creation.

### ✅ Safe: Iteration for Emission

```go
func (dc *DynamicCounter) AllCounters() iter.Seq[*Counter] {
    return func(yield func(*Counter) bool) {
        dc.registry.registry.Range(func(key, value interface{}) bool {
            if !yield(value.(*Counter)) {
                return false
            }
            return true
        })
    }
}
```

`sync.Map.Range()` is safe for concurrent access:
- Provides a snapshot-like view
- Safe even if the map is being modified during iteration
- May include or exclude entries being concurrently modified (documented behavior)

### ⚠️ Consideration: Concurrent Emission and Updates

**Scenario**: Metric values are being updated while emission is happening.

- **Counter/Gauge**: ✅ Safe - atomic operations ensure consistent reads
- **Distribution**: ✅ Safe - `GetAndClear()` uses mutex, ensures atomic read-and-clear

**Potential Issue**: If `GetAndClear()` is called during emission iteration, the distribution may be cleared while being read. However, `GetAndClear()` returns a copy, so this is safe.

### ✅ Safe: Multiple Dynamic Metrics

Each `DynamicCounter`, `DynamicGauge`, and `DynamicDistribution` has its own `LabelRegistry` instance. There's no shared state between different dynamic metrics, so concurrent access to different metrics is safe.

## Comparison with Java Implementation

### Java Library Thread Safety

1. **Registry**: Uses `ConcurrentHashMap.computeIfAbsent()` - atomic operation
2. **Aggregators**:
   - Counters: Use atomic operations or partitioned aggregators
   - Distributions: Use synchronized blocks
3. **Iteration**: `ConcurrentHashMap` iteration is safe for concurrent access

### Go Implementation Equivalency

| Java | Go | Thread Safety |
|------|-----|---------------|
| `ConcurrentHashMap.computeIfAbsent()` | `sync.Map.LoadOrStore()` | ✅ Equivalent |
| Atomic operations | `atomic` package | ✅ Equivalent |
| `synchronized` blocks | `sync.Mutex` | ✅ Equivalent |
| Concurrent iteration | `sync.Map.Range()` | ✅ Equivalent |

## Recommendations

### 1. ✅ Implement Get() Correctly

**Critical**: The `Get()` method must use `LoadOrStore()` to ensure atomicity:

```go
func (lr *LabelRegistry[T]) Get(labelValues []string) T {
    key := labelValuesKey(labelValues)

    // Try to load existing
    if value, ok := lr.registry.Load(key); ok {
        return value.(T)
    }

    // Atomically create and store if absent
    newValue := lr.factory(labelValues)
    actual, _ := lr.registry.LoadOrStore(key, newValue)
    return actual.(T)
}
```

### 2. ✅ Document Factory Function Requirements

The factory function should be:
- **Pure**: No side effects (may be called multiple times)
- **Idempotent**: Same inputs produce equivalent outputs
- **Fast**: Called during hot path (metric updates)

### 3. ✅ Make labelKeys Immutable

Consider:
```go
type LabelRegistry[T any] struct {
    labelKeys []string  // Immutable after creation - DO NOT MODIFY
    registry  sync.Map
    factory   func(labelValues []string) T
}
```

Or use a read-only pattern/documentation.

### 4. ✅ Consider Adding Tests

Add concurrent tests:
- Multiple goroutines updating the same label combination
- Concurrent emission and updates
- Stress tests with many label combinations

### 5. ⚠️ Optional: Add Metrics for Debugging

Consider tracking:
- Number of times factory is called vs. number of stored instances (to detect races)
- Registry size over time

## Summary

| Component | Thread Safety | Notes |
|-----------|--------------|-------|
| `sync.Map` registry | ✅ Safe | Properly used with `LoadOrStore()` |
| Counter operations | ✅ Safe | Atomic operations |
| Gauge operations | ✅ Safe | Atomic operations |
| Distribution operations | ✅ Safe | Mutex-protected |
| Key generation | ✅ Safe | No shared state |
| Iteration for emission | ✅ Safe | `sync.Map.Range()` is concurrent-safe |
| Factory function | ✅ Safe | Pure function, may be called multiple times |
| **Get() implementation** | ⚠️ **Must use LoadOrStore()** | **Critical for correctness** |

## Conclusion

The design is **thread-safe** provided that:

1. ✅ `Get()` method uses `sync.Map.LoadOrStore()` for atomic get-or-create
2. ✅ Factory functions are pure (no side effects)
3. ✅ `labelKeys` slice is never modified after registry creation
4. ✅ Metric instances (Counter/Gauge/Distribution) maintain their thread-safety guarantees

The main risk is an incorrect `Get()` implementation that doesn't use atomic operations. With proper implementation using `LoadOrStore()`, the design matches the thread-safety guarantees of the Java library.
