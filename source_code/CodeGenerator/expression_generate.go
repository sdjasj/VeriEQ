package CodeGenerator

import (
	"fmt"
	"strings"
)

func (e *AssignExpression) GenerateString() string {
	if e.UsedRange == nil {
		return fmt.Sprintf("    assign %s = %s;", e.Operand1.Name, e.Right.GenerateString())
	}
	return fmt.Sprintf("    assign %s[%d:%d] = %s;", e.Operand1.Name, e.UsedRange.r, e.UsedRange.l,
		e.Right.GenerateString())
}

func (b *BinaryExpression) GenerateString() string {
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
