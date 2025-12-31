package gcpmetrics

import "iter"

// DynamicGauge is a gauge that supports dynamic label values.
// Label keys are defined at creation time, and label values are provided
// when setting the gauge. Each unique combination of label values
// gets its own Gauge instance.
type DynamicGauge struct {
	Name      string
	labelKeys []string
	registry  *LabelRegistry[*Gauge]
}

// NewDynamicGauge creates a new DynamicGauge with the given name and label keys.
func NewDynamicGauge(name string, labelKeys ...string) *DynamicGauge {
	return &DynamicGauge{
		Name:      name,
		labelKeys: labelKeys,
		registry: newLabelRegistry(labelKeys, func(vals []string) *Gauge {
			labels := labelValuesToMap(labelKeys, vals)
			return NewGauge(name, labels)
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

// All returns an iterator over all Gauge instances in this DynamicGauge.
// This is used by the emitter to iterate over all label combinations.
func (dg *DynamicGauge) All() iter.Seq[*Gauge] {
	return dg.registry.All()
}
