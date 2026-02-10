package CodeGenerator

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
