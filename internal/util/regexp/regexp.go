package regexp

import "regexp"

type Match[V any] struct {
	Regexp *regexp.Regexp
	Value  V
}

func MatchesAnyRegexp[V any](r []*Match[V], s string) (*Match[V], bool) {
	for _, regex := range r {
		if regex.Regexp.Match([]byte(s)) {
			return regex, true
		}
	}

	return nil, false
}

func StringMatchesAnyRegexp(r []*regexp.Regexp, s string) (*regexp.Regexp, bool) {
	var rs []*Match[any]

	for _, reg := range r {
		val := &Match[any]{
			Regexp: reg,
		}
		rs = append(rs, val)
	}

	selected, ok := MatchesAnyRegexp(rs, s)
	if selected == nil {
		return nil, false
	}

	return selected.Regexp, ok
}

func ExtractFieldsWithMatch[V any](s string, r *Match[V]) map[string]string {
	match := r.Regexp.FindStringSubmatch(s)
	result := make(map[string]string)
	for i, name := range r.Regexp.SubexpNames() {
		if i != 0 && name != "" && i < len(match) {
			result[name] = match[i]
		}
	}

	return result
}

func ExtractFields(s string, r *regexp.Regexp) map[string]string {
	rs := &Match[any]{
		Regexp: r,
	}

	return ExtractFieldsWithMatch(s, rs)
}
