package plugins

type Id int
type RouteInfo struct {
	Id
	Path        string   `json:"path"`
	ServiceId   string   `json:"serviceId"`
	Url         string   `json:"url"`
	CreateTime  string   `json:"createTime"`
	UpdateTime  string   `json:"updateTime"`
	Protocol    string   `json:"protocol"`
	ExcludeUrls []string `json:"excludeUrls"`
	SpecialUrls []string `json:"specialUrls"`
}
