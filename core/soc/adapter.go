package soc

// Adapter provides the required methods for interacting with the loadpoint
type Adapter interface {
	Publish(key string, val interface{})
	SocEstimator() *Estimator
	ActivePhases() int
	Voltage() float64
}
