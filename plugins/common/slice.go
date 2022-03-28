package plugins

import (
	"sort"
	"strings"
)

func SortRoute() {
	routes := [2]RouteInfo{}
	routes[0] = RouteInfo{
		Id: 1,
	}
	routes[1] = RouteInfo{
		Id: 2,
	}
	sort.SliceIsSorted(routes, func(i, j int) bool {
		return routes[i].Id < routes[j].Id
	})
}

func Match(src, target string) bool {
	srcPaths := strings.Split(src, "/")
	targetPaths := strings.Split(target, "/")

	if len(targetPaths) > len(srcPaths) {
		return false
	}
	if strings.HasPrefix(src, target) {
		return true
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
