package settings

//go:generate mockgen -package settings -destination mock.go -mock_names API=MockAPI github.com/evcc-io/evcc/server/db/settings API

type API interface {
	String(key string) (string, error)
	SetString(key string, value string)
}
