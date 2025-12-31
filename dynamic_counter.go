package gcpmetrics

import "iter"

// DynamicCounter is a counter that supports dynamic label values.
// Label keys are defined at creation time, and label values are provided
// when incrementing the counter. Each unique combination of label values
// gets its own Counter instance.
type DynamicCounter struct {
	Name      string
	labelKeys []string
	registry  *LabelRegistry[*Counter]
}

// NewDynamicCounter creates a new DynamicCounter with the given name and label keys.
func NewDynamicCounter(name string, labelKeys ...string) *DynamicCounter {
	return &DynamicCounter{
		Name:      name,
		labelKeys: labelKeys,
		registry: newLabelRegistry(labelKeys, func(vals []string) *Counter {
			labels := labelValuesToMap(labelKeys, vals)
			return NewCounter(name, labels)
		}),
	}
}

// Inc increments the counter by 1 for the given label values.
func (dc *DynamicCounter) Inc(labelValues ...string) {
	dc.registry.Get(labelValues).Inc()
}

// Add adds the given value to the counter for the given label values.
func (dc *DynamicCounter) Add(n int64, labelValues ...string) {
	dc.registry.Get(labelValues).Add(n)
}

// Value returns the current value for the given label values.
func (dc *DynamicCounter) Value(labelValues ...string) int64 {
	return dc.registry.Get(labelValues).Value()
}

// All returns an iterator over all Counter instances in this DynamicCounter.
// This is used by the emitter to iterate over all label combinations.
func (dc *DynamicCounter) All() iter.Seq[*Counter] {
	return dc.registry.All()
}
