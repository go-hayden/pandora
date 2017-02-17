package pod

type IDepend interface {
	Version() string
	Name() string
	Subdepends() []*DependBase
	IsLocal() bool
}

type DependBase struct {
	N string
	V string
}

func (s *DependBase) Version() string {
	return s.V
}

func (s *DependBase) Name() string {
	return s.N
}

func (s *DependBase) Subdepends() []*DependBase {
	return nil
}

func (s *DependBase) IsLocal() bool {
	return false
}

func (s *DependBase) String() string {
	return "[" + s.N + ":" + s.V + "]"
}
