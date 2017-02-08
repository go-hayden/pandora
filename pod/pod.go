package pod

// Type Define
type Pod struct {
	PodRepos []*PodRepo
}

type PodBase struct {
	Name string
	Root string
}

type PodRepo struct {
	PodBase
	Modules []*PodModule
}

type PodModule struct {
	PodBase
	Versions []*PodModuleVersion
}

type PodModuleVersion struct {
	PodBase
	Source   string
	FileName string
	Podspec  *Spec
	Err      error
}
