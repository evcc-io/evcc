package eebus

type status int

const (
	StatusOK status = iota
	StatusLimit
	StatusFailsafe
)
