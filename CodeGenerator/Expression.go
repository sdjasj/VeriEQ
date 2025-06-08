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

// BinaryExpression 表示二元表达式的基本结构
type BinaryExpression struct {
	Left     Expression
	Right    Expression
	Operator string
}

// UnaryExpression 表示一元表达式的基本结构
type UnaryExpression struct {
	Operand  Expression
	Operator string
}

// VariableExpression 表示变量表达式
type VariableExpression struct {
	Var      *Variable
	Range    *BitRange
	hasRange bool
}

// NumberExpression 表示数字表达式
type NumberExpression struct {
	Value string
}

// TernaryExpression 表示三元运算符表达式
type TernaryExpression struct {
	Condition Expression
	TrueExpr  Expression
	FalseExpr Expression
}

// ConcatenationExpression 表示位拼接表达式
type ConcatenationExpression struct {
	Expressions []Expression
}

// ReplicationExpression 表示重复拼接表达式
type ReplicationExpression struct {
	Count      Expression // 重复次数
	Expression Expression // 要重复的表达式
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
	if v.hasRange && rand.Float64() < 0.8 {
		r := v.Range.l + rand.Intn(v.Range.r-v.Range.l+1)
		l := v.Range.l + rand.Intn(r-v.Range.l+1)
		return fmt.Sprintf("%s[%d:%d]", v.Var.Name, r, l)
	}
	return v.Var.Name
}

func (n *NumberExpression) GenerateString() string {
	return n.Value
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

// 实现各种表达式的EquivalentTrans方法
func (b *BinaryExpression) EquivalentTrans() Expression {
	// 根据运算符类型进行等价变换
	switch b.Operator {
	// 算术运算符
	case "+":
		// a + b = b + a (交换律)
		return &BinaryExpression{
			Left:     b.Right,
			Right:    b.Left,
			Operator: "+",
		}
	case "-":
		// a - b = -(b - a)
		return &UnaryExpression{
			Operand: &BinaryExpression{
				Left:     b.Right,
				Right:    b.Left,
				Operator: "-",
			},
			Operator: "-",
		}
	case "*":
		// a * b = b * a (交换律)
		return &BinaryExpression{
			Left:     b.Right,
			Right:    b.Left,
			Operator: "*",
		}
	case "/":
		// 除法没有交换律
		return b
	case "%":
		// 取模没有交换律
		return b

	// 逻辑运算符
	case "&&":
		// a && b = b && a (交换律)
		return &BinaryExpression{
			Left:     b.Right,
			Right:    b.Left,
			Operator: "&&",
		}
	case "||":
		// a || b = b || a (交换律)
		return &BinaryExpression{
			Left:     b.Right,
			Right:    b.Left,
			Operator: "||",
		}

	// 位运算符
	case "&":
		// a & b = b & a (交换律)
		return &BinaryExpression{
			Left:     b.Right,
			Right:    b.Left,
			Operator: "&",
		}
	case "|":
		// a | b = b | a (交换律)
		return &BinaryExpression{
			Left:     b.Right,
			Right:    b.Left,
			Operator: "|",
		}
	case "^":
		// a ^ b = b ^ a (交换律)
		return &BinaryExpression{
			Left:     b.Right,
			Right:    b.Left,
			Operator: "^",
		}
	case "~^":
		// a ~^ b = b ~^ a (交换律)
		return &BinaryExpression{
			Left:     b.Right,
			Right:    b.Left,
			Operator: "~^",
		}

	// 移位运算符
	case "<<":
		// 左移没有交换律
		return b
	case ">>":
		// 右移没有交换律
		return b

	// 比较运算符
	case "==":
		// a == b = b == a (交换律)
		return &BinaryExpression{
			Left:     b.Right,
			Right:    b.Left,
			Operator: "==",
		}
	case "!=":
		// a != b = b != a (交换律)
		return &BinaryExpression{
			Left:     b.Right,
			Right:    b.Left,
			Operator: "!=",
		}
	case "<":
		// a < b = b > a
		return &BinaryExpression{
			Left:     b.Right,
			Right:    b.Left,
			Operator: ">",
		}
	case "<=":
		// a <= b = b >= a
		return &BinaryExpression{
			Left:     b.Right,
			Right:    b.Left,
			Operator: ">=",
		}
	case ">":
		// a > b = b < a
		return &BinaryExpression{
			Left:     b.Right,
			Right:    b.Left,
			Operator: "<",
		}
	case ">=":
		// a >= b = b <= a
		return &BinaryExpression{
			Left:     b.Right,
			Right:    b.Left,
			Operator: "<=",
		}

	default:
		return b
	}
}

func (u *UnaryExpression) EquivalentTrans() Expression {
	switch u.Operator {
	case "!":
		// 双重否定: !!a = a
		if unary, ok := u.Operand.(*UnaryExpression); ok && unary.Operator == "!" {
			return &UnaryExpression{
				Operand:  unary.Operand,
				Operator: "|",
			}
		}
	case "~":
		// 双重取反: ~~a = a
		if unary, ok := u.Operand.(*UnaryExpression); ok && unary.Operator == "~" {
			return unary.Operand
		}
	case "-":
		// 双重负号: --a = a
		if unary, ok := u.Operand.(*UnaryExpression); ok && unary.Operator == "-" {
			return unary.Operand
		}
	}
	return u
}

func (v *VariableExpression) EquivalentTrans() Expression {
	return v
}

func (n *NumberExpression) EquivalentTrans() Expression {
	return n
}

func (t *TernaryExpression) EquivalentTrans() Expression {
	// 如果条件为常量，直接返回对应的表达式
	if num, ok := t.Condition.(*NumberExpression); ok {
		if num.Value == "1" || num.Value == "1'b1" {
			return t.TrueExpr
		}
		if num.Value == "0" || num.Value == "1'b0" {
			return t.FalseExpr
		}
	}
	// 如果条件是一个一元表达式，尝试简化
	if unary, ok := t.Condition.(*UnaryExpression); ok {
		if unary.Operator == "!" {
			// !a ? b : c = a ? c : b
			return &TernaryExpression{
				Condition: unary.Operand,
				TrueExpr:  t.FalseExpr,
				FalseExpr: t.TrueExpr,
			}
		}
	}
	return t
}

func (c *ConcatenationExpression) EquivalentTrans() Expression {
	// 位拼接表达式的等价变换
	// 1. 重新排序拼接项
	// 2. 合并相邻的常量
	// 3. 展开嵌套的拼接
	exprs := make([]Expression, len(c.Expressions))
	for i, expr := range c.Expressions {
		exprs[i] = expr.EquivalentTrans()
	}
	return &ConcatenationExpression{
		Expressions: exprs,
	}
}

func (r *ReplicationExpression) EquivalentTrans() Expression {
	// 重复拼接表达式的等价变换
	// 1. 如果重复次数为1，直接返回表达式
	if num, ok := r.Count.(*NumberExpression); ok {
		if num.Value == "1" || num.Value == "1'b1" {
			return r.Expression
		}
	}
	// 2. 如果重复次数为0，返回空拼接
	if num, ok := r.Count.(*NumberExpression); ok {
		if num.Value == "0" || num.Value == "1'b0" {
			return &ConcatenationExpression{
				Expressions: []Expression{},
			}
		}
	}
	return r
}
