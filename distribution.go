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
	Name                  string
	Unit                  string
	Offset                int64
	Step                  int64
	NumBuckets            int
	Labels                map[string]string
	Buckets               []int64
	NumSamples            int64
	Mean                  float64
	SumOfSquaredDeviation float64
	mu                    sync.Mutex
}

func NewDistribution(name, unit string, step, numBuckets int, labels map[string]string) *Distribution {
	return &Distribution{
		Name:       name,
		Unit:       unit,
		Offset:     0,
		Step:       int64(step),
		NumBuckets: numBuckets,
		Labels:     labels,
		Buckets:    make([]int64, numBuckets+2),
	}
}

func (d *Distribution) Update(value int64) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Update bucket
	bucket := d.BucketForValue(value)
	d.Buckets[bucket] += 1

	// Update numSamples, mean and M2 using Welford's method for accumulating the sum of squared deviations.
	d.NumSamples += 1
	delta := float64(value) - d.Mean
	d.Mean = d.Mean + (delta / float64(d.NumSamples))
	d.SumOfSquaredDeviation = d.SumOfSquaredDeviation + delta*(float64(value)-d.Mean)
}

func (d *Distribution) GetAndClear() *DistributionBuckets {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Make a copy
	result := &DistributionBuckets{
		Buckets:               make([]int64, len(d.Buckets)),
		NumSamples:            d.NumSamples,
		Mean:                  d.Mean,
		SumOfSquaredDeviation: d.SumOfSquaredDeviation,
	}
	copy(result.Buckets, d.Buckets)

	// Clear
	for i := range d.Buckets {
		d.Buckets[i] = 0
	}
	d.NumSamples = 0
	d.Mean = 0
	d.SumOfSquaredDeviation = 0

	return result
}

func (d *Distribution) BucketForValue(value int64) int {
	return int(min(max(0, (value-d.Offset+d.Step)/d.Step), int64(d.NumBuckets+1)))
}
