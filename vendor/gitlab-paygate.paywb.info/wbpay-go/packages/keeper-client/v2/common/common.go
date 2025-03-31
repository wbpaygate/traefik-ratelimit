package common

func InList(list []string, value string) bool {
	if len(list) > 0 {
		for _, listItem := range list {
			if listItem == value {
				return true
			}
		}
	}

	return false
}

func ListToMap(list []string) map[string]struct{} {
	m := make(map[string]struct{}, len(list))
	for _, item := range list {
		m[item] = struct{}{}
	}

	return m
}
