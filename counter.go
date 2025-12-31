package gcpmetrics

import (
	"sync/atomic"
)

// Counter is the public interface for counters.
// Both StaticCounter and DynamicCounter implement this interface.
type Counter interface {
	// Inc increments the counter by 1. For dynamic counters, labelValues specify the label combination.
	Inc(labelValues ...string)
	// Add increments the counter by n. For dynamic counters, labelValues specify the label combination.
	Add(n int64, labelValues ...string)
}

// StaticCounter is a counter with fixed labels defined at creation time.
// It ignores any labelValues passed to Inc/Add methods.
type StaticCounter struct {
	Name   string
	Labels map[string]string
	value  int64
}

// NewStaticCounter creates a new StaticCounter with the given name and labels.
func NewStaticCounter(name string, labels map[string]string) *StaticCounter {
	return &StaticCounter{
		Name:   name,
		Labels: labels,
	}
}

// Inc increments the counter by 1. The labelValues parameter is ignored for static counters.
func (c *StaticCounter) Inc(labelValues ...string) {
	atomic.AddInt64(&c.value, 1)
}

// Add increments the counter by n. The labelValues parameter is ignored for static counters.
func (c *StaticCounter) Add(n int64, labelValues ...string) {
	atomic.AddInt64(&c.value, n)
}

// Value returns the current counter value.
func (c *StaticCounter) Value() int64 {
	return atomic.LoadInt64(&c.value)
}
