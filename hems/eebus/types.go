package eebus

type status int

const (
	StatusNormal status = iota
	StatusFailsafe
)
