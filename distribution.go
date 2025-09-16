package main

import (
	"sync"
)

type DistributionBuckets struct {
	Buckets               []int64
	NumSamples            int64
	Mean                  float64
	SumOfSquaredDeviation float64
}

type Distribution struct {
	Name       string
	Unit       string
	Offset     int64
	Step       int64
	NumBuckets int
	Labels     map[string]string
	value      DistributionBuckets
	mu         sync.Mutex
}

// Create a new Distribution with the given name, unit, step, numBuckets, and labels.
// Unit format is documented at: https://cloud.google.com/monitoring/api/ref_v3/rest/v3/projects.metricDescriptors
func NewDistribution(name, unit string, step, numBuckets int, labels map[string]string) *Distribution {
	return &Distribution{
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

func (d *Distribution) Update(value int64) {
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

func (d *Distribution) GetAndClear() *DistributionBuckets {
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
	for i := range d.value.Buckets {
		d.value.Buckets[i] = 0
	}
	d.value.NumSamples = 0
	d.value.Mean = 0
	d.value.SumOfSquaredDeviation = 0

	return result
}

// BucketBounds returns the bucket boundaries for this distribution.
func (d *Distribution) BucketBounds() []float64 {
	bucketBounds := make([]float64, d.NumBuckets+1)
	for i := 0; i <= d.NumBuckets; i++ {
		bucketBounds[i] = float64(d.Offset) + float64(d.Step)*float64(i)
	}
	return bucketBounds
}

func (d *Distribution) bucketForValue(value int64) int {
	return int(min(max(0, (value-d.Offset+d.Step)/d.Step), int64(d.NumBuckets+1)))
}

// min and max functions to handle int64 values (they are built-in in Go only since 1.21)

func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
