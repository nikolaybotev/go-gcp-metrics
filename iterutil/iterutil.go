// Package iterutil provides iterator utility functions for working with iter.Seq.
package iterutil

import (
	"iter"
	"slices"
)

// MetricRegistry is an interface for metric types that provide an iterator over all their instances.
// This is implemented by DynamicCounter, DynamicGauge, and DynamicDistribution.
type MetricRegistry[T any] interface {
	All() iter.Seq[T]
}

// CombineMetrics creates a single iterator combining static metrics (slice) with dynamic metrics.
// This is useful for iterating over both static and dynamic metrics in a single loop.
func CombineMetrics[T any, D MetricRegistry[T]](static []T, dynamic []D) iter.Seq[T] {
	staticIter := slices.Values(static)
	dynamicIters := Map(slices.Values(dynamic), func(d D) iter.Seq[T] {
		return d.All()
	})
	dynamicFlat := FlattenSeq(dynamicIters)
	return Concat(staticIter, dynamicFlat)
}

// Map transforms each element of an iterator using a function.
func Map[A, B any](seq iter.Seq[A], f func(A) B) iter.Seq[B] {
	return func(yield func(B) bool) {
		for a := range seq {
			if !yield(f(a)) {
				return
			}
		}
	}
}

// FlattenSeq takes an iterator of iterators and returns a single iterator over all elements.
func FlattenSeq[T any](seq iter.Seq[iter.Seq[T]]) iter.Seq[T] {
	return func(yield func(T) bool) {
		for innerSeq := range seq {
			for v := range innerSeq {
				if !yield(v) {
					return
				}
			}
		}
	}
}

// Concat combines multiple iterators into one.
func Concat[T any](seqs ...iter.Seq[T]) iter.Seq[T] {
	return func(yield func(T) bool) {
		for _, seq := range seqs {
			for v := range seq {
				if !yield(v) {
					return
				}
			}
		}
	}
}
