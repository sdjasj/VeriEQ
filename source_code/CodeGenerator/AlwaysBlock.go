package CodeGenerator

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
)

type AlwaysBlockType int

const (
	AlwaysComb AlwaysBlockType = iota
	AlwaysFF
	AlwaysLatch
)

type AlwaysBlock struct {
	Type       AlwaysBlockType
	ClockVars  []*Variable
	ResetVar   *Variable
	ResetValue string
	Statements []Statement
	UsedVars   []*Variable
}

type Statement interface {
	GenerateString() string
}

type IfStatement struct {
	Condition Expression
	TrueBody  []Statement
	ElseBody  []Statement
}

type CaseStatement struct {
	Expression Expression
	Cases      []CaseItem
	Default    []Statement
}

type CaseItem struct {
	Value      string
	Statements []Statement
}

type BlockingAssignment struct {
	Target     *Variable
	Expression Expression
	Range      *BitRange
}

type NonBlockingAssignment struct {
	Target     *Variable
	Expression Expression
	Range      *BitRange
}

func NewAlwaysBlock(blockType AlwaysBlockType) *AlwaysBlock {
	return &AlwaysBlock{
		Type:       blockType,
		Statements: make([]Statement, 0),
	}
}

func (a *AlwaysBlock) SetClock(clockVar *Variable) {
	a.ClockVars = []*Variable{clockVar}
}

func (a *AlwaysBlock) AddClock(clockVar *Variable) {
	a.ClockVars = append(a.ClockVars, clockVar)
}

func (a *AlwaysBlock) SetClocks(clockVars []*Variable) {
	a.ClockVars = clockVars
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

func (a *AlwaysBlock) GenerateString() string {
	var sb strings.Builder

	switch a.Type {
	case AlwaysComb:
		sb.WriteString("always @(*) begin\n")
	case AlwaysFF:
		if len(a.ClockVars) == 0 {
			sb.WriteString("always @(*) begin\n")
		} else {
			sb.WriteString("always @(")
			for i, clock := range a.ClockVars {
				if i > 0 {
					sb.WriteString(" or ")
				}
				if rand.Float32() < 0.5 {
					sb.WriteString("posedge ")
				} else {
					sb.WriteString("negedge ")
				}

				sb.WriteString(clock.Name)
			}
			if a.ResetVar != nil {
				sb.WriteString(" or posedge ")
				sb.WriteString(a.ResetVar.Name)
			}
			sb.WriteString(") begin\n")
		}
	case AlwaysLatch:
		sb.WriteString("always @(*) begin\n")
	}

	if a.Type == AlwaysFF && a.ResetVar != nil {
		sb.WriteString(fmt.Sprintf("  if (%s) begin\n", a.ResetVar.Name))

		for _, target := range a.UsedVars {
			sb.WriteString(fmt.Sprintf("    %s <= %s;\n", target.Name, a.ResetValue))
		}
		sb.WriteString("  end else begin\n")
	}

	for _, stmt := range a.Statements {
		sb.WriteString("  " + stmt.GenerateString() + "\n")
	}

	if a.Type == AlwaysFF && a.ResetVar != nil {
		sb.WriteString("  end\n")
	}

	sb.WriteString("end")

	return sb.String()
}

func (i *IfStatement) GenerateString() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("if (%s) begin\n", i.Condition.GenerateString()))

	for _, stmt := range i.TrueBody {
		sb.WriteString("  " + stmt.GenerateString() + "\n")
	}

	sb.WriteString("end")

	if len(i.ElseBody) > 0 {
		sb.WriteString(" else begin\n")

		for _, stmt := range i.ElseBody {
			sb.WriteString("  " + stmt.GenerateString() + "\n")
		}

		sb.WriteString("end")
	}

	return sb.String()
}

func (c *CaseStatement) GenerateString() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("case (%s)\n", c.Expression.GenerateString()))

	for _, caseItem := range c.Cases {
		sb.WriteString(fmt.Sprintf("  %s: begin\n", caseItem.Value))

		for _, stmt := range caseItem.Statements {
			sb.WriteString("    " + stmt.GenerateString() + "\n")
		}

		sb.WriteString("  end\n")
	}

	if len(c.Default) > 0 {
		sb.WriteString("  default: begin\n")

		for _, stmt := range c.Default {
			sb.WriteString("    " + stmt.GenerateString() + "\n")
		}

		sb.WriteString("  end\n")
	}

	sb.WriteString("endcase")

	return sb.String()
}

func (b *BlockingAssignment) GenerateString() string {
	if b.Range != nil {
		return fmt.Sprintf("%s[%d:%d] = %s;",
			b.Target.Name, b.Range.r, b.Range.l, b.Expression.GenerateString())
	}
	return fmt.Sprintf("%s = %s;", b.Target.Name, b.Expression.GenerateString())
}

func (n *NonBlockingAssignment) GenerateString() string {
	if n.Range != nil {
		return fmt.Sprintf("%s[%d:%d] <= %s;",
			n.Target.Name, n.Range.r, n.Range.l, n.Expression.GenerateString())
	}
	return fmt.Sprintf("%s <= %s;", n.Target.Name, n.Expression.GenerateString())
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

func RandomAlwaysBlockWithTargets(gen *ExpressionGenerator, clockVars []*Variable, maxDepth int, maxWidth int) *AlwaysBlock {

	block := NewAlwaysBlock(AlwaysFF)

	definedVars := make([]*Variable, 0)
	resetVar := clockVars[len(clockVars)-1]
	clockVars = clockVars[:len(clockVars)-1]
	if len(clockVars) > 0 {
		block.SetClocks(clockVars)
	}

	block.SetReset(resetVar, strconv.Itoa(rand.Intn(1000)))

	numExtraStatements := maxWidth
	for i := 0; i < numExtraStatements; i++ {
		stmtType := rand.Intn(2)
		switch stmtType {
		case 0: // 非阻塞赋值
			target := gen.AddRegVariable("")
			definedVars = append(definedVars, target)
			expr := gen.GenerateExpression(maxDepth)
			var ran *BitRange
			if target.hasRange {
				ran = GetRandomRangeFromVar(target)
			}
			block.AddStatement(NewNonBlockingAssignment(target, expr, ran))
		case 1:
			condition := gen.GenerateExpression(maxDepth)
			ifStmt := NewIfStatement(condition)

			numTrueStmts := rand.Intn(3) + 1
			for j := 0; j < numTrueStmts; j++ {
				var target *Variable
				target = gen.AddRegVariable("")
				definedVars = append(definedVars, target)
				expr := gen.GenerateExpression(maxDepth)
				var ran *BitRange
				if target.hasRange {
					ran = GetRandomRangeFromVar(target)
				}
				ifStmt.AddTrueStatement(NewNonBlockingAssignment(target, expr, ran))
			}

			if rand.Float32() < 0.5 {
				numElseStmts := rand.Intn(3) + 1
				for j := 0; j < numElseStmts; j++ {
					var target *Variable
					target = gen.AddRegVariable("")
					definedVars = append(definedVars, target)
					expr := gen.GenerateExpression(maxDepth)
					var ran *BitRange
					if target.hasRange {
						ran = GetRandomRangeFromVar(target)
					}
					ifStmt.AddElseStatement(NewNonBlockingAssignment(target, expr, ran))
				}
			}

			block.AddStatement(ifStmt)
		case 2:
			expr := gen.GenerateExpression(maxDepth)
			caseStmt := NewCaseStatement(expr)

			numCases := rand.Intn(3) + 2
			for j := 0; j < numCases; j++ {
				caseValue := GenerateRandomNumber()
				numCaseStmts := rand.Intn(3) + 1
				caseStatements := make([]Statement, 0, numCaseStmts)

				for k := 0; k < numCaseStmts; k++ {

					var target *Variable
					target = gen.AddRegVariable("")
					definedVars = append(definedVars, target)
					expr := gen.GenerateExpression(maxDepth)
					var ran *BitRange
					if target.hasRange {
						ran = GetRandomRangeFromVar(target)
					}
					caseStatements = append(caseStatements, NewNonBlockingAssignment(target, expr, ran))
				}

				caseStmt.AddCase(caseValue, caseStatements)
			}

			if true {
				numDefaultStmts := rand.Intn(3) + 1
				defaultStatements := make([]Statement, 0, numDefaultStmts)

				for j := 0; j < numDefaultStmts; j++ {
					var target *Variable
					target = gen.AddRegVariable("")
					definedVars = append(definedVars, target)
					expr := gen.GenerateExpression(maxDepth)
					var ran *BitRange
					if target.hasRange {
						ran = GetRandomRangeFromVar(target)
					}
					defaultStatements = append(defaultStatements, NewNonBlockingAssignment(target, expr, ran))
				}

				caseStmt.AddDefault(defaultStatements)
			}

			block.AddStatement(caseStmt)
		}
	}
	block.UsedVars = definedVars
	gen.CurrentDefinedVars = append(gen.CurrentDefinedVars, definedVars...)
	return block
}
