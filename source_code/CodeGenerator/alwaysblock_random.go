package CodeGenerator

import (
	"math/rand"
	"strconv"
)

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
		stmtType := rand.Intn(3)
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
