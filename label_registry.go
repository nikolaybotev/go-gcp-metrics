package gcpmetrics

import (
	"iter"
	"strings"
	"sync"
)

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

// labelValuesKey creates a unique key from label values for use in the registry.
// Uses null byte as separator since it's not a valid character in label values.
func labelValuesKey(values []string) string {
	return strings.Join(values, "\x00")
}

// LabelRegistry manages a thread-safe mapping from label value combinations to metric instances.
// It uses sync.Map for concurrent access and lazy creation of metric instances.
type LabelRegistry[T any] struct {
	labelKeys []string                     // Immutable after creation - DO NOT MODIFY
	registry  sync.Map                     // map[string]T - key is label values joined
	factory   func(labelValues []string) T // Factory function to create new metric instances
}

// newLabelRegistry creates a new LabelRegistry with the given label keys and factory function.
func newLabelRegistry[T any](labelKeys []string, factory func(labelValues []string) T) *LabelRegistry[T] {
	return &LabelRegistry[T]{
		labelKeys: labelKeys,
		factory:   factory,
	}
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

// All returns an iterator over all metric instances in the registry.
// This is used by the emitter to iterate over all label combinations.
func (lr *LabelRegistry[T]) All() iter.Seq[T] {
	return func(yield func(T) bool) {
		lr.registry.Range(func(key, value any) bool {
			return yield(value.(T))
		})
	}
}
