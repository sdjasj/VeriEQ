package CodeGenerator

import (
	"math/rand"
)

func (g *ExpressionGenerator) GenerateExpressionFromPool(depth int, pool []*Variable, depthMap map[*Variable]int, defs map[*Variable]Expression) Expression {
	if depth <= 0 {
		return g.generateBasicExpressionFromPool(pool, depthMap, defs)
	}

	exprType := rand.Intn(6)

	switch exprType {
	case 0:
		return g.generateBinaryExpressionFromPool(depth, pool, depthMap, defs)
	case 1:
		return g.generateUnaryExpressionFromPool(depth, pool, depthMap, defs)
	case 2:
		return g.generateTernaryExpressionFromPool(depth, pool, depthMap, defs)
	case 3:
		return g.generateConcatenationExpressionFromPool(depth, pool, depthMap, defs)
	case 4:
		return g.generateVariableExpressionFromPool(pool, depthMap, defs)
	case 5:
		return g.generateNumberExpression()
	default:
		return g.generateBasicExpressionFromPool(pool, depthMap, defs)
	}
}

func (g *ExpressionGenerator) generateBasicExpressionFromPool(pool []*Variable, depthMap map[*Variable]int, defs map[*Variable]Expression) Expression {
	if rand.Intn(2) == 0 && len(pool) > 0 {
		return g.generateVariableExpressionFromPool(pool, depthMap, defs)
	}
	return g.generateNumberExpression()
}

func (g *ExpressionGenerator) generateBinaryExpressionFromPool(depth int, pool []*Variable, depthMap map[*Variable]int, defs map[*Variable]Expression) Expression {
	operators := []string{
		"+", "-", "*", "/", "%",
		"&&", "||",
		"&", "|", "^", "~&", "~|",
		"==", "!=", "===", "!==", "<", "<=", ">", ">=",
		"<<", ">>", "<<<", ">>>",
		"<<", ">>", "<<<", ">>>",
	}
	operator := operators[rand.Intn(len(operators))]

	left := g.GenerateExpressionFromPool(depth-1, pool, depthMap, defs)
	var right Expression
	if operator == "<<" || operator == ">>" || operator == "<<<" || operator == ">>>" {
		right = g.generateShiftAmountExpression(left)
	} else {
		right = g.GenerateExpressionFromPool(depth-1, pool, depthMap, defs)
	}
	if operator == "/" || operator == "%" {
		right = &ConcatenationExpression{
			Expressions: []Expression{
				&NumberExpression{
					Value: ConstNumber{
						Value:      1,
						BitWidth:   1,
						Signedness: false,
					},
				},
				right,
			},
		}
	}

	return &BinaryExpression{
		Left:     left,
		Right:    right,
		Operator: operator,
	}
}

func (g *ExpressionGenerator) generateUnaryExpressionFromPool(depth int, pool []*Variable, depthMap map[*Variable]int, defs map[*Variable]Expression) Expression {
	operators := []string{"!", "~", "-"}

	operator := operators[rand.Intn(len(operators))]
	operand := g.GenerateExpressionFromPool(depth-1, pool, depthMap, defs)

	return &UnaryExpression{
		Operand:  operand,
		Operator: operator,
	}
}

func (g *ExpressionGenerator) generateTernaryExpressionFromPool(depth int, pool []*Variable, depthMap map[*Variable]int, defs map[*Variable]Expression) Expression {
	condition := g.GenerateExpressionFromPool(depth-1, pool, depthMap, defs)
	trueExpr := g.GenerateExpressionFromPool(depth-1, pool, depthMap, defs)
	falseExpr := g.GenerateExpressionFromPool(depth-1, pool, depthMap, defs)

	return &TernaryExpression{
		Condition: condition,
		TrueExpr:  trueExpr,
		FalseExpr: falseExpr,
	}
}

func (g *ExpressionGenerator) generateVariableExpressionFromPool(pool []*Variable, depthMap map[*Variable]int, defs map[*Variable]Expression) Expression {
	if len(pool) == 0 {
		return g.generateNumberExpression()
	}

	variable := weightedPickVar(pool, depthMap)
	if variable == nil {
		return g.generateNumberExpression()
	}

	if expr, ok := defs[variable]; ok && rand.Float64() < 0.4 {
		return cloneExpression(expr)
	}

	if variable.hasRange && rand.Float64() > 0.8 {
		r := variable.Range.l + rand.Intn(variable.Range.r-variable.Range.l+1)
		l := variable.Range.l + rand.Intn(r-variable.Range.l+1)
		return &VariableExpression{
			Var:      variable,
			hasRange: true,
			Range: &BitRange{
				l: l,
				r: r,
			},
		}
	}

	return &VariableExpression{
		Var:      variable,
		hasRange: false,
	}
}

func (g *ExpressionGenerator) generateConcatenationExpressionFromPool(depth int, pool []*Variable, depthMap map[*Variable]int, defs map[*Variable]Expression) Expression {
	numExprs := rand.Intn(3) + 2
	exprs := make([]Expression, 0, numExprs)

	for i := 0; i < numExprs; i++ {
		if rand.Float64() < -1 {
			count := rand.Intn(8) + 1
			countExpr := &NumberExpression{
				Value: RandomConstNumberWithBitWidth(count, false),
			}
			expr := g.GenerateExpressionFromPool(depth-1, pool, depthMap, defs)
			exprs = append(exprs, &ReplicationExpression{
				Count:      countExpr,
				Expression: expr,
			})
		} else {
			expr := g.GenerateExpressionFromPool(depth-1, pool, depthMap, defs)
			exprs = append(exprs, expr)
		}
	}

	return &ConcatenationExpression{
		Expressions: exprs,
	}
}

func weightedPickVar(pool []*Variable, depthMap map[*Variable]int) *Variable {
	if len(pool) == 0 {
		return nil
	}
	total := 0.0
	for _, v := range pool {
		d := depthMap[v]
		total += 1.0 / (1.0 + float64(d))
	}
	if total <= 0 {
		return pool[rand.Intn(len(pool))]
	}
	r := rand.Float64() * total
	for _, v := range pool {
		d := depthMap[v]
		w := 1.0 / (1.0 + float64(d))
		if r <= w {
			return v
		}
		r -= w
	}
	return pool[len(pool)-1]
}
