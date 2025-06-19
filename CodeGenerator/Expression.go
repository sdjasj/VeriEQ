package CodeGenerator

import (
	"fmt"
	"math/rand"
	"strings"
)

// Expression 表示Verilog表达式的基本接口
type Expression interface {
	GenerateString() string
	EquivalentTrans() Expression
	GetBitWidth() int
	GetSignedness() bool // true is signed; false is unsigned
}

type AssignExpression struct {
	Operand1  *Variable
	Right     Expression
	UsedRange *BitRange
}

func (e *AssignExpression) GenerateString() string {
	if e.UsedRange == nil {
		return fmt.Sprintf("    assign %s = %s;", e.Operand1.Name, e.Right.GenerateString())
	}
	return fmt.Sprintf("    assign %s[%d:%d] = %s;", e.Operand1.Name, e.UsedRange.r, e.UsedRange.l,
		e.Right.GenerateString())
}

type BinaryExpression struct {
	Left     Expression
	Right    Expression
	Operator string

	ctxWidth    int
	ctxSigned   bool
	isCtxSet    bool
	isSignedSet bool
}

type UnaryExpression struct {
	Operand  Expression
	Operator string

	ctxWidth    int
	ctxSigned   bool
	isCtxSet    bool
	isSignedSet bool
}

type VariableExpression struct {
	Var      *Variable
	Range    *BitRange
	hasRange bool

	ctxWidth    int
	ctxSigned   bool
	isCtxSet    bool
	isSignedSet bool
}

type NumberExpression struct {
	Value ConstNumber

	ctxWidth    int
	ctxSigned   bool
	isCtxSet    bool
	isSignedSet bool
}

type TernaryExpression struct {
	Condition Expression
	TrueExpr  Expression
	FalseExpr Expression

	ctxWidth    int
	ctxSigned   bool
	isCtxSet    bool
	isSignedSet bool
}

type ConcatenationExpression struct {
	Expressions []Expression

	ctxWidth    int
	ctxSigned   bool
	isCtxSet    bool
	isSignedSet bool
}

type ReplicationExpression struct {
	Count      Expression // 重复次数
	Expression Expression // 要重复的表达式

	ctxWidth    int
	ctxSigned   bool
	isCtxSet    bool
	isSignedSet bool
}

// 实现各种表达式的GenerateString方法
func (b *BinaryExpression) GenerateString() string {
	if b.Operator == "/" || b.Operator == "%" {
		leftVar := fmt.Sprintf("({1'b1, %s})", b.Right.GenerateString())
		return "(" + b.Left.GenerateString() + " " + b.Operator + " " + leftVar + ")"
	}
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
	return t.Condition.GenerateString() + " ? " + t.TrueExpr.GenerateString() + " : " + t.FalseExpr.GenerateString()
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

// isZero 检测表达式是否等价于常量 0
func isZero(e Expression) bool {
	if n, ok := e.(*NumberExpression); ok {
		return n.Value.Value == 0
	}
	return false
}

// isOne 检测表达式是否等价于常量 1
func isOne(e Expression) bool {
	if n, ok := e.(*NumberExpression); ok {
		return n.Value.Value == 1
	}
	return false
}

// newZero 根据宽度和符号性构造一个 0
func newZero(width int, signed bool) Expression {
	cn := ConstNumber{
		Value:      0,
		BitWidth:   width,
		Signedness: signed,
	}
	return &NumberExpression{Value: cn}
}

func newOne(width int, signed bool) Expression {
	cn := ConstNumber{
		Value:      1,
		BitWidth:   width,
		Signedness: signed,
	}
	return &NumberExpression{Value: cn}
}

// maxWidth 取两个子表达式中最大的位宽
func maxWidth(a, b Expression) int {
	wa := a.GetBitWidth()
	wb := b.GetBitWidth()
	if wa > wb {
		return wa
	}
	return wb
}

// --- BinaryExpression 等价变换 ------------------------------------------------

func (b *BinaryExpression) EquivalentTrans() Expression {
	// 1) 先对孩子结点做等价变换
	left := b.Left.EquivalentTrans()
	right := b.Right.EquivalentTrans()

	// 2) A 类：交换律（A1–A5）
	switch b.Operator {
	case "+", "*", "&", "|", "^":
		if rand.Float64() < 0.5 {
			return &BinaryExpression{
				Left:     right,
				Right:    left,
				Operator: b.Operator,
			}
		}
	}

	// 3) A6：ge ↔ le
	if b.Operator == ">=" {
		return &BinaryExpression{Left: right, Right: left, Operator: "<="}
	}
	if b.Operator == "<=" {
		return &BinaryExpression{Left: right, Right: left, Operator: ">="}
	}

	// B1: shr (>>)
	if b.Operator == ">>" {
		if n, ok := right.(*NumberExpression); ok && n.Value.Value >= uint64(left.GetBitWidth()) {
			return newZero(left.GetBitWidth(), left.GetSignedness())
		}
	}
	// B2: ashr (>>>)，仅当 E 为无符号时
	if b.Operator == ">>>" {
		if n, ok := right.(*NumberExpression); ok &&
			n.Value.Value >= (uint64(left.GetBitWidth())) &&
			!left.GetSignedness() {
			return newZero(left.GetBitWidth(), false)
		}
	}
	// B3: shl (<<)
	if b.Operator == "<<" {
		if n, ok := right.(*NumberExpression); ok && n.Value.Value >= uint64(left.GetBitWidth()) {
			return newZero(left.GetBitWidth(), left.GetSignedness())
		}
	}
	// B4: lshl (<<<)
	if b.Operator == "<<<" {
		if n, ok := right.(*NumberExpression); ok && n.Value.Value >= uint64(left.GetBitWidth()) {
			return newZero(left.GetBitWidth(), left.GetSignedness())
		}
	}

	// 4) B5–B7：幺元/零元简化
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
		// 乘以 0 → 0
		if isZero(left) || isZero(right) {
			w := maxWidth(left, right)
			return newZero(w, left.GetSignedness() && right.GetSignedness())
		}
	case "/":
		if isOne(right) && left.GetBitWidth() == left.GetBitWidth() &&
			left.GetSignedness() == right.GetSignedness() {
			return left
		}
	case "&":
		// B10: E & (~E) → 0
		if u, ok := right.(*UnaryExpression); ok && u.Operator == "~" && u.Operand.GenerateString() == left.GenerateString() {
			w := left.GetBitWidth()
			return newZero(w, left.GetSignedness())
		}

	}

	// 5) 默认情况：构造一个新的节点，保留原操作符
	return &BinaryExpression{
		Left:     left,
		Right:    right,
		Operator: b.Operator,
	}
}

// --- UnaryExpression 等价变换 -------------------------------------------------

func (u *UnaryExpression) EquivalentTrans() Expression {
	if rand.Float64() > 0.5 {
		u.Operand = u.Operand.EquivalentTrans()
	}
	return u
}

// --- NumberExpression 等价变换 -----------------------------------------------

func (n *NumberExpression) EquivalentTrans() Expression {
	// 常数本身不拆分，直接返回
	return n
}

// --- VariableExpression 等价变换 --------------------------------------------

func (v *VariableExpression) EquivalentTrans() Expression {
	// 变量子范围也只是返回自己
	return v
}

// --- TernaryExpression 等价变换 ---------------------------------------------

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
	// 可以在此添加三元式的等价规则（如条件恒真/恒假消除）
	return t
}

// --- ConcatenationExpression 等价变换 ---------------------------------------

func (c *ConcatenationExpression) EquivalentTrans() Expression {
	for i, e := range c.Expressions {
		if rand.Float64() > 0.5 {
			c.Expressions[i] = e.EquivalentTrans()
		}
	}
	return c
}

// --- ReplicationExpression 等价变换 -----------------------------------------

func (r *ReplicationExpression) EquivalentTrans() Expression {
	if rand.Float64() > 0.5 {
		r.Expression = r.Expression.EquivalentTrans()
	}

	return r
}

// --- AssignExpression 等价变换 ---------------------------------------------

func (e *AssignExpression) EquivalentTrans() Expression {
	// 只对右侧表达式做等价变换
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

// BinaryExpression 的位宽和符号性实现
func (b *BinaryExpression) GetBitWidth() int {
	if b.isCtxSet {
		return b.ctxWidth
	}
	b.isCtxSet = true
	switch b.Operator {
	case "+", "-", "*", "/", "%", "&", "|", "^", "^~", "~^":
		// 取左右操作数位宽的最大值
		leftWidth := b.Left.GetBitWidth()
		rightWidth := b.Right.GetBitWidth()
		if leftWidth > rightWidth {
			b.ctxWidth = leftWidth
		} else {
			b.ctxWidth = rightWidth
		}
		return b.ctxWidth
	case "===", "!==", "==", "!=", ">", ">=", "<", "<=":
		// 比较运算符结果为1位
		b.ctxWidth = 1
		return 1
	case "&&", "||":
		// 逻辑运算符结果为1位
		b.ctxWidth = 1
		return 1
	case ">>", "<<", "**", ">>>", "<<<":
		// 移位运算符结果位宽与左操作数相同
		b.ctxWidth = b.Left.GetBitWidth()
		return b.ctxWidth
	default:
		// 默认取左操作数位宽
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
		// 比较和逻辑运算符结果总是无符号
		b.ctxSigned = false
		return false
	default:
		// 其他运算符：如果任一操作数是无符号的，结果就是无符号的
		b.ctxSigned = b.Left.GetSignedness() && b.Right.GetSignedness()
		return b.ctxSigned
	}
}

// UnaryExpression 的位宽和符号性实现
func (u *UnaryExpression) GetBitWidth() int {
	if u.isCtxSet {
		return u.ctxWidth
	}
	u.isCtxSet = true
	switch u.Operator {
	case "&", "~&", "|", "~|", "^", "~^", "^~", "!":
		// 归约运算符结果为1位
		u.ctxWidth = 1
		return 1
	default:
		// 其他一元运算符结果位宽与操作数相同
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
		// 归约运算符结果总是无符号
		u.ctxSigned = false
		return false
	default:
		u.ctxSigned = u.Operand.GetSignedness()
		// 其他一元运算符保持操作数的符号性
		return u.ctxSigned
	}
}

// TernaryExpression 的位宽和符号性实现
func (t *TernaryExpression) GetBitWidth() int {
	// 取真值和假值表达式的最大位宽
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
	// 如果任一操作数是无符号的，结果就是无符号的
	if t.isSignedSet {
		return t.ctxSigned
	}
	t.isSignedSet = true
	t.ctxSigned = t.TrueExpr.GetSignedness() && t.FalseExpr.GetSignedness()
	return t.ctxSigned
}

// ConcatenationExpression 的位宽和符号性实现
func (c *ConcatenationExpression) GetBitWidth() int {
	// 所有操作数位宽之和
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

// ReplicationExpression 的位宽和符号性实现
func (r *ReplicationExpression) GetBitWidth() int {
	if r.isCtxSet {
		return r.ctxWidth
	}
	r.isCtxSet = true
	// 重复次数 * 表达式的位宽
	count := r.Count.GetBitWidth()
	exprWidth := r.Expression.GetBitWidth()
	r.ctxWidth = count * exprWidth
	return r.ctxWidth
}

func (r *ReplicationExpression) GetSignedness() bool {
	// 重复拼接结果总是无符号的
	return false
}

// AssignExpression 的位宽和符号性实现
func (e *AssignExpression) GetBitWidth() int {
	return e.Right.GetBitWidth()
}

func (e *AssignExpression) GetSignedness() bool {
	return e.Right.GetSignedness()
}
