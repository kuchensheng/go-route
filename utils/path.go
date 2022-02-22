package utils

import "strings"

func Match(src, target string) bool {
	srcPaths := strings.Split(src, "/")
	targetPaths := strings.Split(target, "/")

	if len(targetPaths) > len(srcPaths) {
		return false
	}

	for idx, p := range targetPaths {
		if p == "" {
			continue
		}
		if strings.Contains(p, "*") {
			return true
		}
		if p != srcPaths[idx] {
			return false
		}
	}
	return false
}

func IsInSlice(src []string, target string) bool {
	for _, path := range src {
		return Match(target, path)
	}
	return false
}
