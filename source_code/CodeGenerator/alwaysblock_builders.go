package CodeGenerator

import "math/rand"

func NewAlwaysBlock(blockType AlwaysBlockType) *AlwaysBlock {
	return &AlwaysBlock{
		Type:       blockType,
		Statements: make([]Statement, 0),
	}
}

func (a *AlwaysBlock) SetClock(clockVar *Variable) {
	a.ClockVars = []*Variable{clockVar}
	a.ClockPosedge = []bool{a.pickClockEdge()}
}

func (a *AlwaysBlock) AddClock(clockVar *Variable) {
	a.ClockVars = append(a.ClockVars, clockVar)
	a.ClockPosedge = append(a.ClockPosedge, a.pickClockEdge())
}

func (a *AlwaysBlock) SetClocks(clockVars []*Variable) {
	a.ClockVars = clockVars
	a.ClockPosedge = make([]bool, len(clockVars))
	for i := range clockVars {
		a.ClockPosedge[i] = a.pickClockEdge()
	}
}

func (a *AlwaysBlock) SetReset(resetVar *Variable, resetValue string) {
	a.ResetVar = resetVar
	a.ResetValue = resetValue
}

func (a *AlwaysBlock) AddStatement(stmt Statement) {
	a.Statements = append(a.Statements, stmt)
}

func (a *AlwaysBlock) AddIfStatement(condition Expression, trueBody []Statement, elseBody []Statement) {
	a.AddStatement(&IfStatement{
		Condition: condition,
		TrueBody:  trueBody,
		ElseBody:  elseBody,
	})
}

func (a *AlwaysBlock) AddCaseStatement(expression Expression, cases []CaseItem, defaultBody []Statement) {
	a.AddStatement(&CaseStatement{
		Expression: expression,
		Cases:      cases,
		Default:    defaultBody,
	})
}

func (a *AlwaysBlock) AddBlockingAssignment(target *Variable, expression Expression, range_ *BitRange) {
	a.AddStatement(&BlockingAssignment{
		Target:     target,
		Expression: expression,
		Range:      range_,
	})
}

func (a *AlwaysBlock) AddNonBlockingAssignment(target *Variable, expression Expression, range_ *BitRange) {
	a.AddStatement(&NonBlockingAssignment{
		Target:     target,
		Expression: expression,
		Range:      range_,
	})
}

func (a *AlwaysBlock) pickClockEdge() bool {
	if a.ForcePosedge {
		return true
	}
	return rand.Float32() < 0.5
}

func NewIfStatement(condition Expression) *IfStatement {
	return &IfStatement{
		Condition: condition,
		TrueBody:  make([]Statement, 0),
		ElseBody:  make([]Statement, 0),
	}
}

func (i *IfStatement) AddTrueStatement(stmt Statement) {
	i.TrueBody = append(i.TrueBody, stmt)
}

func (i *IfStatement) AddElseStatement(stmt Statement) {
	i.ElseBody = append(i.ElseBody, stmt)
}

func NewCaseStatement(expression Expression) *CaseStatement {
	return &CaseStatement{
		Expression: expression,
		Cases:      make([]CaseItem, 0),
		Default:    make([]Statement, 0),
	}
}

func (c *CaseStatement) AddCase(value string, statements []Statement) {
	c.Cases = append(c.Cases, CaseItem{
		Value:      value,
		Statements: statements,
	})
}

func (c *CaseStatement) AddDefault(statements []Statement) {
	c.Default = statements
}

func NewBlockingAssignment(target *Variable, expression Expression, range_ *BitRange) *BlockingAssignment {
	return &BlockingAssignment{
		Target:     target,
		Expression: expression,
		Range:      range_,
	}
}

func NewNonBlockingAssignment(target *Variable, expression Expression, range_ *BitRange) *NonBlockingAssignment {
	return &NonBlockingAssignment{
		Target:     target,
		Expression: expression,
		Range:      range_,
	}
}
