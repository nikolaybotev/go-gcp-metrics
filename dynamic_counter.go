package gcpmetrics

import (
	"iter"
	"maps"
)

// DynamicCounter is a counter that supports dynamic label values.
// Label keys are defined at creation time, and label values are provided
// when incrementing the counter. Each unique combination of label values
// gets its own StaticCounter instance.
type DynamicCounter struct {
	Name         string
	staticLabels map[string]string
	labelKeys    []string
	registry     *LabelRegistry[*StaticCounter]
}

// NewDynamicCounter creates a new DynamicCounter with the given name, static labels, and dynamic label keys.
// Static labels are fixed at creation time and included in all emitted metrics.
// Dynamic label keys define which labels will have values provided at runtime.
func NewDynamicCounter(name string, staticLabels map[string]string, labelKeys ...string) *DynamicCounter {
	// Handle nil staticLabels gracefully
	if staticLabels == nil {
		staticLabels = make(map[string]string)
	}
	return &DynamicCounter{
		Name:         name,
		staticLabels: staticLabels,
		labelKeys:    labelKeys,
		registry: newLabelRegistry(labelKeys, func(vals []string) *StaticCounter {
			// Merge static labels with dynamic label values
			labels := make(map[string]string, len(staticLabels)+len(labelKeys))
			maps.Copy(labels, staticLabels)
			dynamicLabels := labelValuesToMap(labelKeys, vals)
			maps.Copy(labels, dynamicLabels)
			return NewStaticCounter(name, labels)
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

// All returns an iterator over all StaticCounter instances in this DynamicCounter.
// This is used by the emitter to iterate over all label combinations.
func (dc *DynamicCounter) All() iter.Seq[*StaticCounter] {
	return dc.registry.All()
}
