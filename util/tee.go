package util

// TeeAttacher allows to attach a listener to a tee
type TeeAttacher interface {
	Attach() <-chan Param
}

// Tee distributed parameters to subscribers
type Tee struct {
	recv []chan<- Param
}

// Attach creates a new receiver channel and attaches it to the tee
func (t *Tee) Attach() <-chan Param {
	// TODO find better approach to prevent deadlocks
	// this will buffer the receiver channel to prevent deadlocks when consumers use mutex-protected loadpoint api
	out := make(chan Param, 16)
	t.add(out)
	return out
}

// add attaches a receiver channel to the tee
func (t *Tee) add(out chan<- Param) {
	t.recv = append(t.recv, out)
}

// Run starts parameter distribution
func (t *Tee) Run(in <-chan Param) {
	for msg := range in {
		for _, recv := range t.recv {
			recv <- msg
		}
	}
}
