package gcpmetrics

// MetricsCore contains the backend-agnostic functionality shared by all Metrics implementations.
// It implements the Metrics interface and can be embedded by backend-specific implementations.
type MetricsCore struct {
	Counters            []*Counter
	Distributions       []*Distribution
	Gauges              []*Gauge
	BeforeEmitListeners []func()
}

// NewMetricsCore creates a new MetricsCore instance.
func NewMetricsCore() *MetricsCore {
	return &MetricsCore{
		Counters:            []*Counter{},
		Distributions:       []*Distribution{},
		Gauges:              []*Gauge{},
		BeforeEmitListeners: []func(){},
	}
}

// addCounter adds a Counter to the metrics.
func (me *MetricsCore) addCounter(counter *Counter) {
	me.Counters = append(me.Counters, counter)
}

// Counter creates a new Counter, adds it to the metrics, and returns it.
func (me *MetricsCore) Counter(name string, labels map[string]string) *Counter {
	counter := NewCounter(name, labels)
	me.addCounter(counter)
	return counter
}

// addGauge adds a Gauge to the metrics.
func (me *MetricsCore) addGauge(g *Gauge) {
	me.Gauges = append(me.Gauges, g)
}

// Gauge creates a new Gauge, adds it to the metrics, and returns it.
func (me *MetricsCore) Gauge(name string, labels map[string]string) *Gauge {
	g := NewGauge(name, labels)
	me.addGauge(g)
	return g
}

// addDistribution adds a Distribution to the metrics.
func (me *MetricsCore) addDistribution(dist *Distribution) {
	me.Distributions = append(me.Distributions, dist)
}

// Distribution creates a new Distribution, adds it to the metrics, and returns it.
func (me *MetricsCore) Distribution(
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
func (me *MetricsCore) AddBeforeEmitListener(listener func()) {
	me.BeforeEmitListeners = append(me.BeforeEmitListeners, listener)
}
