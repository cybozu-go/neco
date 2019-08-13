package cke

func compareStrings(s1, s2 []string) bool {
	if len(s1) != len(s2) {
		return false
	}
	for i := range s1 {
		if s1[i] != s2[i] {
			return false
		}
	}
	return true
}

func compareStringMap(m1, m2 map[string]string) bool {
	if len(m1) != len(m2) {
		return false
	}
	for k, v := range m1 {
		if v2, ok := m2[k]; !ok || v != v2 {
			return false
		}
	}
	return true
}

func compareMounts(m1, m2 []Mount) bool {
	if len(m1) != len(m2) {
		return false
	}

	for i := range m1 {
		if !m1[i].Equal(m2[i]) {
			return false
		}
	}
	return true
}
