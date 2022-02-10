package sponsor

var Subject, Token string

func IsAuthorized() bool {
	return len(Subject) > 0
}
