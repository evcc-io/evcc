package util

func Masked(s any) string {
	if s != "" {
		return "***"
	}
	return ""
}
