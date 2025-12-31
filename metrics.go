package gcpmetrics

// MetricsCollector defines the public interface for metrics implementations.
type MetricsCollector interface {
	// Counter creates a counter with optional static labels and dynamic label keys.
	Counter(name string, labels map[string]string, labelKeys ...string) Counter
	// Gauge creates a gauge with optional static labels and dynamic label keys.
	// If labelKeys is empty, returns a StaticGauge; otherwise returns a DynamicGauge.
	Gauge(name string, labels map[string]string, labelKeys ...string) Gauge
	// Distribution creates a distribution with optional static labels and dynamic label keys.
	// If labelKeys is empty, returns a StaticDistribution; otherwise returns a DynamicDistribution.
	Distribution(name, unit string, step, numBuckets int, labels map[string]string, labelKeys ...string) Distribution
	// Lifecycle
	AddBeforeEmitListener(listener func())
}

// Metrics contains the backend-agnostic functionality shared by all Metrics implementations.
// It implements the MetricsCollector interface and can be embedded by backend-specific implementations.
type Metrics struct {
	// Static label metrics
	Counters      []*StaticCounter
	Distributions []*StaticDistribution
	Gauges        []*StaticGauge
	// Dynamic label metrics
	DynamicCounters      []*DynamicCounter
	DynamicDistributions []*DynamicDistribution
	DynamicGauges        []*DynamicGauge
	// Lifecycle
	BeforeEmitListeners []func()
}

// NewMetrics creates a new Metrics instance.
func NewMetrics() *Metrics {
	return &Metrics{
		Counters:             []*StaticCounter{},
		Distributions:        []*StaticDistribution{},
		Gauges:               []*StaticGauge{},
		DynamicCounters:      []*DynamicCounter{},
		DynamicDistributions: []*DynamicDistribution{},
		DynamicGauges:        []*DynamicGauge{},
		BeforeEmitListeners:  []func(){},
	}
}

// Counter creates a counter with optional static labels and dynamic label keys.
// If labelKeys is empty, returns a StaticCounter; otherwise returns a DynamicCounter.
// Both implement the Counter interface.
func (me *Metrics) Counter(name string, labels map[string]string, labelKeys ...string) Counter {
	if len(labelKeys) == 0 {
		counter := NewStaticCounter(name, labels)
		me.Counters = append(me.Counters, counter)
		return counter
	}
	counter := NewDynamicCounter(name, labels, labelKeys...)
	me.DynamicCounters = append(me.DynamicCounters, counter)
	return counter
}

// Gauge creates a gauge with optional static labels and dynamic label keys.
// If labelKeys is empty, returns a StaticGauge; otherwise returns a DynamicGauge.
// Both implement the Gauge interface.
func (me *Metrics) Gauge(name string, labels map[string]string, labelKeys ...string) Gauge {
	if len(labelKeys) == 0 {
		gauge := NewStaticGauge(name, labels)
		me.Gauges = append(me.Gauges, gauge)
		return gauge
	}
	gauge := NewDynamicGauge(name, labels, labelKeys...)
	me.DynamicGauges = append(me.DynamicGauges, gauge)
	return gauge
}

// Distribution creates a distribution with optional static labels and dynamic label keys.
// If labelKeys is empty, returns a StaticDistribution; otherwise returns a DynamicDistribution.
// Both implement the Distribution interface.
func (me *Metrics) Distribution(
	name,
	unit string,
	step,
	numBuckets int,
	labels map[string]string,
	labelKeys ...string,
) Distribution {
	if len(labelKeys) == 0 {
		dist := NewStaticDistribution(name, unit, step, numBuckets, labels)
		me.Distributions = append(me.Distributions, dist)
		return dist
	}
	dist := NewDynamicDistribution(name, unit, step, numBuckets, labels, labelKeys...)
	me.DynamicDistributions = append(me.DynamicDistributions, dist)
	return dist
}

// AddBeforeEmitListener adds a listener that will be called before each emit.
func (me *Metrics) AddBeforeEmitListener(listener func()) {
	me.BeforeEmitListeners = append(me.BeforeEmitListeners, listener)
}

// NotifyBeforeEmitListeners calls all registered before-emit listeners.
func (m *Metrics) notifyBeforeEmitListeners() {
	for _, listener := range m.BeforeEmitListeners {
		if listener != nil {
			listener()
		}
	}
}
