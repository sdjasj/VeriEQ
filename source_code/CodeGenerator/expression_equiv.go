package CodeGenerator

import "math/rand"

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
		if n, ok := right.(*NumberExpression); ok && n.Value.Value >= uint64(effectiveWidth(b)) {
			return newZero(effectiveWidth(b), effectiveSignedness(b))
		}
	}

	if b.Operator == ">>>" {
		if n, ok := right.(*NumberExpression); ok &&
			n.Value.Value >= uint64(effectiveWidth(b)) &&
			!effectiveSignedness(b) {
			//if left.GetRealBitWidth() == 0 {
			//	fmt.Println(left.GenerateString()+"fuck !!!!!!!!!!! ")
			//}
			return newZero(effectiveWidth(b), false)
		}
	}

	if b.Operator == "<<" {
		if n, ok := right.(*NumberExpression); ok && n.Value.Value >= uint64(effectiveWidth(b)) {
			return newZero(effectiveWidth(b), effectiveSignedness(b))
		}
	}

	if b.Operator == "<<<" {
		if n, ok := right.(*NumberExpression); ok && n.Value.Value >= uint64(effectiveWidth(b)) {
			return newZero(effectiveWidth(b), effectiveSignedness(b))
		}
	}

	switch b.Operator {
	case "+":
		if isZero(right) && right.GetBitWidth() == left.GetBitWidth() &&
			right.GetSignedness() == left.GetSignedness() {
			return left
		}
		if isZero(left) && left.GetBitWidth() == right.GetBitWidth() &&
			left.GetSignedness() == right.GetSignedness() {
			return right
		}
	case "*":
		if isOne(right) && right.GetBitWidth() == left.GetBitWidth() &&
			right.GetSignedness() == left.GetSignedness() {
			return left
		}
		if isOne(left) && left.GetBitWidth() == right.GetBitWidth() &&
			left.GetSignedness() == right.GetSignedness() {
			return right
		}

		if isZero(left) || isZero(right) {
			w := maxWidth(left, right)
			return newZero(w, left.GetRealSignedness() && right.GetRealSignedness())
		}
	case "/":
		if isOne(right) && left.GetBitWidth() == right.GetBitWidth() &&
			left.GetSignedness() == right.GetSignedness() {
			return left
		}
	case "&":
		if u, ok := right.(*UnaryExpression); ok && u.Operator == "~" && u.Operand.GenerateString() == left.GenerateString() {
			w := effectiveWidth(left)
			return newZero(w, effectiveSignedness(left))
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
	width := n.Value.BitWidth
	signed := n.Value.Signedness
	if width <= 0 {
		return n
	}
	ctxWidth := effectiveWidth(n)
	ctxSigned := effectiveSignedness(n)
	allowWidthSensitive := ctxWidth == width && ctxSigned == signed

	mask := allOnesValue(width)
	value := n.Value.Value & mask

	candidates := []Expression{n}
	zero := newConst(0, width, signed)
	one := newConst(1, width, signed)

	candidates = append(candidates, &BinaryExpression{Left: n, Right: zero, Operator: "+"})
	candidates = append(candidates, &BinaryExpression{Left: zero, Right: n, Operator: "+"})
	if !(signed && width == 1) {
		candidates = append(candidates, &BinaryExpression{Left: n, Right: one, Operator: "*"})
		candidates = append(candidates, &BinaryExpression{Left: one, Right: n, Operator: "*"})
		candidates = append(candidates, &BinaryExpression{Left: n, Right: one, Operator: "/"})
	}

	if !signed {
		unsignedZero := newConst(0, width, false)
		unsignedOnes := newConst(allOnesValue(width), width, false)
		candidates = append(candidates, &BinaryExpression{Left: n, Right: unsignedZero, Operator: "|"})
		if allowWidthSensitive {
			candidates = append(candidates, &BinaryExpression{Left: n, Right: unsignedOnes, Operator: "&"})
		}
	}

	if value == 0 {
		left := newConst(0, width, signed)
		candidates = append(candidates, &BinaryExpression{
			Left:     left,
			Right:    &UnaryExpression{Operator: "~", Operand: newConst(0, width, signed)},
			Operator: "&",
		})

		shift := width + rand.Intn(3)
		shiftVal := uint64(shift)
		shiftExpr := newConst(shiftVal, bitWidthForValue(shiftVal), false)
		shiftLeft := newConst(1, width, signed)
		candidates = append(candidates, &BinaryExpression{Left: shiftLeft, Right: shiftExpr, Operator: "<<"})
		candidates = append(candidates, &BinaryExpression{Left: shiftLeft, Right: shiftExpr, Operator: ">>"})
		candidates = append(candidates, &BinaryExpression{Left: shiftLeft, Right: shiftExpr, Operator: "<<<"})
		if !signed {
			candidates = append(candidates, &BinaryExpression{Left: shiftLeft, Right: shiftExpr, Operator: ">>>"})
		}
	}

	return candidates[rand.Intn(len(candidates))]
}

func (v *VariableExpression) EquivalentTrans() Expression {
	width := v.GetBitWidth()
	signed := v.GetSignedness()
	if width <= 0 {
		return v
	}
	ctxWidth := effectiveWidth(v)
	ctxSigned := effectiveSignedness(v)
	allowWidthSensitive := ctxWidth == width && ctxSigned == signed

	candidates := []Expression{v}
	zero := newConst(0, width, signed)
	one := newConst(1, width, signed)

	candidates = append(candidates, &BinaryExpression{Left: v, Right: zero, Operator: "+"})
	candidates = append(candidates, &BinaryExpression{Left: zero, Right: v, Operator: "+"})
	if !(signed && width == 1) {
		candidates = append(candidates, &BinaryExpression{Left: v, Right: one, Operator: "*"})
		candidates = append(candidates, &BinaryExpression{Left: one, Right: v, Operator: "*"})
		candidates = append(candidates, &BinaryExpression{Left: v, Right: one, Operator: "/"})
	}

	if !signed {
		unsignedZero := newConst(0, width, false)
		candidates = append(candidates, &BinaryExpression{Left: v, Right: unsignedZero, Operator: "|"})
		if allowWidthSensitive {
			unsignedOnes := newConst(allOnesValue(width), width, false)
			candidates = append(candidates, &BinaryExpression{Left: v, Right: unsignedOnes, Operator: "&"})
		}
	}

	return candidates[rand.Intn(len(candidates))]
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
