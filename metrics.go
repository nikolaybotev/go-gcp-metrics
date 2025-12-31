package gcpmetrics

// MetricsCollector defines the public interface for metrics implementations.
type MetricsCollector interface {
	// Static label metrics
	Counter(name string, labels map[string]string) *Counter
	Gauge(name string, labels map[string]string) *Gauge
	Distribution(name, unit string, step, numBuckets int, labels map[string]string) *Distribution
	// Dynamic label metrics
	CounterWithLabels(name string, labelKeys ...string) *DynamicCounter
	GaugeWithLabels(name string, labelKeys ...string) *DynamicGauge
	DistributionWithLabels(name, unit string, step, numBuckets int, labelKeys ...string) *DynamicDistribution
	// Lifecycle
	AddBeforeEmitListener(listener func())
}

// Metrics contains the backend-agnostic functionality shared by all Metrics implementations.
// It implements the MetricsCollector interface and can be embedded by backend-specific implementations.
type Metrics struct {
	// Static label metrics
	Counters      []*Counter
	Distributions []*Distribution
	Gauges        []*Gauge
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
		Counters:             []*Counter{},
		Distributions:        []*Distribution{},
		Gauges:               []*Gauge{},
		DynamicCounters:      []*DynamicCounter{},
		DynamicDistributions: []*DynamicDistribution{},
		DynamicGauges:        []*DynamicGauge{},
		BeforeEmitListeners:  []func(){},
	}
}

// addCounter adds a Counter to the metrics.
func (me *Metrics) addCounter(counter *Counter) {
	me.Counters = append(me.Counters, counter)
}

// Counter creates a new Counter, adds it to the metrics, and returns it.
func (me *Metrics) Counter(name string, labels map[string]string) *Counter {
	counter := NewCounter(name, labels)
	me.addCounter(counter)
	return counter
}

// addGauge adds a Gauge to the metrics.
func (me *Metrics) addGauge(g *Gauge) {
	me.Gauges = append(me.Gauges, g)
}

// Gauge creates a new Gauge, adds it to the metrics, and returns it.
func (me *Metrics) Gauge(name string, labels map[string]string) *Gauge {
	g := NewGauge(name, labels)
	me.addGauge(g)
	return g
}

// addDistribution adds a Distribution to the metrics.
func (me *Metrics) addDistribution(dist *Distribution) {
	me.Distributions = append(me.Distributions, dist)
}

// Distribution creates a new Distribution, adds it to the metrics, and returns it.
func (me *Metrics) Distribution(
	name,
	unit string,
	step,
	numBuckets int,
	labels map[string]string,
) *Distribution {
	dist := NewDistribution(name, unit, step, numBuckets, labels)
	me.addDistribution(dist)
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

// addDynamicCounter adds a DynamicCounter to the metrics.
func (me *Metrics) addDynamicCounter(counter *DynamicCounter) {
	me.DynamicCounters = append(me.DynamicCounters, counter)
}

// CounterWithLabels creates a new DynamicCounter, adds it to the metrics, and returns it.
// Label keys are defined at creation time, and label values are provided when incrementing.
func (me *Metrics) CounterWithLabels(name string, labelKeys ...string) *DynamicCounter {
	counter := NewDynamicCounter(name, labelKeys...)
	me.addDynamicCounter(counter)
	return counter
}

// addDynamicGauge adds a DynamicGauge to the metrics.
func (me *Metrics) addDynamicGauge(g *DynamicGauge) {
	me.DynamicGauges = append(me.DynamicGauges, g)
}

// GaugeWithLabels creates a new DynamicGauge, adds it to the metrics, and returns it.
// Label keys are defined at creation time, and label values are provided when setting.
func (me *Metrics) GaugeWithLabels(name string, labelKeys ...string) *DynamicGauge {
	g := NewDynamicGauge(name, labelKeys...)
	me.addDynamicGauge(g)
	return g
}

// addDynamicDistribution adds a DynamicDistribution to the metrics.
func (me *Metrics) addDynamicDistribution(dist *DynamicDistribution) {
	me.DynamicDistributions = append(me.DynamicDistributions, dist)
}

// DistributionWithLabels creates a new DynamicDistribution, adds it to the metrics, and returns it.
// Label keys are defined at creation time, and label values are provided when updating.
func (me *Metrics) DistributionWithLabels(
	name,
	unit string,
	step,
	numBuckets int,
	labelKeys ...string,
) *DynamicDistribution {
	dist := NewDynamicDistribution(name, unit, step, numBuckets, labelKeys...)
	me.addDynamicDistribution(dist)
	return dist
}
