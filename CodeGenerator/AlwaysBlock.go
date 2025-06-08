package CodeGenerator

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
)

// AlwaysBlockType 定义always块的类型
type AlwaysBlockType int

const (
	AlwaysComb  AlwaysBlockType = iota // 组合逻辑
	AlwaysFF                           // 时序逻辑
	AlwaysLatch                        // 锁存器逻辑
)

// AlwaysBlock 表示一个always块
type AlwaysBlock struct {
	Type       AlwaysBlockType // 块类型
	ClockVars  []*Variable     // 时钟变量列表（用于时序逻辑）
	ResetVar   *Variable       // 复位变量（可选）
	ResetValue string          // 复位值（可选）
	Statements []Statement     // 块内的语句
	UsedVars   []*Variable
}

// Statement 表示always块内的语句接口
type Statement interface {
	GenerateString() string
}

// IfStatement 表示if语句
type IfStatement struct {
	Condition Expression
	TrueBody  []Statement
	ElseBody  []Statement
}

// CaseStatement 表示case语句
type CaseStatement struct {
	Expression Expression
	Cases      []CaseItem
	Default    []Statement
}

// CaseItem 表示case语句中的一个分支
type CaseItem struct {
	Value      string
	Statements []Statement
}

// BlockingAssignment 表示阻塞赋值语句
type BlockingAssignment struct {
	Target     *Variable
	Expression Expression
	Range      *BitRange
}

// NonBlockingAssignment 表示非阻塞赋值语句
type NonBlockingAssignment struct {
	Target     *Variable
	Expression Expression
	Range      *BitRange
}

// NewAlwaysBlock 创建一个新的always块
func NewAlwaysBlock(blockType AlwaysBlockType) *AlwaysBlock {
	return &AlwaysBlock{
		Type:       blockType,
		Statements: make([]Statement, 0),
	}
}

// SetClock 设置时钟变量
func (a *AlwaysBlock) SetClock(clockVar *Variable) {
	a.ClockVars = []*Variable{clockVar}
}

// AddClock 添加一个时钟变量
func (a *AlwaysBlock) AddClock(clockVar *Variable) {
	a.ClockVars = append(a.ClockVars, clockVar)
}

// SetClocks 设置多个时钟变量
func (a *AlwaysBlock) SetClocks(clockVars []*Variable) {
	a.ClockVars = clockVars
}

// SetReset 设置复位变量和值
func (a *AlwaysBlock) SetReset(resetVar *Variable, resetValue string) {
	a.ResetVar = resetVar
	a.ResetValue = resetValue
}

// AddStatement 添加语句到always块
func (a *AlwaysBlock) AddStatement(stmt Statement) {
	a.Statements = append(a.Statements, stmt)
}

// AddIfStatement 添加if语句
func (a *AlwaysBlock) AddIfStatement(condition Expression, trueBody []Statement, elseBody []Statement) {
	a.AddStatement(&IfStatement{
		Condition: condition,
		TrueBody:  trueBody,
		ElseBody:  elseBody,
	})
}

// AddCaseStatement 添加case语句
func (a *AlwaysBlock) AddCaseStatement(expression Expression, cases []CaseItem, defaultBody []Statement) {
	a.AddStatement(&CaseStatement{
		Expression: expression,
		Cases:      cases,
		Default:    defaultBody,
	})
}

// AddBlockingAssignment 添加阻塞赋值语句
func (a *AlwaysBlock) AddBlockingAssignment(target *Variable, expression Expression, range_ *BitRange) {
	a.AddStatement(&BlockingAssignment{
		Target:     target,
		Expression: expression,
		Range:      range_,
	})
}

// AddNonBlockingAssignment 添加非阻塞赋值语句
func (a *AlwaysBlock) AddNonBlockingAssignment(target *Variable, expression Expression, range_ *BitRange) {
	a.AddStatement(&NonBlockingAssignment{
		Target:     target,
		Expression: expression,
		Range:      range_,
	})
}

// GenerateString 生成always块的Verilog代码
func (a *AlwaysBlock) GenerateString() string {
	var sb strings.Builder

	// 生成always块头部
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

	// 如果有复位逻辑，添加复位语句
	if a.Type == AlwaysFF && a.ResetVar != nil {
		sb.WriteString(fmt.Sprintf("  if (%s) begin\n", a.ResetVar.Name))
		// 这里需要知道哪些寄存器需要复位
		// 简化处理：假设所有寄存器都需要复位
		for _, target := range a.UsedVars {
			sb.WriteString(fmt.Sprintf("    %s <= %s;\n", target.Name, a.ResetValue))
		}
		sb.WriteString("  end else begin\n")
	}

	// 生成块内语句
	for _, stmt := range a.Statements {
		sb.WriteString("  " + stmt.GenerateString() + "\n")
	}

	// 如果有复位逻辑，关闭else块
	if a.Type == AlwaysFF && a.ResetVar != nil {
		sb.WriteString("  end\n")
	}

	sb.WriteString("end")

	return sb.String()
}

// GenerateString 生成if语句的Verilog代码
func (i *IfStatement) GenerateString() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("if (%s) begin\n", i.Condition.GenerateString()))

	// 生成if分支的语句
	for _, stmt := range i.TrueBody {
		sb.WriteString("  " + stmt.GenerateString() + "\n")
	}

	sb.WriteString("end")

	// 如果有else分支，生成else语句
	if len(i.ElseBody) > 0 {
		sb.WriteString(" else begin\n")

		// 生成else分支的语句
		for _, stmt := range i.ElseBody {
			sb.WriteString("  " + stmt.GenerateString() + "\n")
		}

		sb.WriteString("end")
	}

	return sb.String()
}

// GenerateString 生成case语句的Verilog代码
func (c *CaseStatement) GenerateString() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("case (%s)\n", c.Expression.GenerateString()))

	// 生成各个分支
	for _, caseItem := range c.Cases {
		sb.WriteString(fmt.Sprintf("  %s: begin\n", caseItem.Value))

		// 生成分支内的语句
		for _, stmt := range caseItem.Statements {
			sb.WriteString("    " + stmt.GenerateString() + "\n")
		}

		sb.WriteString("  end\n")
	}

	// 如果有default分支，生成default语句
	if len(c.Default) > 0 {
		sb.WriteString("  default: begin\n")

		// 生成default分支的语句
		for _, stmt := range c.Default {
			sb.WriteString("    " + stmt.GenerateString() + "\n")
		}

		sb.WriteString("  end\n")
	}

	sb.WriteString("endcase")

	return sb.String()
}

// GenerateString 生成阻塞赋值语句的Verilog代码
func (b *BlockingAssignment) GenerateString() string {
	if b.Range != nil {
		return fmt.Sprintf("%s[%d:%d] = %s;",
			b.Target.Name, b.Range.r, b.Range.l, b.Expression.GenerateString())
	}
	return fmt.Sprintf("%s = %s;", b.Target.Name, b.Expression.GenerateString())
}

// GenerateString 生成非阻塞赋值语句的Verilog代码
func (n *NonBlockingAssignment) GenerateString() string {
	if n.Range != nil {
		return fmt.Sprintf("%s[%d:%d] <= %s;",
			n.Target.Name, n.Range.r, n.Range.l, n.Expression.GenerateString())
	}
	return fmt.Sprintf("%s <= %s;", n.Target.Name, n.Expression.GenerateString())
}

// NewIfStatement 创建一个新的if语句
func NewIfStatement(condition Expression) *IfStatement {
	return &IfStatement{
		Condition: condition,
		TrueBody:  make([]Statement, 0),
		ElseBody:  make([]Statement, 0),
	}
}

// AddTrueStatement 添加if分支的语句
func (i *IfStatement) AddTrueStatement(stmt Statement) {
	i.TrueBody = append(i.TrueBody, stmt)
}

// AddElseStatement 添加else分支的语句
func (i *IfStatement) AddElseStatement(stmt Statement) {
	i.ElseBody = append(i.ElseBody, stmt)
}

// NewCaseStatement 创建一个新的case语句
func NewCaseStatement(expression Expression) *CaseStatement {
	return &CaseStatement{
		Expression: expression,
		Cases:      make([]CaseItem, 0),
		Default:    make([]Statement, 0),
	}
}

// AddCase 添加一个case分支
func (c *CaseStatement) AddCase(value string, statements []Statement) {
	c.Cases = append(c.Cases, CaseItem{
		Value:      value,
		Statements: statements,
	})
}

// AddDefault 添加default分支
func (c *CaseStatement) AddDefault(statements []Statement) {
	c.Default = statements
}

// NewBlockingAssignment 创建一个新的阻塞赋值语句
func NewBlockingAssignment(target *Variable, expression Expression, range_ *BitRange) *BlockingAssignment {
	return &BlockingAssignment{
		Target:     target,
		Expression: expression,
		Range:      range_,
	}
}

// NewNonBlockingAssignment 创建一个新的非阻塞赋值语句
func NewNonBlockingAssignment(target *Variable, expression Expression, range_ *BitRange) *NonBlockingAssignment {
	return &NonBlockingAssignment{
		Target:     target,
		Expression: expression,
		Range:      range_,
	}
}

// RandomAlwaysBlockWithTargets 根据给定的左值和右值随机生成always块
func RandomAlwaysBlockWithTargets(gen *ExpressionGenerator, clockVars []*Variable, maxDepth int, maxWidth int) *AlwaysBlock {
	// 只生成时序逻辑always块
	block := NewAlwaysBlock(AlwaysFF)
	// 设置时钟和复位
	definedVars := make([]*Variable, 0)
	resetVar := clockVars[len(clockVars)-1]
	clockVars = clockVars[:len(clockVars)-1]
	if len(clockVars) > 0 {
		block.SetClocks(clockVars)
	}

	block.SetReset(resetVar, strconv.Itoa(rand.Intn(1000)))

	//// 随机生成语句，确保使用所有给定的左值和右值
	//usedTargets := make(map[*Variable]bool)
	//usedExprs := make(map[Expression]bool)
	//
	//// 首先，尝试为每个目标变量分配一个表达式
	//for i, target := range targets {
	//	if !usedTargets[target] && !usedExprs[expressions[i]] {
	//		block.AddStatement(NewNonBlockingAssignment(target, expressions[i], nil))
	//		usedTargets[target] = true
	//		usedExprs[expressions[i]] = true
	//	}
	//}

	// 然后，随机生成额外的语句
	numExtraStatements := maxWidth
	for i := 0; i < numExtraStatements; i++ {
		// 随机选择语句类型
		//stmtType := rand.Intn(3)
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
		case 1: // if语句
			condition := gen.GenerateExpression(maxDepth)
			ifStmt := NewIfStatement(condition)

			// 添加true分支语句
			numTrueStmts := rand.Intn(3) + 1
			for j := 0; j < numTrueStmts; j++ {
				// 如果没有未使用的目标变量或表达式，则随机生成
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

			// 50%的概率添加else分支
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
		case 2: // case语句
			expr := gen.GenerateExpression(maxDepth)
			caseStmt := NewCaseStatement(expr)

			// 添加2-4个case分支
			numCases := rand.Intn(3) + 2
			for j := 0; j < numCases; j++ {
				caseValue := GenerateRandomNumber()
				numCaseStmts := rand.Intn(3) + 1
				caseStatements := make([]Statement, 0, numCaseStmts)

				for k := 0; k < numCaseStmts; k++ {
					// 优先使用未使用的目标变量
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

			// 50%的概率添加default分支
			if true {
				numDefaultStmts := rand.Intn(3) + 1
				defaultStatements := make([]Statement, 0, numDefaultStmts)

				for j := 0; j < numDefaultStmts; j++ {
					// 优先使用未使用的目标变量
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
