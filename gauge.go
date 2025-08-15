package main

import "sync/atomic"

type Gauge struct {
	Name   string
	Labels map[string]string
	value  int64
}

func NewGauge(name string, labels map[string]string) *Gauge {
	return &Gauge{
		Name:   name,
		Labels: labels,
	}
}

func (g *Gauge) Set(n int64) {
	atomic.StoreInt64(&g.value, n)
}

func (g *Gauge) Value() int64 {
	return atomic.LoadInt64(&g.value)
}
