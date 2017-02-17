package pod

type Podfile struct {
	FilePath string
	Header   []byte
	Targets  []*Target
	Footer   []byte
}

type Target struct {
	Name    string
	Depends []*Depend
}

type Depend struct {
	DependBase
	SpecPath    string
	SpecDepends []*DependBase
	Type        string
	Err         error
}

// *** Private ***
type p_podfile struct {
	Target_definitions []*p_target_definition
}

type p_target_definition struct {
	Abstract     bool
	Children     []*p_target
	Dependencies []interface{}
	Name         string
}

type p_target struct {
	Dependencies []interface{}
	Name         string
}
