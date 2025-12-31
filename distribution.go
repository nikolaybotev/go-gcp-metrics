package gcpmetrics

import (
	"sync"
)

// Distribution is the public interface for distributions.
// Both StaticDistribution and DynamicDistribution implement this interface.
type Distribution interface {
	// Update records a value in the distribution. For dynamic distributions, labelValues specify the label combination.
	Update(value int64, labelValues ...string)
}

// DistributionBuckets holds the bucket data for a distribution.
type DistributionBuckets struct {
	Buckets               []int64
	NumSamples            int64
	Mean                  float64
	SumOfSquaredDeviation float64
}

// StaticDistribution is a distribution with fixed labels defined at creation time.
// It ignores any labelValues passed to Update method.
type StaticDistribution struct {
	Name       string
	Unit       string
	Offset     int64
	Step       int64
	NumBuckets int
	Labels     map[string]string
	value      DistributionBuckets
	mu         sync.Mutex
}

// NewStaticDistribution creates a new StaticDistribution with the given name, unit, step, numBuckets, and labels.
// Unit format is documented at: https://cloud.google.com/monitoring/api/ref_v3/rest/v3/projects.metricDescriptors
func NewStaticDistribution(name, unit string, step, numBuckets int, labels map[string]string) *StaticDistribution {
	return &StaticDistribution{
		Name:       name,
		Unit:       unit,
		Offset:     0,
		Step:       int64(step),
		NumBuckets: numBuckets,
		Labels:     labels,
		value: DistributionBuckets{
			// Allocate numBuckets + 2 to account for underflow (bucket 0) and overflow (last bucket)
			Buckets: make([]int64, numBuckets+2),
		},
	}
}

// Update records a value in the distribution. The labelValues parameter is ignored for static distributions.
func (d *StaticDistribution) Update(value int64, labelValues ...string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Update bucket
	bucket := d.bucketForValue(value)
	d.value.Buckets[bucket] += 1

	// Update numSamples, mean and M2 using Welford's method for accumulating the sum of squared deviations.
	d.value.NumSamples += 1
	delta := float64(value) - d.value.Mean
	d.value.Mean = d.value.Mean + (delta / float64(d.value.NumSamples))
	d.value.SumOfSquaredDeviation = d.value.SumOfSquaredDeviation + delta*(float64(value)-d.value.Mean)
}

// GetAndClear returns the current distribution data and resets the distribution.
func (d *StaticDistribution) GetAndClear() *DistributionBuckets {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Make a copy
	result := &DistributionBuckets{
		Buckets:               make([]int64, len(d.value.Buckets)),
		NumSamples:            d.value.NumSamples,
		Mean:                  d.value.Mean,
		SumOfSquaredDeviation: d.value.SumOfSquaredDeviation,
	}
	copy(result.Buckets, d.value.Buckets)

	// Clear
	clear(d.value.Buckets)
	d.value.NumSamples = 0
	d.value.Mean = 0
	d.value.SumOfSquaredDeviation = 0

	return result
}

// BucketBounds returns the bucket boundaries for this distribution.
func (d *StaticDistribution) BucketBounds() []float64 {
	bucketBounds := make([]float64, d.NumBuckets+1)
	for i := 0; i <= d.NumBuckets; i++ {
		bucketBounds[i] = float64(d.Offset) + float64(d.Step)*float64(i)
	}
	return bucketBounds
}

func (d *StaticDistribution) bucketForValue(value int64) int {
	return int(min(max(0, (value-d.Offset+d.Step)/d.Step), int64(d.NumBuckets+1)))
}
