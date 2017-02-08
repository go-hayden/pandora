package pod

type PodProject struct {
	FilePath               string
	Podfiles               []*Podfile
	CommonDeps             map[string]*PFDependence
	AdditionalCodeTemplate string
	AdditionalCode         string
}

type Podfile struct {
	FilePath               string
	HeaderTemplate         string
	Header                 string
	Targets                map[string]map[string]*PFDependence
	AdditionalCodeTemplate string
	AdditionalCode         string
}

type PFDependence struct {
	Version string
	SpecURI string
	Type    string
	Desc    string
}
