package validation

func ContainsAllRequiredFields(fields map[string]string) bool {
	req := []string{"value", "type", "place", "account"}

	reqSet := make(map[string]struct{})
	for _, field := range req {
		reqSet[field] = struct{}{}
	}

	for field := range reqSet {
		_, ok := fields[field]
		if !ok {
			return false
		}
	}

	return true
}
