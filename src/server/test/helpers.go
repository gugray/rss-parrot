package test

import "strings"

func strSliceMatch(items []string) func(x any) bool {
	res := func(x any) bool {
		slice, ok := x.([]string)
		if !ok {
			return false
		}
		if len(slice) != len(items) {
			return false
		}
		for i := 0; i < len(slice); i++ {
			if slice[i] != items[i] {
				return false
			}
		}
		return true
	}
	return res
}

func strStartsWith(prefix string) func(x any) bool {
	res := func(x any) bool {
		str, ok := x.(string)
		if !ok {
			return false
		}
		return strings.HasPrefix(str, prefix)
	}
	return res
}
