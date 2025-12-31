package gcpmetrics

import "iter"

// DynamicDistribution is a distribution that supports dynamic label values.
// Label keys are defined at creation time, and label values are provided
// when updating the distribution. Each unique combination of label values
// gets its own Distribution instance.
type DynamicDistribution struct {
	Name       string
	Unit       string
	Step       int
	NumBuckets int
	labelKeys  []string
	registry   *LabelRegistry[*Distribution]
}

// NewDynamicDistribution creates a new DynamicDistribution with the given parameters and label keys.
func NewDynamicDistribution(name, unit string, step, numBuckets int, labelKeys ...string) *DynamicDistribution {
	return &DynamicDistribution{
		Name:       name,
		Unit:       unit,
		Step:       step,
		NumBuckets: numBuckets,
		labelKeys:  labelKeys,
		registry: newLabelRegistry(labelKeys, func(vals []string) *Distribution {
			labels := labelValuesToMap(labelKeys, vals)
			return NewDistribution(name, unit, step, numBuckets, labels)
		}),
	}
}

// Update records a value in the distribution for the given label values.
func (dd *DynamicDistribution) Update(value int64, labelValues ...string) {
	dd.registry.Get(labelValues).Update(value)
}

// All returns an iterator over all Distribution instances in this DynamicDistribution.
// This is used by the emitter to iterate over all label combinations.
func (dd *DynamicDistribution) All() iter.Seq[*Distribution] {
	return dd.registry.All()
}
