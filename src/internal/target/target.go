package target

type Target struct {
	Targets []string          `json:"targets",yaml:"targets"`
	Labels  map[string]string `json:"labels",yaml:"labels"`
	Source  string            `json:"-",yaml:"source"`
}