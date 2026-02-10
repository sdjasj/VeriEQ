package CodeGenerator

type Expression interface {
	GenerateString() string
	EquivalentTrans() Expression
	GetBitWidth() int
	GetSignedness() bool // true is signed; false is unsigned
	PropagateType(width int, signed bool)
	GetRealBitWidth() int
	GetRealSignedness() bool
}

type AssignExpression struct {
	Operand1  *Variable
	Right     Expression
	UsedRange *BitRange

	realWidth  int
	realSigned bool
}

type BinaryExpression struct {
	Left     Expression
	Right    Expression
	Operator string

	ctxWidth    int
	ctxSigned   bool
	isCtxSet    bool
	isSignedSet bool

	realWidth  int
	realSigned bool
}

type UnaryExpression struct {
	Operand  Expression
	Operator string

	ctxWidth    int
	ctxSigned   bool
	isCtxSet    bool
	isSignedSet bool

	realWidth  int
	realSigned bool
}

type VariableExpression struct {
	Var      *Variable
	Range    *BitRange
	hasRange bool

	ctxWidth    int
	ctxSigned   bool
	isCtxSet    bool
	isSignedSet bool

	realWidth  int
	realSigned bool
}

type NumberExpression struct {
	Value ConstNumber

	ctxWidth    int
	ctxSigned   bool
	isCtxSet    bool
	isSignedSet bool

	realWidth  int
	realSigned bool
}

type TernaryExpression struct {
	Condition Expression
	TrueExpr  Expression
	FalseExpr Expression

	ctxWidth    int
	ctxSigned   bool
	isCtxSet    bool
	isSignedSet bool

	realWidth  int
	realSigned bool
}

type ConcatenationExpression struct {
	Expressions []Expression

	ctxWidth    int
	ctxSigned   bool
	isCtxSet    bool
	isSignedSet bool

	realWidth  int
	realSigned bool
}

type ReplicationExpression struct {
	Count      Expression
	Expression Expression

	ctxWidth    int
	ctxSigned   bool
	isCtxSet    bool
	isSignedSet bool

	realWidth  int
	realSigned bool
}
