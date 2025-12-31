# Dynamic Labels Design

## Overview

This document describes the design for implementing dynamic labels in the Go `go-gcp-metrics` library, based on the Java `gcp-metrics` library implementation.

## Key Concepts: Label Keys vs Label Values

The design distinguishes between:
- **Label Keys**: Defined when creating a metric (e.g., `"status"`, `"method"`)
- **Label Values**: Provided when recording values (e.g., `"200"`, `"GET"`)

This separation enables a single metric instance to handle multiple label combinations dynamically.

## Java Library Implementation

### Architecture

The Java library uses a **registry pattern** to manage dynamic labels:

1. **LabelAggregatorWriterRegistry**: Interface that maps label value combinations to aggregators
   - `PerLabelAggregatorWriterRegistry`: For metrics with labels (uses `ConcurrentHashMap`)
   - `SingleAggregatorWriterRegistry`: For metrics without labels (optimization)

2. **Metric Creation Flow**:
   ```java
   // Define label keys when creating the metric
   Counter counter = metrics.counter("api_requests", "", "status", "endpoint");

   // Provide label values when using the metric
   counter.inc(1, "200", "/api/users");
   counter.inc(1, "404", "/api/users");
   counter.inc(1, "200", "/api/posts");
   ```

3. **Internal Mechanism**:
   - Each metric stores the **label keys** (e.g., `["status", "endpoint"]`)
   - When `inc(n, "200", "/api/users")` is called:
     - The registry looks up or creates an aggregator for the label combination `["200", "/api/users"]`
     - Each unique label combination gets its own aggregator instance
     - The aggregator is registered with the emitter for that specific label combination

### Key Components

#### 1. LabelAggregatorWriterRegistry Interface
```java
public sealed interface LabelAggregatorWriterRegistry<T>
        permits SingleAggregatorWriterRegistry, PerLabelAggregatorWriterRegistry {
    T getAggregatorForLabelValue(String... labelValue);
}
```

#### 2. PerLabelAggregatorWriterRegistry
- Maintains a `ConcurrentHashMap<ImmutableList<String>, T>` mapping label values to aggregators
- Uses `computeIfAbsent` to lazily create aggregators for new label combinations
- Each aggregator is registered with the emitter when first created

#### 3. Metric Classes (GCloudCounter, GCloudGauge, GCloudDistribution)
- Store `labelKey` (ImmutableList<String>) - the keys defined at creation
- Store `aggregators` (LabelAggregatorWriterRegistry) - the registry for getting aggregators
- Methods accept `labelValue` (String...) - the values provided at runtime

### Example Usage Pattern

```java
// Create a counter with label keys
Counter counter = metrics.counter("http_requests", "", "status", "method");

// Use with different label combinations
counter.inc(1, "200", "GET");      // Creates aggregator for ["200", "GET"]
counter.inc(1, "404", "GET");      // Creates aggregator for ["404", "GET"]
counter.inc(1, "200", "POST");     // Creates aggregator for ["200", "POST"]
counter.inc(1, "200", "GET");      // Reuses aggregator for ["200", "GET"]
```

## Current Go Library Implementation

### Static Labels Approach

The Go library currently uses **static labels**:

```go
// Labels are fixed at creation time
counter := metrics.Counter("api_requests", map[string]string{
    "env": "prod",
    "region": "us-east",
})

// Labels cannot be changed when incrementing
counter.Inc()  // Always uses the same labels: env=prod, region=us-east
```

### Limitations

1. **No Dynamic Labels**: Each metric instance has one fixed set of labels
2. **Multiple Instances Required**: To track different label combinations, you must create separate metric instances
3. **Memory Overhead**: Each combination requires a separate metric object

### Current Structure

```go
type Counter struct {
    Name   string
    Labels map[string]string  // Fixed at creation
    value  int64
}

func (c *Counter) Inc() {
    atomic.AddInt64(&c.value, 1)
}
```

## Proposed Implementation for Go

### Design Approach

To implement dynamic labels in Go, we should:

1. **Separate Label Keys from Label Values**
   - Store label keys (ordered slice) in the metric
   - Accept label values as parameters in methods

2. **Use a Registry Pattern**
   - Create a registry that maps label value combinations to metric instances
   - Use `sync.Map` for thread-safe concurrent access

3. **Maintain Backward Compatibility**
   - Keep the existing API for static labels
   - Add new methods/functions for dynamic labels

### Proposed API Design

#### Using Variadic Arguments
```go
// Static labels (existing API - unchanged)
counter := metrics.Counter("api_requests", map[string]string{"env": "prod"})
counter.Inc()

// Dynamic labels (new API using variadic arguments for label keys)
counter := metrics.CounterWithLabels("api_requests", "status", "endpoint")
counter.Inc(1, "200", "/api/users")
counter.Inc(1, "404", "/api/users")
counter.Inc(1, "200", "/api/posts")

// Similar for gauges and distributions
gauge := metrics.GaugeWithLabels("temperature", "location", "sensor")
gauge.Set(25, "datacenter1", "sensor1")

distribution := metrics.DistributionWithLabels("latency", "ms", 100, 10, "endpoint", "method")
distribution.Update(150, "/api/users", "GET")
```

This approach:
- Uses Go variadic arguments (`...string`) for label keys, matching idiomatic Go style
- Mirrors the Java library's API pattern: `counter(String name, String unit, String... label)`
- Keeps the API clean and intuitive
- Maintains backward compatibility with existing static label API

### Implementation Structure

```go
// labelValuesToMap converts label keys and values to a map[string]string.
// This function handles mismatched lengths permissively:
// - If fewer values than keys: only the first N keys get values (where N = len(values))
// - If more values than keys: only the first N values are used (where N = len(keys))
// This matches the Java library's permissive behavior.
func labelValuesToMap(keys []string, values []string) map[string]string {
    result := make(map[string]string)
    for i, key := range keys {
        if i < len(values) {
            result[key] = values[i]
        }
        // If i >= len(values), key is left without a value (permissive behavior)
    }
    return result
}

// Label registry for managing dynamic label combinations
type LabelRegistry[T any] struct {
    labelKeys []string  // Immutable after creation - DO NOT MODIFY
    registry  sync.Map  // map[string]*T - key is label values joined
    factory   func(labelValues []string) T
}

// Get retrieves or creates a metric instance for the given label values.
// This method is thread-safe and uses atomic operations to ensure only one
// instance is created per unique label combination, even under concurrent access.
func (lr *LabelRegistry[T]) Get(labelValues []string) T {
    key := labelValuesKey(labelValues)

    // Try to load existing value
    if value, ok := lr.registry.Load(key); ok {
        return value.(T)
    }

    // Atomically create and store if absent (matches Java's computeIfAbsent)
    // Note: factory may be called multiple times in race conditions, but only
    // one result will be stored. Factory should be pure (no side effects).
    newValue := lr.factory(labelValues)
    actual, _ := lr.registry.LoadOrStore(key, newValue)
    return actual.(T)
}

// Dynamic counter
type DynamicCounter struct {
    name      string
    labelKeys []string  // Stored from variadic arguments
    registry  *LabelRegistry[*Counter]
}

// Method signature using variadic arguments for label keys
func (m *Metrics) CounterWithLabels(name string, labelKeys ...string) *DynamicCounter {
    return &DynamicCounter{
        name:      name,
        labelKeys: labelKeys,  // Variadic args become slice
        registry:  newLabelRegistry[*Counter](labelKeys, func(vals []string) *Counter {
            labels := labelValuesToMap(labelKeys, vals)
            return NewCounter(name, labels)
        }),
    }
}

func (dc *DynamicCounter) Inc(n int64, labelValues ...string) {
    // Get or create counter for this label combination
    counter := dc.registry.Get(labelValues)
    counter.Add(n)
}

// Similar pattern for DynamicGauge
type DynamicGauge struct {
    name      string
    labelKeys []string
    registry  *LabelRegistry[*Gauge]
}

func (m *Metrics) GaugeWithLabels(name string, labelKeys ...string) *DynamicGauge {
    return &DynamicGauge{
        name:      name,
        labelKeys: labelKeys,
        registry:  newLabelRegistry[*Gauge](labelKeys, func(vals []string) *Gauge {
            labels := labelValuesToMap(labelKeys, vals)
            return NewGauge(name, labels)
        }),
    }
}

// Similar pattern for DynamicDistribution
type DynamicDistribution struct {
    name      string
    labelKeys []string
    registry  *LabelRegistry[*Distribution]
}

func (m *Metrics) DistributionWithLabels(name, unit string, step, numBuckets int, labelKeys ...string) *DynamicDistribution {
    return &DynamicDistribution{
        name:      name,
        labelKeys: labelKeys,
        registry:  newLabelRegistry[*Distribution](labelKeys, func(vals []string) *Distribution {
            labels := labelValuesToMap(labelKeys, vals)
            return NewDistribution(name, unit, step, numBuckets, labels)
        }),
    }
}
```

### Key Implementation Details

1. **Label Key Ordering**: Label keys are stored in the order provided by the user

2. **Label Value to Map Conversion**: The `labelValuesToMap()` function (shown in Implementation Structure above) is a shared utility used by all dynamic metric types (Counters, Gauges, and Distributions) to convert label keys and values into a map. This ensures consistent permissive behavior across all metric types.

3. **Registry Key Generation**: Create a unique key from label values:
   ```go
   func labelValuesKey(values []string) string {
       return strings.Join(values, "\x00")  // Use null byte as separator
   }
   ```

4. **Thread Safety**:
   - Use `sync.Map` for concurrent access to the registry
   - The `Get()` method must use `LoadOrStore()` for atomic get-or-create (see implementation above)
   - Metric instances (Counter/Gauge/Distribution) are already thread-safe:
     - Counter: Uses `atomic` operations
     - Gauge: Uses `atomic` operations
     - Distribution: Uses `sync.Mutex`
   - Factory functions should be pure (no side effects) as they may be called multiple times in race conditions
   - See `docs/THREAD_SAFETY_ANALYSIS.md` for detailed thread safety analysis

5. **Emission**: When emitting, iterate over all label combinations in the registry and emit each as a separate time series

### Integration with Existing Code

The emitter would need to be updated to:
1. Handle both static and dynamic metrics
2. For dynamic metrics, iterate over all label combinations in the registry
3. Emit each combination as a separate time series

#### Iteration Approaches in Go

Go doesn't have a built-in `Iterable` interface like Java, but there are several approaches for flattening collections:

**Recommended Approach: Using iterutil with Flatten and Concat**

To keep the emitter code minimal (single loop per metric type), use an iterutil library with `Flatten` and `Concat` functions along with `slices.Values` to create a single iterator combining static and dynamic metrics:

```go
import "iter"
import "slices"
import "github.com/example/iterutil" // or your iterutil library

// DynamicCounter returns an iterator over all its Counter instances
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

// In emitter, combine static and dynamic counters into a single iterator:
// 1. Convert static counters slice to iterator
staticCounters := slices.Values(metrics.Counters)

// 2. Map dynamic counters to their iterators, then flatten
dynamicCounterIterators := iterutil.Map(
    slices.Values(metrics.DynamicCounters),
    func(dc *DynamicCounter) iter.Seq[*Counter] {
        return dc.AllCounters()
    },
)
dynamicCounters := iterutil.FlattenSeq(dynamicCounterIterators)

// 4. Concat static and dynamic iterators into one
allCounters := iterutil.Concat(staticCounters, dynamicCounters)

// 5. Single loop handles all counters (static + dynamic)
for counter := range allCounters {
    value := counter.Value()
    ts := &monitoringpb.TimeSeries{
        Metric:   me.buildMetric(counter.Name, counter.Labels),
        Resource: me.MonitoredResource,
        Points: []*monitoringpb.Point{
            {
                Interval: interval,
                Value: &monitoringpb.TypedValue{
                    Value: &monitoringpb.TypedValue_Int64Value{
                        Int64Value: value,
                    },
                },
            },
        },
    }
    timeSeriesList = append(timeSeriesList, ts)
}
```

The same pattern applies for gauges and distributions, keeping the emitter code minimal with just one loop per metric type.

**Note on iterutil functions**: The `iterutil.Map`, `iterutil.FlattenSeq`, and `iterutil.Concat` functions are expected to be provided by an iterutil library. If such a library doesn't exist, these can be implemented as:

```go
// Map transforms each element of an iterator using a function
func Map[A, B any](seq iter.Seq[A], f func(A) B) iter.Seq[B] {
    return func(yield func(B) bool) {
        for a := range seq {
            if !yield(f(a)) {
                return
            }
        }
    }
}

// FlattenSeq takes an iterator of iterators and returns a single iterator over all elements
func FlattenSeq[T any](seq iter.Seq[iter.Seq[T]]) iter.Seq[T] {
    return func(yield func(T) bool) {
        for innerSeq := range seq {
            for v := range innerSeq {
                if !yield(v) {
                    return
                }
            }
        }
    }
}

// Concat combines multiple iterators into one
func Concat[T any](seqs ...iter.Seq[T]) iter.Seq[T] {
    return func(yield func(T) bool) {
        for _, seq := range seqs {
            for v := range seq {
                if !yield(v) {
                    return
                }
            }
        }
    }
}
```

This approach keeps the emitter code minimal with a single loop per metric type, avoiding code duplication and making it easy to handle both static and dynamic metrics uniformly.

### Example Implementation Flow

```go
// 1. Create dynamic counter with variadic label keys
counter := metrics.CounterWithLabels("http_requests", "status", "method")

// 2. Use with different label combinations
counter.Inc(1, "200", "GET")   // Creates Counter with labels: status=200, method=GET
counter.Inc(1, "404", "GET")   // Creates Counter with labels: status=404, method=GET

// 3. During emission
// For each label combination in registry:
//   - Get the Counter instance
//   - Build metric with label keys + label values
//   - Emit as separate time series
```

## Benefits of Dynamic Labels

1. **Flexibility**: Single metric instance handles multiple label combinations
2. **Memory Efficiency**: Only creates aggregators for combinations actually used
3. **API Simplicity**: Cleaner API for metrics with varying labels
4. **Performance**: Lazy creation of aggregators only when needed

## Migration Path

1. **Phase 1**: Implement dynamic labels alongside existing static labels
2. **Phase 2**: Update examples and documentation
3. **Phase 3**: (Optional) Deprecate static labels in favor of dynamic labels

## Design Decisions

### Label Key Validation

**Decision: Follow Java Library Approach (No Validation)**

The Go library will **NOT validate label keys** according to Google Cloud Monitoring's naming conventions. This matches the Java library's behavior.

**Rationale:**
- Consistent with Java library implementation
- Simpler implementation
- Errors will be caught by Google Cloud Monitoring API at emission time
- Users are responsible for ensuring valid label keys

**Google Cloud Monitoring Label Key Requirements** (for reference):
According to the [official documentation](https://docs.cloud.google.com/monitoring/api/v3/naming-conventions#naming-types-and-labels), label keys must:
- Use only lower-case letters (`a-z`), digits (`0-9`), and underscores (`_`)
- Start with a letter
- Have a maximum length of 100 characters
- Be unique within the metric type
- Have no more than 30 labels per metric type

### Label Value Validation

**Decision: Follow Java Library Approach (Permissive, Silent)**

The Go library will use a **permissive, silent approach** for handling mismatched label values, matching the Java library's behavior.

#### Behavior When Values Don't Match Keys

1. **Fewer values than keys** (e.g., 1 value for 2 keys):
   - The library accepts all provided values
   - Only the first N keys (where N = number of values) get paired with values
   - Remaining keys are left without values (empty strings or missing labels)
   - No error is thrown

2. **More values than keys** (e.g., 3 values for 2 keys):
   - The library trims the values array to match the number of keys
   - Only the first N values (where N = number of keys) are used
   - Extra values are silently ignored
   - No error is thrown

**Implementation:**
```go
// Similar to Java's Math.min() approach
func labelValuesToMap(keys []string, values []string) map[string]string {
    result := make(map[string]string)
    for i, key := range keys {
        if i < len(values) {
            result[key] = values[i]
        }
        // If i >= len(values), key is left without a value
    }
    return result
}
```

**Rationale:**
- Consistent with Java library implementation
- Flexible API that doesn't fail on mismatches
- Users are responsible for providing correct number of values

### Serialization

**Decision: Not a Concern**

Serialization is not a concern in Go for this library. Unlike Java, Go doesn't have built-in serialization requirements for metrics objects. The library design does not need to account for serialization.

### Memory Management

**Decision: User Responsibility**

Memory cleanup is not a library concern. The user is responsible for being responsible with the amount of key-value pairs they produce. The library will:

- Create metric instances for each unique label combination as needed
- Store them in the registry for reuse
- Not provide explicit cleanup mechanisms
- Rely on Go's garbage collector for memory management

**Rationale:**
- Keeps the library simple and focused
- Users control their metric usage patterns
- Go's garbage collector handles cleanup automatically
- Consistent with Go's philosophy of simplicity
