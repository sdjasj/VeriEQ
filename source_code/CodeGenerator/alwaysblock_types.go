package CodeGenerator

type AlwaysBlockType int

const (
	AlwaysComb AlwaysBlockType = iota
	AlwaysFF
	AlwaysLatch
)

type AlwaysBlock struct {
	Type         AlwaysBlockType
	ClockVars    []*Variable
	ClockPosedge []bool
	ResetVar     *Variable
	ResetValue   string
	Statements   []Statement
	UsedVars     []*Variable
	ForcePosedge bool
}

type Statement interface {
	GenerateString() string
}

type IfStatement struct {
	Condition Expression
	TrueBody  []Statement
	ElseBody  []Statement
}

type CaseStatement struct {
	Expression Expression
	Cases      []CaseItem
	Default    []Statement
}

type CaseItem struct {
	Value      string
	Statements []Statement
}

type BlockingAssignment struct {
	Target     *Variable
	Expression Expression
	Range      *BitRange
}

type NonBlockingAssignment struct {
	Target     *Variable
	Expression Expression
	Range      *BitRange
}
