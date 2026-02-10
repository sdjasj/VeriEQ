package CodeGenerator

import (
	"math/rand"
	"strconv"
	"strings"
)

const (
	controlFlowTransformProbability = 0.5
	maxDeadCaseWidth                = 30
)

func (g *ExpressionGenerator) ApplyControlFlowTransforms(block *AlwaysBlock) *AlwaysBlock {
	if block == nil {
		return nil
	}
	return &AlwaysBlock{
		Type:         block.Type,
		ClockVars:    append([]*Variable(nil), block.ClockVars...),
		ClockPosedge: append([]bool(nil), block.ClockPosedge...),
		ResetVar:     block.ResetVar,
		ResetValue:   block.ResetValue,
		Statements:   g.transformStatements(block.Statements),
		UsedVars:     append([]*Variable(nil), block.UsedVars...),
		ForcePosedge: block.ForcePosedge,
	}
}

func (g *ExpressionGenerator) buildSeqBlockString(block *AlwaysBlock, transform bool) string {
	if block == nil {
		return ""
	}
	if transform {
		return g.ApplyControlFlowTransforms(block).GenerateString() + "\n"
	}
	return block.GenerateString() + "\n"
}

func (g *ExpressionGenerator) buildAlwaysBlocksString(blocks []*AlwaysBlock, transform bool) string {
	if len(blocks) == 0 {
		return ""
	}
	var sb strings.Builder
	for _, block := range blocks {
		if block == nil {
			continue
		}
		rendered := block
		if transform {
			rendered = g.ApplyControlFlowTransforms(block)
		}
		sb.WriteString(rendered.GenerateString())
		sb.WriteString("\n")
	}
	return sb.String()
}

func (g *ExpressionGenerator) transformStatements(stmts []Statement) []Statement {
	if len(stmts) == 0 {
		return nil
	}
	out := make([]Statement, 0, len(stmts))
	for _, stmt := range stmts {
		out = append(out, g.transformStatement(stmt))
	}
	return out
}

func (g *ExpressionGenerator) transformStatement(stmt Statement) Statement {
	switch s := stmt.(type) {
	case *IfStatement:
		trueBody := g.transformStatements(s.TrueBody)
		elseBody := g.transformStatements(s.ElseBody)
		baseIf := &IfStatement{
			Condition: cloneExpression(s.Condition),
			TrueBody:  trueBody,
			ElseBody:  elseBody,
		}
		if !g.EnableControlFlowEquiv || rand.Float64() >= controlFlowTransformProbability {
			return baseIf
		}
		if len(baseIf.ElseBody) == 0 {
			return baseIf
		}
		var candidates []Statement
		if candidate, ok := ifToCaseB1(baseIf); ok {
			candidates = append(candidates, candidate)
		}
		if candidate, ok := ifToCaseB2(baseIf); ok {
			candidates = append(candidates, candidate)
		}
		if len(candidates) == 0 {
			return baseIf
		}
		return candidates[rand.Intn(len(candidates))]
	case *CaseStatement:
		cases := make([]CaseItem, 0, len(s.Cases))
		for _, c := range s.Cases {
			cases = append(cases, CaseItem{
				Value:      c.Value,
				Statements: g.transformStatements(c.Statements),
			})
		}
		defaultBody := g.transformStatements(s.Default)
		baseCase := &CaseStatement{
			Expression: cloneExpression(s.Expression),
			Cases:      cases,
			Default:    defaultBody,
		}
		if !g.EnableControlFlowEquiv || rand.Float64() >= controlFlowTransformProbability {
			return baseCase
		}
		if rand.Intn(2) == 0 {
			if candidate, ok := g.caseToIfChainB4(baseCase); ok {
				return candidate
			}
			if candidate, ok := g.caseAddDeadBranchB3(baseCase); ok {
				return candidate
			}
		} else {
			if candidate, ok := g.caseAddDeadBranchB3(baseCase); ok {
				return candidate
			}
			if candidate, ok := g.caseToIfChainB4(baseCase); ok {
				return candidate
			}
		}
		return baseCase
	case *BlockingAssignment:
		return &BlockingAssignment{
			Target:     s.Target,
			Expression: cloneExpression(s.Expression),
			Range:      cloneRange(s.Range),
		}
	case *NonBlockingAssignment:
		return &NonBlockingAssignment{
			Target:     s.Target,
			Expression: cloneExpression(s.Expression),
			Range:      cloneRange(s.Range),
		}
	default:
		return stmt
	}
}

func ifToCaseB1(stmt *IfStatement) (Statement, bool) {
	if stmt == nil || len(stmt.ElseBody) == 0 {
		return nil, false
	}
	cond, ok := stmt.Condition.(*BinaryExpression)
	if !ok || cond.Operator != "==" {
		return nil, false
	}
	return &CaseStatement{
		Expression: cloneExpression(cond.Left),
		Cases: []CaseItem{
			{
				Value:      cond.Right.GenerateString(),
				Statements: cloneStatements(stmt.TrueBody),
			},
		},
		Default: cloneStatements(stmt.ElseBody),
	}, true
}

func ifToCaseB2(stmt *IfStatement) (Statement, bool) {
	if stmt == nil || len(stmt.ElseBody) == 0 {
		return nil, false
	}
	return &CaseStatement{
		Expression: cloneExpression(stmt.Condition),
		Cases: []CaseItem{
			{
				Value:      "1'b0",
				Statements: cloneStatements(stmt.ElseBody),
			},
		},
		Default: cloneStatements(stmt.TrueBody),
	}, true
}

func (g *ExpressionGenerator) caseAddDeadBranchB3(stmt *CaseStatement) (Statement, bool) {
	if stmt == nil || len(stmt.Cases) == 0 {
		return nil, false
	}
	width := effectiveWidth(stmt.Expression)
	if width <= 0 || width > maxDeadCaseWidth {
		return nil, false
	}
	if effectiveSignedness(stmt.Expression) {
		return nil, false
	}
	for _, c := range stmt.Cases {
		info, ok := parseVerilogLiteral(c.Value)
		if !ok || info.signed {
			return nil, false
		}
	}
	deadValue := uint64(1) << uint(width)
	deadLiteral := strconv.FormatUint(deadValue, 10)
	deadBody := cloneStatements(stmt.Default)
	if len(deadBody) == 0 && len(stmt.Cases) > 0 {
		deadBody = cloneStatements(stmt.Cases[0].Statements)
	}
	newCases := make([]CaseItem, 0, len(stmt.Cases)+1)
	newCases = append(newCases, stmt.Cases...)
	newCases = append(newCases, CaseItem{
		Value:      deadLiteral,
		Statements: deadBody,
	})
	return &CaseStatement{
		Expression: cloneExpression(stmt.Expression),
		Cases:      newCases,
		Default:    cloneStatements(stmt.Default),
	}, true
}

func (g *ExpressionGenerator) caseToIfChainB4(stmt *CaseStatement) (Statement, bool) {
	if stmt == nil || len(stmt.Cases) == 0 {
		return nil, false
	}
	width := effectiveWidth(stmt.Expression)
	signed := effectiveSignedness(stmt.Expression)
	if width <= 0 {
		return nil, false
	}
	var root *IfStatement
	var current *IfStatement
	for _, c := range stmt.Cases {
		info, ok := parseVerilogLiteral(c.Value)
		if !ok || info.width != width || info.signed != signed {
			return nil, false
		}
		cond := &BinaryExpression{
			Left:     cloneExpression(stmt.Expression),
			Right:    newConst(info.value, info.width, info.signed),
			Operator: "==",
		}
		next := &IfStatement{
			Condition: cond,
			TrueBody:  cloneStatements(c.Statements),
		}
		if root == nil {
			root = next
		} else {
			current.ElseBody = []Statement{next}
		}
		current = next
	}
	if root == nil {
		return nil, false
	}
	if len(stmt.Default) > 0 {
		current.ElseBody = cloneStatements(stmt.Default)
	}
	return root, true
}

type literalInfo struct {
	width  int
	signed bool
	value  uint64
}

func parseVerilogLiteral(lit string) (literalInfo, bool) {
	s := strings.TrimSpace(lit)
	if s == "" {
		return literalInfo{}, false
	}
	if strings.ContainsAny(s, "xXzZ?") {
		return literalInfo{}, false
	}
	if idx := strings.IndexByte(s, '\''); idx != -1 {
		widthStr := strings.TrimSpace(s[:idx])
		if widthStr == "" {
			return literalInfo{}, false
		}
		width, err := strconv.Atoi(widthStr)
		if err != nil || width <= 0 {
			return literalInfo{}, false
		}
		rest := s[idx+1:]
		if rest == "" {
			return literalInfo{}, false
		}
		signed := false
		if rest[0] == 's' || rest[0] == 'S' {
			signed = true
			rest = rest[1:]
		}
		if rest == "" {
			return literalInfo{}, false
		}
		base := rest[0]
		digits := strings.ReplaceAll(rest[1:], "_", "")
		if digits == "" {
			return literalInfo{}, false
		}
		var radix int
		switch base {
		case 'b', 'B':
			radix = 2
		case 'o', 'O':
			radix = 8
		case 'd', 'D':
			radix = 10
		case 'h', 'H':
			radix = 16
		default:
			return literalInfo{}, false
		}
		value, err := strconv.ParseUint(digits, radix, 64)
		if err != nil {
			return literalInfo{}, false
		}
		return literalInfo{width: width, signed: signed, value: value}, true
	}
	s = strings.ReplaceAll(s, "_", "")
	if strings.HasPrefix(s, "-") {
		return literalInfo{}, false
	}
	value, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return literalInfo{}, false
	}
	return literalInfo{width: 32, signed: true, value: value}, true
}

func cloneStatements(stmts []Statement) []Statement {
	if len(stmts) == 0 {
		return nil
	}
	out := make([]Statement, 0, len(stmts))
	for _, stmt := range stmts {
		out = append(out, cloneStatement(stmt))
	}
	return out
}

func cloneStatement(stmt Statement) Statement {
	switch s := stmt.(type) {
	case *IfStatement:
		return &IfStatement{
			Condition: cloneExpression(s.Condition),
			TrueBody:  cloneStatements(s.TrueBody),
			ElseBody:  cloneStatements(s.ElseBody),
		}
	case *CaseStatement:
		cases := make([]CaseItem, 0, len(s.Cases))
		for _, c := range s.Cases {
			cases = append(cases, CaseItem{
				Value:      c.Value,
				Statements: cloneStatements(c.Statements),
			})
		}
		return &CaseStatement{
			Expression: cloneExpression(s.Expression),
			Cases:      cases,
			Default:    cloneStatements(s.Default),
		}
	case *BlockingAssignment:
		return &BlockingAssignment{
			Target:     s.Target,
			Expression: cloneExpression(s.Expression),
			Range:      cloneRange(s.Range),
		}
	case *NonBlockingAssignment:
		return &NonBlockingAssignment{
			Target:     s.Target,
			Expression: cloneExpression(s.Expression),
			Range:      cloneRange(s.Range),
		}
	default:
		return stmt
	}
}

func cloneRange(r *BitRange) *BitRange {
	if r == nil {
		return nil
	}
	return &BitRange{l: r.l, r: r.r}
}
