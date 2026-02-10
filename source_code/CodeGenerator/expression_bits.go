package CodeGenerator

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
	default:
		lw := b.Left.GetBitWidth()
		rw := b.Right.GetBitWidth()
		b.ctxWidth = binaryResultWidth(b.Operator, lw, rw)
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
	case ">>", "<<", ">>>", "<<<":
		b.ctxSigned = b.Left.GetSignedness()
		return b.ctxSigned
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
