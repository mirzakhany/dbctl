package container

const (
	// LabelType is the label used to identify the type of database
	LabelType = "dbctl_type"
	// LabelCustom is the label used to identify a database
	LabelCustom = "dbctl_custom"
)

type Container struct {
	ID     string
	Name   string
	Labels map[string]string
}

type CreateRequest struct {
	Name         string
	Image        string
	ExposedPorts []string // allow specifying protocol info
	Cmd          []string
	Env          map[string]string
	Labels       map[string]string
}
