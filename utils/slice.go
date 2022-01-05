package utils

import (
	"isc-route-service/pkg/domain"
	"sort"
)

func SortRoute() {
	routes := [2]domain.RouteInfo{}
	routes[0] = domain.RouteInfo{
		Id: 1,
	}
	routes[1] = domain.RouteInfo{
		Id: 2,
	}
	sort.SliceIsSorted(routes, func(i, j int) bool {
		return routes[i].Id < routes[j].Id
	})
}
