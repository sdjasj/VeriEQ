package CodeGenerator

import "math/bits"

func isZero(e Expression) bool {
	if n, ok := e.(*NumberExpression); ok {
		return n.Value.Value == 0
	}
	return false
}

func isOne(e Expression) bool {
	if n, ok := e.(*NumberExpression); ok {
		if n.Value.Signedness && n.Value.BitWidth == 1 {
			return false
		}
		return n.Value.Value == 1
	}
	return false
}

func newZero(width int, signed bool) Expression {
	if width <= 0 {
		width = 1
	}
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

func newConst(value uint64, width int, signed bool) *NumberExpression {
	cn := ConstNumber{
		Value:      value,
		BitWidth:   width,
		Signedness: signed,
	}
	return &NumberExpression{
		Value:       cn,
		ctxWidth:    width,
		ctxSigned:   signed,
		isCtxSet:    true,
		isSignedSet: true,
		realWidth:   width,
		realSigned:  signed,
	}
}

func allOnesValue(width int) uint64 {
	if width <= 0 {
		return 0
	}
	if width >= 64 {
		return ^uint64(0)
	}
	return (uint64(1) << uint(width)) - 1
}

func bitWidthForValue(value uint64) int {
	if value == 0 {
		return 1
	}
	return bits.Len64(value)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func binaryResultWidth(op string, lw, rw int) int {
	if lw <= 0 {
		lw = 1
	}
	if rw <= 0 {
		rw = 1
	}
	switch op {
	case "+", "-":
		return maxInt(lw, rw) + 1
	case "*":
		return lw + rw
	case "/", "%":
		return lw
	case "<<", ">>", "<<<", ">>>":
		return lw
	case "**":
		return lw
	case "==", "!=", "===", "!==", ">", ">=", "<", "<=", "&&", "||":
		return 1
	case "&", "|", "^", "^~", "~^":
		return maxInt(lw, rw)
	default:
		return maxInt(lw, rw)
	}
}

func maxWidth(a, b Expression) int {
	wa := effectiveWidth(a)
	wb := effectiveWidth(b)
	if wa > wb {
		return wa
	}
	return wb
}

func effectiveWidth(e Expression) int {
	if w := e.GetRealBitWidth(); w > 0 {
		return w
	}
	return e.GetBitWidth()
}

func effectiveSignedness(e Expression) bool {
	if e.GetRealBitWidth() > 0 {
		return e.GetRealSignedness()
	}
	return e.GetSignedness()
}
