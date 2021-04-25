package sponsor

var Subject string

func IsAuthorized() bool {
	return len(Subject) > 0
}
