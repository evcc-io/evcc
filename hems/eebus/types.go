package eebus

type status int

const (
	StatusUnlimited status = iota
	StatusLimited
	StatusFailsafe
)
