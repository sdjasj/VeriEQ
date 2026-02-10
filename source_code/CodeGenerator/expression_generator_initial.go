package CodeGenerator

import (
	"fmt"
	"math/rand"
	"sort"
)

type initialModuleParts struct {
	combAssigns []*AssignExpression
	seqBlock    *AlwaysBlock
	outputStr   string
	isInput     map[*Variable]struct{}
	isOutput    map[*Variable]struct{}
}

func (g *ExpressionGenerator) generateInitialModuleParts() *initialModuleParts {
	g.Clear()

	isInput := make(map[*Variable]struct{})
	isOutput := make(map[*Variable]struct{})
	depth := make(map[*Variable]int)
	defs := make(map[*Variable]Expression)

	for i := 0; i < g.InputNums; i++ {
		varName := fmt.Sprintf("in%d", i)
		variable := g.AddWireVariable(varName)
		resetVarAttrs(variable)
		g.CurrentDefinedVars = append(g.CurrentDefinedVars, variable)
		g.InputVars = append(g.InputVars, variable)
		g.InputPortVars = append(g.InputPortVars, variable)
		isInput[variable] = struct{}{}
		depth[variable] = 0
	}
	for i := 0; i < g.ClockNums; i++ {
		varName := fmt.Sprintf("clock_%d", i)
		variable := g.AddVariableNotArray(varName, VarTypeWire)
		resetVarAttrs(variable)
		g.CurrentDefinedVars = append(g.CurrentDefinedVars, variable)
		g.InputVars = append(g.InputVars, variable)
		g.ClockVars = append(g.ClockVars, variable)
		isInput[variable] = struct{}{}
		depth[variable] = 0
	}

	targetSize := g.InputNums + g.AssignCount
	if targetSize <= len(g.CurrentDefinedVars) {
		targetSize = len(g.CurrentDefinedVars) + 1
	}

	remaining := targetSize - len(g.CurrentDefinedVars)
	seqSlots := minInt(maxInt(1, g.MaxWidth), remaining)
	if seqSlots < 1 {
		seqSlots = 1
	}

	seqStatements := make([]Statement, seqSlots)
	openSlots := make([]int, seqSlots)
	for i := 0; i < seqSlots; i++ {
		openSlots[i] = i
	}

	seqBlock := NewAlwaysBlock(AlwaysFF)
	var clockVar *Variable
	if len(g.ClockVars) > 0 {
		clockVar = g.ClockVars[rand.Intn(len(g.ClockVars))]
	} else if len(g.InputVars) > 0 {
		clockVar = g.InputVars[rand.Intn(len(g.InputVars))]
	}
	if clockVar != nil {
		seqBlock.SetClock(clockVar)
		seqBlock.ResetVar = clockVar
		seqBlock.ResetValue = "0"
		seqBlock.ForcePosedge = true
	}
	seqBlock.Statements = seqStatements

	combAssigns := make([]*AssignExpression, 0)
	seqTargets := make([]*Variable, 0)

	for len(g.CurrentDefinedVars) < targetSize {
		expr := g.GenerateExpressionFromPool(g.MaxDepth, g.CurrentDefinedVars, depth, defs)
		remainingSignals := targetSize - len(g.CurrentDefinedVars)
		remainingSeqSlots := len(openSlots)

		useSeq := false
		if remainingSeqSlots > 0 {
			if remainingSeqSlots >= remainingSignals {
				useSeq = true
			} else {
				useSeq = rand.Float64() < 0.5
			}
		}

		var newVar *Variable
		if useSeq {
			newVar = g.AddRegVariable("")
		} else {
			newVar = g.AddWireVariable("")
		}
		resetVarAttrs(newVar)

		defs[newVar] = expr
		depth[newVar] = 1 + maxDepthInExpr(expr, depth)
		g.CurrentDefinedVars = append(g.CurrentDefinedVars, newVar)

		if useSeq {
			slotIdx := rand.Intn(len(openSlots))
			pos := openSlots[slotIdx]
			openSlots = append(openSlots[:slotIdx], openSlots[slotIdx+1:]...)
			seqStatements[pos] = g.buildSeqStatement(expr, newVar, g.CurrentDefinedVars, depth, defs)
			seqTargets = append(seqTargets, newVar)
		} else {
			combAssigns = append(combAssigns, &AssignExpression{
				Operand1: newVar,
				Right:    expr,
			})
		}
	}

	for _, pos := range openSlots {
		expr := g.GenerateExpressionFromPool(g.MaxDepth, g.CurrentDefinedVars, depth, defs)
		newVar := g.AddRegVariable("")
		resetVarAttrs(newVar)
		defs[newVar] = expr
		depth[newVar] = 1 + maxDepthInExpr(expr, depth)
		g.CurrentDefinedVars = append(g.CurrentDefinedVars, newVar)
		seqStatements[pos] = g.buildSeqStatement(expr, newVar, g.CurrentDefinedVars, depth, defs)
		seqTargets = append(seqTargets, newVar)
	}

	for i, stmt := range seqStatements {
		if stmt == nil {
			expr := g.GenerateExpressionFromPool(g.MaxDepth, g.CurrentDefinedVars, depth, defs)
			newVar := g.AddRegVariable("")
			resetVarAttrs(newVar)
			defs[newVar] = expr
			depth[newVar] = 1 + maxDepthInExpr(expr, depth)
			g.CurrentDefinedVars = append(g.CurrentDefinedVars, newVar)
			seqStatements[i] = g.buildSeqStatement(expr, newVar, g.CurrentDefinedVars, depth, defs)
			seqTargets = append(seqTargets, newVar)
		}
	}

	seqBlock.UsedVars = seqTargets

	g.inferAttrsByDepth(g.CurrentDefinedVars, depth, defs)

	usedOperands := make(map[*Variable]struct{})
	for _, assign := range combAssigns {
		collectVarsInExpr(assign.Right, usedOperands)
	}
	collectVarsFromStatements(seqStatements, usedOperands)

	unused := make([]*Variable, 0)
	for _, v := range g.CurrentDefinedVars {
		if _, used := usedOperands[v]; !used {
			unused = append(unused, v)
		}
	}
	if len(unused) == 0 && len(g.CurrentDefinedVars) > 0 {
		unused = append(unused, g.CurrentDefinedVars[len(g.CurrentDefinedVars)-1])
	}

	outputVar := g.AddWireVariable(fmt.Sprintf("out%d", 0))
	resetVarAttrs(outputVar)
	g.OutputVars = append(g.OutputVars, outputVar)
	isOutput[outputVar] = struct{}{}
	g.CurrentDefinedVars = append(g.CurrentDefinedVars, outputVar)

	outputStr := buildOutputAssign(outputVar, unused)

	return &initialModuleParts{
		combAssigns: combAssigns,
		seqBlock:    seqBlock,
		outputStr:   outputStr,
		isInput:     isInput,
		isOutput:    isOutput,
	}
}

func resetVarAttrs(v *Variable) {
	v.hasRange = false
	v.Range = nil
	v.isSigned = false
}

func (g *ExpressionGenerator) buildSeqStatement(expr Expression, target *Variable, pool []*Variable, depthMap map[*Variable]int, defs map[*Variable]Expression) Statement {
	choice := rand.Float64()
	if choice < 0.5 {
		return NewNonBlockingAssignment(target, expr, nil)
	}
	if choice < 0.8 {
		condition := g.GenerateExpressionFromPool(g.MaxDepth, pool, depthMap, defs)
		ifStmt := NewIfStatement(condition)
		ifStmt.AddTrueStatement(NewNonBlockingAssignment(target, expr, nil))
		if rand.Float64() < 0.5 {
			ifStmt.AddElseStatement(NewNonBlockingAssignment(target, expr, nil))
		}
		return ifStmt
	}

	caseExpr := g.GenerateExpressionFromPool(g.MaxDepth, pool, depthMap, defs)
	caseStmt := NewCaseStatement(caseExpr)
	numCases := rand.Intn(3) + 2
	for i := 0; i < numCases; i++ {
		caseStmt.AddCase(GenerateRandomNumber(), []Statement{
			NewNonBlockingAssignment(target, expr, nil),
		})
	}
	if rand.Float64() < 0.7 {
		caseStmt.AddDefault([]Statement{
			NewNonBlockingAssignment(target, expr, nil),
		})
	}
	return caseStmt
}

func (g *ExpressionGenerator) inferAttrsByDepth(vars []*Variable, depth map[*Variable]int, defs map[*Variable]Expression) {
	sorted := sortByDepth(vars, depth)
	widthSet := make(map[*Variable]bool)
	signedSet := make(map[*Variable]bool)

	for _, v := range sorted {
		if !widthSet[v] {
			width := g.randomWidth()
			setVarWidth(v, width)
			widthSet[v] = true
		}

		if !signedSet[v] {
			signed := balanceSignedness(defs[v], signedSet)
			setVarSignedness(v, signed)
			signedSet[v] = true
		}

		expr, ok := defs[v]
		if !ok {
			continue
		}
		assign := &AssignExpression{
			Operand1: v,
			Right:    expr,
		}
		assign.GetBitWidth()
		assign.GetSignedness()
		assign.PropagateType(0, false)
		applyInferredAttrs(expr, widthSet, signedSet)
	}
}

func (g *ExpressionGenerator) randomWidth() int {
	min := g.MinRangeWidth
	max := g.MaxRangeWidth
	if min < 1 {
		min = 1
	}
	if max < min {
		max = min
	}
	return rand.Intn(max-min+1) + min
}

func setVarWidth(v *Variable, width int) {
	if width < 1 {
		width = 1
	}
	if width == 1 {
		v.hasRange = false
		v.Range = nil
		return
	}
	v.hasRange = true
	v.Range = &BitRange{l: 0, r: width - 1}
}

func setVarSignedness(v *Variable, signed bool) {
	v.isSigned = signed
}

func balanceSignedness(expr Expression, signedSet map[*Variable]bool) bool {
	signed, unsigned := countSignedness(expr, signedSet)
	if signed == 0 && unsigned == 0 {
		return rand.Float64() < 0.5
	}
	if signed == unsigned {
		return rand.Float64() < 0.5
	}
	return signed < unsigned
}

func countSignedness(expr Expression, signedSet map[*Variable]bool) (int, int) {
	var signed, unsigned int
	switch e := expr.(type) {
	case *VariableExpression:
		if signedSet[e.Var] {
			if e.Var.isSigned {
				signed++
			} else {
				unsigned++
			}
		}
	case *BinaryExpression:
		ls, lu := countSignedness(e.Left, signedSet)
		rs, ru := countSignedness(e.Right, signedSet)
		signed += ls + rs
		unsigned += lu + ru
	case *UnaryExpression:
		ls, lu := countSignedness(e.Operand, signedSet)
		signed += ls
		unsigned += lu
	case *TernaryExpression:
		ls, lu := countSignedness(e.Condition, signedSet)
		ts, tu := countSignedness(e.TrueExpr, signedSet)
		fs, fu := countSignedness(e.FalseExpr, signedSet)
		signed += ls + ts + fs
		unsigned += lu + tu + fu
	case *ConcatenationExpression:
		for _, part := range e.Expressions {
			ps, pu := countSignedness(part, signedSet)
			signed += ps
			unsigned += pu
		}
	case *ReplicationExpression:
		es, eu := countSignedness(e.Expression, signedSet)
		signed += es
		unsigned += eu
	}
	return signed, unsigned
}

func applyInferredAttrs(expr Expression, widthSet map[*Variable]bool, signedSet map[*Variable]bool) {
	switch e := expr.(type) {
	case *VariableExpression:
		if !widthSet[e.Var] {
			setVarWidth(e.Var, e.realWidth)
			widthSet[e.Var] = true
		}
		if !signedSet[e.Var] {
			setVarSignedness(e.Var, e.realSigned)
			signedSet[e.Var] = true
		}
	case *BinaryExpression:
		applyInferredAttrs(e.Left, widthSet, signedSet)
		applyInferredAttrs(e.Right, widthSet, signedSet)
	case *UnaryExpression:
		applyInferredAttrs(e.Operand, widthSet, signedSet)
	case *TernaryExpression:
		applyInferredAttrs(e.Condition, widthSet, signedSet)
		applyInferredAttrs(e.TrueExpr, widthSet, signedSet)
		applyInferredAttrs(e.FalseExpr, widthSet, signedSet)
	case *ConcatenationExpression:
		for _, part := range e.Expressions {
			applyInferredAttrs(part, widthSet, signedSet)
		}
	case *ReplicationExpression:
		applyInferredAttrs(e.Expression, widthSet, signedSet)
		applyInferredAttrs(e.Count, widthSet, signedSet)
	}
}

func sortByDepth(vars []*Variable, depth map[*Variable]int) []*Variable {
	sorted := make([]*Variable, len(vars))
	copy(sorted, vars)
	sort.Slice(sorted, func(i, j int) bool {
		return depth[sorted[i]] > depth[sorted[j]]
	})
	return sorted
}

func maxDepthInExpr(expr Expression, depth map[*Variable]int) int {
	switch e := expr.(type) {
	case *VariableExpression:
		return depth[e.Var]
	case *BinaryExpression:
		return maxInt(maxDepthInExpr(e.Left, depth), maxDepthInExpr(e.Right, depth))
	case *UnaryExpression:
		return maxDepthInExpr(e.Operand, depth)
	case *TernaryExpression:
		left := maxDepthInExpr(e.Condition, depth)
		mid := maxDepthInExpr(e.TrueExpr, depth)
		right := maxDepthInExpr(e.FalseExpr, depth)
		return maxInt(left, maxInt(mid, right))
	case *ConcatenationExpression:
		maxDepth := 0
		for _, part := range e.Expressions {
			partDepth := maxDepthInExpr(part, depth)
			if partDepth > maxDepth {
				maxDepth = partDepth
			}
		}
		return maxDepth
	case *ReplicationExpression:
		return maxDepthInExpr(e.Expression, depth)
	default:
		return 0
	}
}

func collectVarsInExpr(expr Expression, used map[*Variable]struct{}) {
	switch e := expr.(type) {
	case *VariableExpression:
		used[e.Var] = struct{}{}
	case *BinaryExpression:
		collectVarsInExpr(e.Left, used)
		collectVarsInExpr(e.Right, used)
	case *UnaryExpression:
		collectVarsInExpr(e.Operand, used)
	case *TernaryExpression:
		collectVarsInExpr(e.Condition, used)
		collectVarsInExpr(e.TrueExpr, used)
		collectVarsInExpr(e.FalseExpr, used)
	case *ConcatenationExpression:
		for _, part := range e.Expressions {
			collectVarsInExpr(part, used)
		}
	case *ReplicationExpression:
		collectVarsInExpr(e.Count, used)
		collectVarsInExpr(e.Expression, used)
	}
}

func collectVarsFromStatements(stmts []Statement, used map[*Variable]struct{}) {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *BlockingAssignment:
			collectVarsInExpr(s.Expression, used)
		case *NonBlockingAssignment:
			collectVarsInExpr(s.Expression, used)
		case *IfStatement:
			collectVarsInExpr(s.Condition, used)
			collectVarsFromStatements(s.TrueBody, used)
			collectVarsFromStatements(s.ElseBody, used)
		case *CaseStatement:
			collectVarsInExpr(s.Expression, used)
			for _, c := range s.Cases {
				collectVarsFromStatements(c.Statements, used)
			}
			collectVarsFromStatements(s.Default, used)
		}
	}
}

func buildOutputAssign(outputVar *Variable, signals []*Variable) string {
	if len(signals) == 0 {
		return fmt.Sprintf("    assign %s = 1'b0;\n", outputVar.Name)
	}
	var out string
	out = fmt.Sprintf("    assign %s = ", outputVar.Name)
	for i, v := range signals {
		if i > 0 {
			out += " + "
		}
		out += v.Name
	}
	out += ";\n"
	return out
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
