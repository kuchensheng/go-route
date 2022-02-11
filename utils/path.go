package utils

import "strings"

func Match(src, target string) bool {
	srcPaths := strings.Split(src, "/")
	targetPaths := strings.Split(target, "/")

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
