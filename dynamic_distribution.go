package gcpmetrics

import (
	"iter"
	"maps"
)

// DynamicDistribution is a distribution that supports dynamic label values.
// Label keys are defined at creation time, and label values are provided
// when updating the distribution. Each unique combination of label values
// gets its own StaticDistribution instance.
type DynamicDistribution struct {
	Name         string
	Unit         string
	Step         int
	NumBuckets   int
	staticLabels map[string]string
	labelKeys    []string
	registry     *LabelRegistry[*StaticDistribution]
}

// NewDynamicDistribution creates a new DynamicDistribution with the given parameters, static labels, and dynamic label keys.
// Static labels are fixed at creation time and included in all emitted metrics.
// Dynamic label keys define which labels will have values provided at runtime.
func NewDynamicDistribution(name, unit string, step, numBuckets int, staticLabels map[string]string, labelKeys ...string) *DynamicDistribution {
	// Handle nil staticLabels gracefully
	if staticLabels == nil {
		staticLabels = make(map[string]string)
	}
	return &DynamicDistribution{
		Name:         name,
		Unit:         unit,
		Step:         step,
		NumBuckets:   numBuckets,
		staticLabels: staticLabels,
		labelKeys:    labelKeys,
		registry: newLabelRegistry(labelKeys, func(vals []string) *StaticDistribution {
			// Merge static labels with dynamic label values
			labels := make(map[string]string, len(staticLabels)+len(labelKeys))
			maps.Copy(labels, staticLabels)
			dynamicLabels := labelValuesToMap(labelKeys, vals)
			maps.Copy(labels, dynamicLabels)
			return NewStaticDistribution(name, unit, step, numBuckets, labels)
		}),
	}
}

// Update records a value in the distribution for the given label values.
func (dd *DynamicDistribution) Update(value int64, labelValues ...string) {
	dd.registry.Get(labelValues).Update(value)
}

// All returns an iterator over all StaticDistribution instances in this DynamicDistribution.
// This is used by the emitter to iterate over all label combinations.
func (dd *DynamicDistribution) All() iter.Seq[*StaticDistribution] {
	return dd.registry.All()
}
