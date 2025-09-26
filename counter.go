package gcpmetrics

import (
	"sync/atomic"
)

type Counter struct {
	Name   string
	Labels map[string]string
	value  int64
}

func NewCounter(name string, labels map[string]string) *Counter {
	return &Counter{
		Name:   name,
		Labels: labels,
	}
}

func (c *Counter) Inc() {
	atomic.AddInt64(&c.value, 1)
}

func (c *Counter) Add(n int64) {
	atomic.AddInt64(&c.value, n)
}

func (c *Counter) Value() int64 {
	return atomic.LoadInt64(&c.value)
}
