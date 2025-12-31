package gcpmetrics

import "sync/atomic"

// Gauge is the public interface for gauges.
// Both StaticGauge and DynamicGauge implement this interface.
type Gauge interface {
	// Set sets the gauge value. For dynamic gauges, labelValues specify the label combination.
	Set(n int64, labelValues ...string)
}

// StaticGauge is a gauge with fixed labels defined at creation time.
// It ignores any labelValues passed to Set method.
type StaticGauge struct {
	Name   string
	Labels map[string]string
	value  int64
}

// NewStaticGauge creates a new StaticGauge with the given name and labels.
func NewStaticGauge(name string, labels map[string]string) *StaticGauge {
	return &StaticGauge{
		Name:   name,
		Labels: labels,
	}
}

// Set sets the gauge value. The labelValues parameter is ignored for static gauges.
func (g *StaticGauge) Set(n int64, labelValues ...string) {
	atomic.StoreInt64(&g.value, n)
}

// Value returns the current gauge value.
func (g *StaticGauge) Value() int64 {
	return atomic.LoadInt64(&g.value)
}
