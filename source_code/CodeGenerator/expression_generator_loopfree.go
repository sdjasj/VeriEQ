package CodeGenerator

func (g *ExpressionGenerator) GenerateLoopFreeExpression(depth int) Expression {
	return g.GenerateExpression(depth)

}

func (g *ExpressionGenerator) GenerateLoopFreeAssignment(target *Variable) *AssignExpression {

	if len(g.CurrentDefinedVars) == 0 {
		return nil
	}

	expr := g.GenerateLoopFreeExpression(g.MaxDepth)
	if target == nil {
		target = g.AddWireVariable("")
	}
	//g.CurrentDefinedVars = append(g.CurrentDefinedVars, target)

	return &AssignExpression{
		Operand1: target,
		Right:    expr,
	}
}

func (g *ExpressionGenerator) GenerateLoopFreeOutputAssignment(target *Variable) *AssignExpression {

	if len(g.CurrentDefinedVars) == 0 {
		return nil
	}

	expr := g.GenerateLoopFreeExpression(g.MaxDepth)
	if target == nil {
		target = g.AddWireVariable("")
	}

	return &AssignExpression{
		Operand1: target,
		Right:    expr,
	}
}

func (g *ExpressionGenerator) Clear() {
	g.CurrentDefinedVars = make([]*Variable, 0)
	g.OutputVars = make([]*Variable, 0)
	g.InputVars = make([]*Variable, 0)
	g.ClockVars = make([]*Variable, 0)

}
