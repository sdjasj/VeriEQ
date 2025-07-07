package CodeGenerator

import (
	"fmt"
	"math/rand"
	"strings"
)

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

// Implement GetRealBitWidth and GetRealSignedness

func (e *AssignExpression) GetRealBitWidth() int {
	return e.realWidth
}

func (e *AssignExpression) GetRealSignedness() bool {
	return e.realSigned
}

func (b *BinaryExpression) GetRealBitWidth() int {
	return b.realWidth
}

func (b *BinaryExpression) GetRealSignedness() bool {
	return b.realSigned
}

func (u *UnaryExpression) GetRealBitWidth() int {
	return u.realWidth
}

func (u *UnaryExpression) GetRealSignedness() bool {
	return u.realSigned
}

func (v *VariableExpression) GetRealBitWidth() int {
	return v.realWidth
}

func (v *VariableExpression) GetRealSignedness() bool {
	return v.realSigned
}

func (n *NumberExpression) GetRealBitWidth() int {
	return n.realWidth
}

func (n *NumberExpression) GetRealSignedness() bool {
	return n.realSigned
}

func (t *TernaryExpression) GetRealBitWidth() int {
	return t.realWidth
}

func (t *TernaryExpression) GetRealSignedness() bool {
	return t.realSigned
}

func (c *ConcatenationExpression) GetRealBitWidth() int {
	return c.realWidth
}

func (c *ConcatenationExpression) GetRealSignedness() bool {
	return c.realSigned
}

func (r *ReplicationExpression) GetRealBitWidth() int {
	return r.realWidth
}

func (r *ReplicationExpression) GetRealSignedness() bool {
	return r.realSigned
}

func (e *AssignExpression) GenerateString() string {
	if e.UsedRange == nil {
		return fmt.Sprintf("    assign %s = %s;", e.Operand1.Name, e.Right.GenerateString())
	}
	return fmt.Sprintf("    assign %s[%d:%d] = %s;", e.Operand1.Name, e.UsedRange.r, e.UsedRange.l,
		e.Right.GenerateString())
}

func (b *BinaryExpression) GenerateString() string {
	//if b.Operator == "/" || b.Operator == "%" {
	//	leftVar := fmt.Sprintf("({1'b1, %s})", b.Right.GenerateString())
	//	return "(" + b.Left.GenerateString() + " " + b.Operator + " " + leftVar + ")"
	//}
	return "(" + b.Left.GenerateString() + " " + b.Operator + " " + b.Right.GenerateString() + ")"
}

func (u *UnaryExpression) GenerateString() string {
	return u.Operator + "(" + u.Operand.GenerateString() + ")"
}

func (v *VariableExpression) GenerateString() string {
	if v.hasRange {
		return fmt.Sprintf("%s[%d:%d]", v.Var.Name, v.Range.r, v.Range.l)
	}
	return v.Var.Name
}

func (n *NumberExpression) GenerateString() string {
	return n.Value.ToVerilogLiteral()
}

func (t *TernaryExpression) GenerateString() string {
	return fmt.Sprintf("((%s) ? (%s) : (%s))", t.Condition.GenerateString(), t.TrueExpr.GenerateString(), t.FalseExpr.GenerateString())
}

func (c *ConcatenationExpression) GenerateString() string {
	var parts []string
	for _, expr := range c.Expressions {
		parts = append(parts, expr.GenerateString())
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

func (r *ReplicationExpression) GenerateString() string {
	return fmt.Sprintf("{%s{%s}}", r.Count.GenerateString(), r.Expression.GenerateString())
}

func isZero(e Expression) bool {
	if n, ok := e.(*NumberExpression); ok {
		return n.Value.Value == 0
	}
	return false
}

func isOne(e Expression) bool {
	if n, ok := e.(*NumberExpression); ok {
		return n.Value.Value == 1
	}
	return false
}

func newZero(width int, signed bool) Expression {
	cn := ConstNumber{
		Value:      0,
		BitWidth:   width,
		Signedness: signed,
	}
	return &NumberExpression{
		Value: cn, ctxWidth: width,
		ctxSigned:   signed,
		isCtxSet:    true,
		isSignedSet: true,
		realWidth:   width,
		realSigned:  signed}
}

func maxWidth(a, b Expression) int {
	wa := a.GetRealBitWidth()
	wb := b.GetRealBitWidth()
	if wa > wb {
		return wa
	}
	return wb
}

func (b *BinaryExpression) clone(left, right Expression, op string) *BinaryExpression {
	return &BinaryExpression{
		Left:        left,
		Right:       right,
		Operator:    op,
		ctxWidth:    b.ctxWidth,
		ctxSigned:   b.ctxSigned,
		isCtxSet:    b.isCtxSet,
		isSignedSet: b.isSignedSet,
		realWidth:   b.realWidth,
		realSigned:  b.realSigned,
	}
}

func (b *BinaryExpression) EquivalentTrans() Expression {
	left := b.Left.EquivalentTrans()
	right := b.Right.EquivalentTrans()

	switch b.Operator {
	case "+", "*", "&", "|", "^":
		if rand.Float64() < 0.5 {
			return b.clone(right, left, b.Operator)
		}
	}

	if b.Operator == ">=" {
		return b.clone(right, left, "<=")
	}
	if b.Operator == "<=" {
		return b.clone(right, left, ">=")
	}

	if b.Operator == ">>" {
		if n, ok := right.(*NumberExpression); ok && n.Value.Value >= uint64(left.GetRealBitWidth()) {
			return newZero(left.GetRealBitWidth(), left.GetSignedness())
		}
	}

	if b.Operator == ">>>" {
		if n, ok := right.(*NumberExpression); ok &&
			n.Value.Value >= (uint64(left.GetRealBitWidth())) &&
			!left.GetSignedness() {
			//if left.GetRealBitWidth() == 0 {
			//	fmt.Println(left.GenerateString()+"fuck !!!!!!!!!!! ")
			//}
			return newZero(left.GetRealBitWidth(), false)
		}
	}

	if b.Operator == "<<" {
		if n, ok := right.(*NumberExpression); ok && n.Value.Value >= uint64(left.GetRealBitWidth()) {
			return newZero(left.GetRealBitWidth(), left.GetSignedness())
		}
	}

	if b.Operator == "<<<" {
		if n, ok := right.(*NumberExpression); ok && n.Value.Value >= uint64(left.GetRealBitWidth()) {
			return newZero(left.GetRealBitWidth(), left.GetSignedness())
		}
	}

	switch b.Operator {
	case "+":
		if isZero(right) && right.GetBitWidth() == left.GetBitWidth() &&
			right.GetSignedness() == left.GetSignedness() {
			return left
		}
		if isZero(left) && left.GetBitWidth() == left.GetBitWidth() &&
			left.GetSignedness() == right.GetSignedness() {
			return right
		}
	case "*":
		if isOne(right) && right.GetBitWidth() == left.GetBitWidth() &&
			right.GetSignedness() == left.GetSignedness() {
			return left
		}
		if isOne(left) && left.GetBitWidth() == left.GetBitWidth() &&
			left.GetSignedness() == right.GetSignedness() {
			return right
		}

		if isZero(left) || isZero(right) {
			w := maxWidth(left, right)
			return newZero(w, left.GetRealSignedness() && right.GetRealSignedness())
		}
	case "/":
		if isOne(right) && left.GetBitWidth() == left.GetBitWidth() &&
			left.GetSignedness() == right.GetSignedness() {
			return left
		}
	case "&":
		if u, ok := right.(*UnaryExpression); ok && u.Operator == "~" && u.Operand.GenerateString() == left.GenerateString() {
			w := left.GetRealBitWidth()
			return newZero(w, left.GetRealSignedness())
		}
	}

	return b
}

func (u *UnaryExpression) EquivalentTrans() Expression {
	if rand.Float64() > 0.5 {
		u.Operand = u.Operand.EquivalentTrans()
	}
	return u
}

func (n *NumberExpression) EquivalentTrans() Expression {
	// 常数本身不拆分，直接返回
	return n
}

func (v *VariableExpression) EquivalentTrans() Expression {
	// 变量子范围也只是返回自己
	return v
}

func (t *TernaryExpression) EquivalentTrans() Expression {
	if rand.Float64() > 0.5 {
		t.Condition = t.Condition.EquivalentTrans()
	}
	if rand.Float64() > 0.5 {
		t.TrueExpr = t.TrueExpr.EquivalentTrans()
	}
	if rand.Float64() > 0.5 {
		t.FalseExpr = t.FalseExpr.EquivalentTrans()
	}

	return t
}

func (c *ConcatenationExpression) EquivalentTrans() Expression {
	for i, e := range c.Expressions {
		if rand.Float64() > 0.5 {
			c.Expressions[i] = e.EquivalentTrans()
		}
	}
	return c
}

func (r *ReplicationExpression) EquivalentTrans() Expression {
	if rand.Float64() > 0.5 {
		r.Expression = r.Expression.EquivalentTrans()
	}

	return r
}

func (e *AssignExpression) EquivalentTrans() Expression {

	right := e.Right.EquivalentTrans()
	return &AssignExpression{
		Operand1:  e.Operand1,
		Right:     right,
		UsedRange: e.UsedRange,
	}
}

// implement bit width
func (e *NumberExpression) GetBitWidth() int {
	if e.isCtxSet {
		return e.ctxWidth
	}
	e.ctxWidth = e.Value.BitWidth
	e.isCtxSet = true
	return e.ctxWidth
}

// implement sign
func (e *NumberExpression) GetSignedness() bool {
	if e.isSignedSet {
		return e.ctxSigned
	}
	e.isSignedSet = true
	e.ctxSigned = e.Value.Signedness
	return e.ctxSigned
}

func (e *VariableExpression) GetBitWidth() int {
	if e.isCtxSet {
		return e.ctxWidth
	}
	if e.hasRange {
		e.ctxWidth = e.Range.GetWidth()
	} else {
		e.ctxWidth = e.Var.GetWidth()
	}
	e.isCtxSet = true
	return e.ctxWidth
}

func (e *VariableExpression) GetSignedness() bool {
	if e.isSignedSet {
		return e.ctxSigned
	}
	if e.hasRange {
		e.ctxSigned = false
	} else {
		e.ctxSigned = e.Var.isSigned
	}
	e.isSignedSet = true
	return e.ctxSigned
}

func (b *BinaryExpression) GetBitWidth() int {
	if b.isCtxSet {
		return b.ctxWidth
	}
	b.isCtxSet = true
	switch b.Operator {
	case "+", "-", "*", "/", "%", "&", "|", "^", "^~", "~^":

		leftWidth := b.Left.GetBitWidth()
		rightWidth := b.Right.GetBitWidth()
		if leftWidth > rightWidth {
			b.ctxWidth = leftWidth
		} else {
			b.ctxWidth = rightWidth
		}
		return b.ctxWidth
	case "===", "!==", "==", "!=", ">", ">=", "<", "<=":
		b.ctxWidth = 1
		return 1
	case "&&", "||":

		b.ctxWidth = 1
		return 1
	case ">>", "<<", "**", ">>>", "<<<":

		b.ctxWidth = b.Left.GetBitWidth()
		return b.ctxWidth
	default:

		b.ctxWidth = b.Left.GetBitWidth()
		return b.ctxWidth
	}
}

func (b *BinaryExpression) GetSignedness() bool {
	if b.isSignedSet {
		return b.ctxSigned
	}
	b.isSignedSet = true
	switch b.Operator {
	case "===", "!==", "==", "!=", ">", ">=", "<", "<=", "&&", "||":

		b.ctxSigned = false
		return false
	default:

		b.ctxSigned = b.Left.GetSignedness() && b.Right.GetSignedness()
		return b.ctxSigned
	}
}

func (u *UnaryExpression) GetBitWidth() int {
	if u.isCtxSet {
		return u.ctxWidth
	}
	u.isCtxSet = true
	switch u.Operator {
	case "&", "~&", "|", "~|", "^", "~^", "^~", "!":

		u.ctxWidth = 1
		return 1
	default:

		u.ctxWidth = u.Operand.GetBitWidth()
		return u.ctxWidth
	}
}

func (u *UnaryExpression) GetSignedness() bool {
	if u.isSignedSet {
		return u.ctxSigned
	}
	u.isSignedSet = true
	switch u.Operator {
	case "&", "~&", "|", "~|", "^", "~^", "^~", "!":

		u.ctxSigned = false
		return false
	default:
		u.ctxSigned = u.Operand.GetSignedness()

		return u.ctxSigned
	}
}

func (t *TernaryExpression) GetBitWidth() int {

	if t.isCtxSet {
		return t.ctxWidth
	}
	t.isCtxSet = true
	trueWidth := t.TrueExpr.GetBitWidth()
	falseWidth := t.FalseExpr.GetBitWidth()
	if trueWidth > falseWidth {
		t.ctxWidth = trueWidth
		return trueWidth
	}
	t.ctxWidth = falseWidth
	return falseWidth
}

func (t *TernaryExpression) GetSignedness() bool {

	if t.isSignedSet {
		return t.ctxSigned
	}
	t.isSignedSet = true
	t.ctxSigned = t.TrueExpr.GetSignedness() && t.FalseExpr.GetSignedness()
	return t.ctxSigned
}

func (c *ConcatenationExpression) GetBitWidth() int {

	if c.isCtxSet {
		return c.ctxWidth
	}
	c.isCtxSet = true
	totalWidth := 0
	for _, expr := range c.Expressions {
		totalWidth += expr.GetBitWidth()
	}
	c.ctxWidth = totalWidth
	return c.ctxWidth
}

func (c *ConcatenationExpression) GetSignedness() bool {
	// 拼接结果总是无符号的
	return false
}

func (r *ReplicationExpression) GetBitWidth() int {
	if r.isCtxSet {
		return r.ctxWidth
	}
	r.isCtxSet = true

	count := r.Count.(*NumberExpression).Value.Value
	exprWidth := r.Expression.GetBitWidth()
	r.ctxWidth = int(count) * exprWidth
	return r.ctxWidth
}

func (r *ReplicationExpression) GetSignedness() bool {
	// 重复拼接结果总是无符号的
	return false
}

func (e *AssignExpression) GetBitWidth() int {
	r := e.Right.GetBitWidth()
	l := 1
	if e.UsedRange != nil {
		l = e.UsedRange.GetWidth()
	}

	if r > l {
		return r
	}
	return l
}

func (e *AssignExpression) GetSignedness() bool {
	return e.Right.GetSignedness()
}

// PropagateType methods
func (b *BinaryExpression) PropagateType(width int, signed bool) {
	b.realWidth = width
	b.realSigned = signed
	//if b.realWidth == 0 {
	//	fmt.Println(b.GenerateString())
	//}
	switch b.Operator {
	case "==", "!=", "===", "!==", ">", ">=", "<", "<=":
		lw := b.Left.GetBitWidth()
		rw := b.Right.GetBitWidth()
		maxw := lw
		if rw > lw {
			maxw = rw
		}
		ls := b.Left.GetSignedness()
		rs := b.Right.GetSignedness()
		mergedSigned := ls && rs
		b.realWidth = 1
		b.realSigned = false
		b.Left.PropagateType(maxw, mergedSigned)
		b.Right.PropagateType(maxw, mergedSigned)
	case ">>", "<<", ">>>", "<<<":
		b.Left.PropagateType(width, signed)
		b.Right.PropagateType(b.Right.GetBitWidth(), b.Right.GetSignedness()) // Right is always treated as unsigned
	default:
		b.Left.PropagateType(width, signed)
		b.Right.PropagateType(width, signed)
	}
}

func (u *UnaryExpression) PropagateType(width int, signed bool) {
	u.realWidth = width
	u.realSigned = signed
	switch u.Operator {
	case "&", "~&", "|", "~|", "^", "~^", "^~", "!":

		u.realWidth = 1
		u.realSigned = false
		u.Operand.PropagateType(u.Operand.GetBitWidth(), u.Operand.GetSignedness())
	default:

		u.Operand.PropagateType(width, signed)
	}
}

func (v *VariableExpression) PropagateType(width int, signed bool) {
	v.realWidth = width
	v.realSigned = signed
}

func (n *NumberExpression) PropagateType(width int, signed bool) {
	n.realWidth = width
	n.realSigned = signed
}

func (t *TernaryExpression) PropagateType(width int, signed bool) {
	t.realWidth = width
	t.realSigned = signed
	t.Condition.PropagateType(t.Condition.GetBitWidth(), t.Condition.GetSignedness())
	t.TrueExpr.PropagateType(width, signed)
	t.FalseExpr.PropagateType(width, signed)
}

func (c *ConcatenationExpression) PropagateType(width int, signed bool) {
	c.realWidth = width
	c.realSigned = false
	for i := 0; i < len(c.Expressions); i++ {
		c.Expressions[i].PropagateType(c.Expressions[i].GetBitWidth(), c.Expressions[i].GetSignedness()) // self-determined
	}
}

func (r *ReplicationExpression) PropagateType(width int, signed bool) {
	r.realWidth = width
	r.realSigned = false
	r.Count.PropagateType(r.Count.GetBitWidth(), r.Count.GetSignedness())
	r.Expression.PropagateType(r.Expression.GetBitWidth(), r.Expression.GetSignedness())
}

func (e *AssignExpression) PropagateType(width int, signed bool) {
	width = e.GetBitWidth()
	signed = e.Right.GetSignedness()
	e.Right.PropagateType(width, signed)
}
