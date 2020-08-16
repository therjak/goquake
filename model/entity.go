package model

type Entity struct {
	properties map[string]string
}

func NewEntity(p map[string]string) *Entity {
	return &Entity{p}
}

func (e *Entity) Property(name string) (string, bool) {
	v, ok := e.properties[name]
	return v, ok
}

func (e *Entity) Name() (string, bool) {
	v, ok := e.properties["classname"]
	return v, ok
}

func (e *Entity) PropertyNames() []string {
	n := []string{}
	for k, _ := range e.properties {
		n = append(n, k)
	}
	return n
}
