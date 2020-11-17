package detect

type Result struct {
	Task
	Host    string
	Details interface{}
}
