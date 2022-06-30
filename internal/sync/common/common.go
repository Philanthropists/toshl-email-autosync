package common

import (
	"regexp"

	"github.com/Philanthropists/toshl-email-autosync/internal/logger"
)

var version string

type RegexWithValue[V any] struct {
	Regexp *regexp.Regexp
	Value  V
}

func GenericMatchesAnyRegexp[V any](r []*RegexWithValue[V], s string) (bool, *RegexWithValue[V]) {
	for _, regex := range r {
		if regex.Regexp.Match([]byte(s)) {
			return true, regex
		}
	}

	return false, nil
}

func MatchesAnyRegexp(r []*regexp.Regexp, s string) (bool, *regexp.Regexp) {
	var rs []*RegexWithValue[bool]

	for _, reg := range r {
		val := &RegexWithValue[bool]{
			Regexp: reg,
			Value:  false,
		}
		rs = append(rs, val)
	}

	res, selected := GenericMatchesAnyRegexp[bool](rs, s)
	if selected == nil {
		return false, nil
	}
	return res, selected.Regexp
}

func GenericExtractFieldsStringWithRegexp[V any](s string, r *RegexWithValue[V]) map[string]string {
	match := r.Regexp.FindStringSubmatch(s)
	result := make(map[string]string)
	for i, name := range r.Regexp.SubexpNames() {
		if i != 0 && name != "" && i < len(match) {
			result[name] = match[i]
		}
	}

	return result
}

func ExtractFieldsStringWithRegexp(s string, r *regexp.Regexp) map[string]string {
	rs := &RegexWithValue[bool]{
		Regexp: r,
		Value:  false,
	}

	res := GenericExtractFieldsStringWithRegexp[bool](s, rs)

	return res
}

func ContainsAllRequiredFields(fields map[string]string) bool {
	requiredFields := []string{"value", "type", "place", "account"}
	requiredFieldsSet := map[string]struct{}{}
	for _, field := range requiredFields {
		requiredFieldsSet[field] = struct{}{}
	}

	for field := range requiredFieldsSet {
		_, ok := fields[field]
		if !ok {
			return false
		}
	}

	return true
}

func PrintVersion(commit string) {
	log := logger.GetLogger()
	defer log.Sync()

	log.Infof("Commit version: %s", commit)
	version = commit
}

func GetVersion() string {
	return version
}
