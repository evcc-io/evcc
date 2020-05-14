package cmd

import "github.com/andig/evcc/util"

// Tee distributed parameters to subscribers
type Tee struct {
	recv []chan<- util.Param
}

// Attach creates a new receiver channel and attaches it to the tee
func (t *Tee) Attach() <-chan util.Param {
	out := make(chan util.Param)
	t.Add(out)
	return out
}

// Add attaches a receiver channel to the tee
func (t *Tee) Add(out chan<- util.Param) {
	t.recv = append(t.recv, out)
}

// Run starts parameter distribution
func (t *Tee) Run(in <-chan util.Param) {
	for msg := range in {
		for _, recv := range t.recv {
			recv <- msg
		}
	}
}
