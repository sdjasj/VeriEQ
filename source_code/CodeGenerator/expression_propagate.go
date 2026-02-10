package CodeGenerator

// PropagateType methods
func (b *BinaryExpression) PropagateType(width int, signed bool) {
	lw := b.Left.GetBitWidth()
	rw := b.Right.GetBitWidth()
	ls := b.Left.GetSignedness()
	rs := b.Right.GetSignedness()

	switch b.Operator {
	case "==", "!=", "===", "!==", ">", ">=", "<", "<=":
		maxw := maxInt(lw, rw)
		mergedSigned := ls && rs
		b.realWidth = 1
		b.realSigned = false
		b.Left.PropagateType(maxw, mergedSigned)
		b.Right.PropagateType(maxw, mergedSigned)
	case "&&", "||":
		b.realWidth = 1
		b.realSigned = false
		b.Left.PropagateType(lw, ls)
		b.Right.PropagateType(rw, rs)
	case ">>", "<<", ">>>", "<<<", "**":
		exprWidth := lw
		if width > exprWidth {
			exprWidth = width
		}
		b.realWidth = exprWidth
		b.realSigned = ls
		b.Left.PropagateType(lw, ls)
		b.Right.PropagateType(rw, rs) // Right is self-determined
	default:
		exprWidth := binaryResultWidth(b.Operator, lw, rw)
		if width > exprWidth {
			exprWidth = width
		}
		exprSigned := ls && rs
		b.realWidth = exprWidth
		b.realSigned = exprSigned
		opWidth := maxInt(lw, rw)
		b.Left.PropagateType(opWidth, exprSigned)
		b.Right.PropagateType(opWidth, exprSigned)
	}
}

func (u *UnaryExpression) PropagateType(width int, signed bool) {
	switch u.Operator {
	case "&", "~&", "|", "~|", "^", "~^", "^~", "!":
		u.realWidth = 1
		u.realSigned = false
		u.Operand.PropagateType(u.Operand.GetBitWidth(), u.Operand.GetSignedness())
	case "+", "-", "~":
		u.realWidth = width
		u.realSigned = signed
		u.Operand.PropagateType(width, signed)
	default:
		u.realWidth = width
		u.realSigned = signed
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
	totalWidth := 0
	for i := 0; i < len(c.Expressions); i++ {
		totalWidth += c.Expressions[i].GetBitWidth()
	}
	c.realWidth = totalWidth
	c.realSigned = false
	for i := 0; i < len(c.Expressions); i++ {
		c.Expressions[i].PropagateType(c.Expressions[i].GetBitWidth(), c.Expressions[i].GetSignedness()) // self-determined
	}
}

func (r *ReplicationExpression) PropagateType(width int, signed bool) {
	countExpr, ok := r.Count.(*NumberExpression)
	if ok {
		r.realWidth = int(countExpr.Value.Value) * r.Expression.GetBitWidth()
	} else {
		r.realWidth = r.Expression.GetBitWidth()
	}
	r.realSigned = false
	r.Count.PropagateType(r.Count.GetBitWidth(), r.Count.GetSignedness())
	r.Expression.PropagateType(r.Expression.GetBitWidth(), r.Expression.GetSignedness())
}

func (e *AssignExpression) PropagateType(width int, signed bool) {
	if e.UsedRange != nil {
		width = e.UsedRange.GetWidth()
	} else {
		width = e.Operand1.GetWidth()
	}
	signed = e.Right.GetSignedness()
	e.realWidth = width
	e.realSigned = signed
	e.Right.PropagateType(width, signed)
}
