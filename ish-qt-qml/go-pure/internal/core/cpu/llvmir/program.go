package llvmir

type OpKind uint8

const (
	OpConst OpKind = iota
	OpAdd
	OpSub
	OpMul
)

type Op struct {
	Kind  OpKind
	Value uint64
}

type Program struct {
	Name string
	Ops  []Op
}

func (p Program) functionName() string {
	if p.Name == "" {
		return "guest_block"
	}
	return p.Name
}
