package plugins

type Id int
type RouteInfo struct {
	Id
	Path        string   `json:"path"`
	ServiceId   string   `json:"serviceId"`
	Url         string   `json:"url"`
	Protocol    string   `json:"protocol"`
	ExcludeUrl  string   `json:"excludeUrl"`
	ExcludeUrls []string `json:"excludeUrls"`
	SpecialUrl  string   `json:"specialUrl"`
	SpecialUrls []string `json:"specialUrls"`
	CreateTime  string   `json:"create_time"`
	UpdateTime  string   `json:"update_time"`
	AppCode     string   `json:"appCode"`
}
