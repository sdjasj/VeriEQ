package CodeGenerator

import (
	"fmt"
	"math"
	"math/rand"
	"strconv"
)

func (g *ExpressionGenerator) GenerateExpression(depth int) Expression {

	if depth <= 0 {
		return g.generateBasicExpression()
	}

	exprType := rand.Intn(6)

	switch exprType {
	case 0:
		// 生成二元表达式
		return g.generateBinaryExpression(depth)
	case 1:
		// 生成一元表达式
		return g.generateUnaryExpression(depth)
	case 2:
		// 生成三元表达式
		return g.generateTernaryExpression(depth)
	case 3:
		// 生成位拼接表达式
		return g.generateConcatenationExpression(depth)
	case 4:
		// 生成变量表达式
		return g.generateVariableExpression()
	case 5:
		// 生成数字表达式
		return g.generateNumberExpression()

	default:
		return g.generateBasicExpression()
	}
}

func (g *ExpressionGenerator) generateBasicExpression() Expression {
	if rand.Intn(2) == 0 && len(g.CurrentDefinedVars) > 0 {
		return g.generateVariableExpression()
	}
	return g.generateNumberExpression()
}

func (g *ExpressionGenerator) generateBinaryExpression(depth int) Expression {
	operators := []string{
		"+", "-", "*", "/", "%",
		"&&", "||",
		"&", "|", "^", "~&", "~|",
		"==", "!=", "===", "!==", "<", "<=", ">", ">=",
		"<<", ">>", "<<<", ">>>",
	}
	operator := operators[rand.Intn(len(operators))]

	left := g.GenerateExpression(depth - 1)
	var right Expression
	if operator == "<<" || operator == ">>" || operator == "<<<" || operator == ">>>" {
		right = g.generateShiftAmountExpression(left)
	} else {
		right = g.GenerateExpression(depth - 1)
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

func (g *ExpressionGenerator) generateShiftAmountExpression(left Expression) Expression {
	const maxShiftAmount = 31

	limit := maxShiftAmount
	if left != nil {
		leftWidth := left.GetBitWidth()
		if leftWidth < limit {
			limit = leftWidth
		}
	}
	if limit < 0 {
		limit = 0
	}
	value := 0
	if limit > 0 {
		value = rand.Intn(limit + 1)
	}
	width := bitsNeeded(value)
	return &NumberExpression{
		Value: ConstNumber{
			Value:      uint64(value),
			BitWidth:   width,
			Signedness: false,
		},
	}
}

func (g *ExpressionGenerator) generateUnaryExpression(depth int) Expression {
	operators := []string{"!", "~", "-"}

	operator := operators[rand.Intn(len(operators))]

	operand := g.GenerateExpression(depth - 1)

	return &UnaryExpression{
		Operand:  operand,
		Operator: operator,
	}
}

func (g *ExpressionGenerator) generateTernaryExpression(depth int) Expression {
	condition := g.GenerateExpression(depth - 1)
	trueExpr := g.GenerateExpression(depth - 1)
	falseExpr := g.GenerateExpression(depth - 1)

	return &TernaryExpression{
		Condition: condition,
		TrueExpr:  trueExpr,
		FalseExpr: falseExpr,
	}
}

func (g *ExpressionGenerator) generateVariableExpression() Expression {

	if len(g.CurrentDefinedVars) == 0 {
		// 如果没有可用变量，生成一个数字表达式
		return g.generateNumberExpression()
	}

	variable := g.CurrentDefinedVars[rand.Intn(len(g.CurrentDefinedVars))]
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

func bitsNeeded(value int) int {
	if value == 0 {
		return 1
	}
	return int(math.Floor(math.Log2(float64(value)))) + 1
}

func parseInt(valueStr string, base int) int {
	val, _ := strconv.ParseInt(valueStr, base, 64)
	return int(val)
}

func GenerateRandomNumber() string {
	format := rand.Intn(4)

	var numStr string
	switch format {
	case 0:
		value := rand.Intn(1000)
		width := bitsNeeded(value)
		numStr = fmt.Sprintf("%d'd%d", width, value)

	case 1:
		length := rand.Intn(10) + 1 // 1 到 10 位
		binStr := ""
		for i := 0; i < length; i++ {
			binStr += fmt.Sprintf("%d", rand.Intn(2))
		}
		numStr = fmt.Sprintf("%d'b%s", length+rand.Intn(6), binStr)

	case 2:
		digits := rand.Intn(4) + 1
		octStr := ""
		for i := 0; i < digits; i++ {
			octStr += fmt.Sprintf("%o", rand.Intn(8))
		}
		value := parseInt(octStr, 8)
		width := bitsNeeded(value)
		numStr = fmt.Sprintf("%d'o%s", width, octStr)

	case 3:
		digits := rand.Intn(3) + 1
		hexStr := ""
		for i := 0; i < digits; i++ {
			hexStr += fmt.Sprintf("%x", rand.Intn(16))
		}
		value := parseInt(hexStr, 16)
		width := bitsNeeded(value)
		numStr = fmt.Sprintf("%d'h%s", width, hexStr)
	}
	return numStr
}

func (g *ExpressionGenerator) generateNumberExpression() Expression {
	return &NumberExpression{
		Value: RandomConstNumber(),
	}
}

func (g *ExpressionGenerator) generateConcatenationExpression(depth int) Expression {
	numExprs := rand.Intn(3) + 2
	exprs := make([]Expression, 0, numExprs)

	for i := 0; i < numExprs; i++ {
		if rand.Float64() < -1 {
			count := rand.Intn(8) + 1
			countExpr := &NumberExpression{
				Value: RandomConstNumberWithBitWidth(count, false),
			}
			expr := g.GenerateExpression(depth - 1)
			exprs = append(exprs, &ReplicationExpression{
				Count:      countExpr,
				Expression: expr,
			})
		} else {
			expr := g.GenerateExpression(depth - 1)
			exprs = append(exprs, expr)
		}
	}

	return &ConcatenationExpression{
		Expressions: exprs,
	}
}
