package CodeGenerator

import (
	"fmt"
	"math/rand"
)

func (g *ExpressionGenerator) AddVariable(name string, varType VerilogVarType) *Variable {
	if _, exists := g.Variables[name]; exists {
		return g.Variables[name]
	}

	v := &Variable{
		Name: name,
		Type: varType,
	}

	if rand.Float64() < g.ProbabilityOfRange {
		v.hasRange = true
		r := rand.Intn(g.MaxRangeWidth-g.MinRangeWidth) + g.MinRangeWidth
		l := rand.Intn(r + 1)
		v.Range = &BitRange{
			r: r,
			l: l,
		}
	}

	if rand.Float64() < g.ProbabilityOfSigned {
		v.isSigned = true
	}

	g.Variables[name] = v
	return v
}

func (g *ExpressionGenerator) AddVariableNotArray(name string, varType VerilogVarType) *Variable {
	if _, exists := g.Variables[name]; exists {
		return g.Variables[name]
	}

	v := &Variable{
		Name: name,
		Type: varType,
	}

	g.Variables[name] = v
	return v
}

func (g *ExpressionGenerator) AddWireVariable(name string) *Variable {
	if name == "" {
		name = fmt.Sprintf("wire_%d", g.WireIndex)
		g.WireIndex++
	}
	return g.AddVariable(name, VarTypeWire)
}

func (g *ExpressionGenerator) AddRegVariable(name string) *Variable {
	if name == "" {
		name = fmt.Sprintf("reg_%d", g.RegIndex)
		g.RegIndex++
	}
	return g.AddVariable(name, VarTypeReg)
}
