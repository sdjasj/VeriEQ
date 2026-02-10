package CodeGenerator

func cloneExpression(expr Expression) Expression {
	switch e := expr.(type) {
	case *BinaryExpression:
		return &BinaryExpression{
			Left:        cloneExpression(e.Left),
			Right:       cloneExpression(e.Right),
			Operator:    e.Operator,
			ctxWidth:    e.ctxWidth,
			ctxSigned:   e.ctxSigned,
			isCtxSet:    e.isCtxSet,
			isSignedSet: e.isSignedSet,
			realWidth:   e.realWidth,
			realSigned:  e.realSigned,
		}
	case *UnaryExpression:
		return &UnaryExpression{
			Operand:     cloneExpression(e.Operand),
			Operator:    e.Operator,
			ctxWidth:    e.ctxWidth,
			ctxSigned:   e.ctxSigned,
			isCtxSet:    e.isCtxSet,
			isSignedSet: e.isSignedSet,
			realWidth:   e.realWidth,
			realSigned:  e.realSigned,
		}
	case *VariableExpression:
		var rng *BitRange
		if e.hasRange && e.Range != nil {
			rng = &BitRange{l: e.Range.l, r: e.Range.r}
		}
		return &VariableExpression{
			Var:         e.Var,
			Range:       rng,
			hasRange:    e.hasRange,
			ctxWidth:    e.ctxWidth,
			ctxSigned:   e.ctxSigned,
			isCtxSet:    e.isCtxSet,
			isSignedSet: e.isSignedSet,
			realWidth:   e.realWidth,
			realSigned:  e.realSigned,
		}
	case *NumberExpression:
		return &NumberExpression{
			Value:       e.Value,
			ctxWidth:    e.ctxWidth,
			ctxSigned:   e.ctxSigned,
			isCtxSet:    e.isCtxSet,
			isSignedSet: e.isSignedSet,
			realWidth:   e.realWidth,
			realSigned:  e.realSigned,
		}
	case *TernaryExpression:
		return &TernaryExpression{
			Condition:   cloneExpression(e.Condition),
			TrueExpr:    cloneExpression(e.TrueExpr),
			FalseExpr:   cloneExpression(e.FalseExpr),
			ctxWidth:    e.ctxWidth,
			ctxSigned:   e.ctxSigned,
			isCtxSet:    e.isCtxSet,
			isSignedSet: e.isSignedSet,
			realWidth:   e.realWidth,
			realSigned:  e.realSigned,
		}
	case *ConcatenationExpression:
		parts := make([]Expression, 0, len(e.Expressions))
		for _, part := range e.Expressions {
			parts = append(parts, cloneExpression(part))
		}
		return &ConcatenationExpression{
			Expressions: parts,
			ctxWidth:    e.ctxWidth,
			ctxSigned:   e.ctxSigned,
			isCtxSet:    e.isCtxSet,
			isSignedSet: e.isSignedSet,
			realWidth:   e.realWidth,
			realSigned:  e.realSigned,
		}
	case *ReplicationExpression:
		return &ReplicationExpression{
			Count:       cloneExpression(e.Count),
			Expression:  cloneExpression(e.Expression),
			ctxWidth:    e.ctxWidth,
			ctxSigned:   e.ctxSigned,
			isCtxSet:    e.isCtxSet,
			isSignedSet: e.isSignedSet,
			realWidth:   e.realWidth,
			realSigned:  e.realSigned,
		}
	default:
		return expr
	}
}

func cloneAssignExpressions(assigns []*AssignExpression) []*AssignExpression {
	cloned := make([]*AssignExpression, 0, len(assigns))
	for _, assign := range assigns {
		var rng *BitRange
		if assign.UsedRange != nil {
			rng = &BitRange{l: assign.UsedRange.l, r: assign.UsedRange.r}
		}
		cloned = append(cloned, &AssignExpression{
			Operand1:   assign.Operand1,
			Right:      cloneExpression(assign.Right),
			UsedRange:  rng,
			realWidth:  assign.realWidth,
			realSigned: assign.realSigned,
		})
	}
	return cloned
}

func (g *ExpressionGenerator) ApplyEquivalenceTransforms(assigns []*AssignExpression) []*AssignExpression {
	base := cloneAssignExpressions(assigns)
	transformed := make([]*AssignExpression, len(base))
	for i, a := range base {
		transformed[i] = a.EquivalentTrans().(*AssignExpression)
	}
	return transformed
}
