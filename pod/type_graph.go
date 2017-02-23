package pod

type GraphPodfile map[string]*GraphModule

type GraphModule struct {
	Name            string
	Version         string
	UpdateToVersion string
	NewestVersion   string
	IsCommon        bool
	IsNew           bool
	IsImplicit      bool
	IsLocal         bool
	Depends         []*DependBase
	flattenDepends  []string
}
