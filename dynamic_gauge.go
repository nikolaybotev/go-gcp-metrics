package gcpmetrics

import (
	"iter"
	"maps"
)

// DynamicGauge is a gauge that supports dynamic label values.
// Label keys are defined at creation time, and label values are provided
// when setting the gauge. Each unique combination of label values
// gets its own StaticGauge instance.
type DynamicGauge struct {
	Name         string
	staticLabels map[string]string
	labelKeys    []string
	registry     *LabelRegistry[*StaticGauge]
}

// NewDynamicGauge creates a new DynamicGauge with the given name, static labels, and dynamic label keys.
// Static labels are fixed at creation time and included in all emitted metrics.
// Dynamic label keys define which labels will have values provided at runtime.
func NewDynamicGauge(name string, staticLabels map[string]string, labelKeys ...string) *DynamicGauge {
	// Handle nil staticLabels gracefully
	if staticLabels == nil {
		staticLabels = make(map[string]string)
	}
	return &DynamicGauge{
		Name:         name,
		staticLabels: staticLabels,
		labelKeys:    labelKeys,
		registry: newLabelRegistry(labelKeys, func(vals []string) *StaticGauge {
			// Merge static labels with dynamic label values
			labels := make(map[string]string, len(staticLabels)+len(labelKeys))
			maps.Copy(labels, staticLabels)
			dynamicLabels := labelValuesToMap(labelKeys, vals)
			maps.Copy(labels, dynamicLabels)
			return NewStaticGauge(name, labels)
		}),
	}
}

// Set sets the gauge value for the given label values.
func (dg *DynamicGauge) Set(n int64, labelValues ...string) {
	dg.registry.Get(labelValues).Set(n)
}

// Value returns the current value for the given label values.
func (dg *DynamicGauge) Value(labelValues ...string) int64 {
	return dg.registry.Get(labelValues).Value()
}

// All returns an iterator over all StaticGauge instances in this DynamicGauge.
// This is used by the emitter to iterate over all label combinations.
func (dg *DynamicGauge) All() iter.Seq[*StaticGauge] {
	return dg.registry.All()
}
