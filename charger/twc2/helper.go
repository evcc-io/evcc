package twc2

func equals(l, r []byte) bool {
	if len(l) != len(r) {
		return false
	}

	for i, b := range l {
		if r[i] != b {
			return false
		}
	}

	return true
}
