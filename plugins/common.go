package plugins

type PluginError struct {
	StatusCode string
	Content    interface{}
}

func (e *PluginError) Error() string {
	return e.StatusCode
}
