package util

func Masked(s string) string {
	if s != "" {
		return "***"
	}
	return ""
}
