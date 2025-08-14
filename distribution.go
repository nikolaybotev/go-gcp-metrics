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
	state      DistributionBuckets
	mu         sync.Mutex
}

func NewDistribution(name, unit string, step, numBuckets int, labels map[string]string) *Distribution {
	return &Distribution{
		Name:       name,
		Unit:       unit,
		Offset:     0,
		Step:       int64(step),
		NumBuckets: numBuckets,
		Labels:     labels,
		state: DistributionBuckets{
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
	d.state.Buckets[bucket] += 1

	// Update numSamples, mean and M2 using Welford's method for accumulating the sum of squared deviations.
	d.state.NumSamples += 1
	delta := float64(value) - d.state.Mean
	d.state.Mean = d.state.Mean + (delta / float64(d.state.NumSamples))
	d.state.SumOfSquaredDeviation = d.state.SumOfSquaredDeviation + delta*(float64(value)-d.state.Mean)
}

func (d *Distribution) GetAndClear() *DistributionBuckets {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Make a copy
	result := &DistributionBuckets{
		Buckets:               make([]int64, len(d.state.Buckets)),
		NumSamples:            d.state.NumSamples,
		Mean:                  d.state.Mean,
		SumOfSquaredDeviation: d.state.SumOfSquaredDeviation,
	}
	copy(result.Buckets, d.state.Buckets)

	// Clear
	for i := range d.state.Buckets {
		d.state.Buckets[i] = 0
	}
	d.state.NumSamples = 0
	d.state.Mean = 0
	d.state.SumOfSquaredDeviation = 0

	return result
}

func (d *Distribution) bucketForValue(value int64) int {
	return int(min(max(0, (value-d.Offset+d.Step)/d.Step), int64(d.NumBuckets+1)))
}
