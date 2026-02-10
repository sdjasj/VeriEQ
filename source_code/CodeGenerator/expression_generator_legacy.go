package CodeGenerator

import (
	"fmt"
	"math/rand"
	"strings"
)

type legacyModuleParts struct {
	assignExpressions []*AssignExpression
	alwaysBlocks      []*AlwaysBlock
	outputStr         string
	isInput           map[*Variable]struct{}
	isOutput          map[*Variable]struct{}
}

func (g *ExpressionGenerator) generateLegacyModuleParts() *legacyModuleParts {
	g.Clear()

	isInput := make(map[*Variable]struct{})
	isOutput := make(map[*Variable]struct{})

	for i := 0; i < g.InputNums; i++ {
		varName := fmt.Sprintf("in%d", i)
		variable := g.AddWireVariable(varName)
		g.CurrentDefinedVars = append(g.CurrentDefinedVars, variable)
		g.InputVars = append(g.InputVars, variable)
		g.InputPortVars = append(g.InputPortVars, variable)
		isInput[variable] = struct{}{}
	}

	for i := 0; i < g.ClockNums; i++ {
		varName := fmt.Sprintf("clock_%d", i)
		variable := g.AddVariableNotArray(varName, VarTypeWire)
		g.InputVars = append(g.InputVars, variable)
		g.ClockVars = append(g.ClockVars, variable)
		isInput[variable] = struct{}{}
	}

	for i := 0; i < g.OutputNums; i++ {
		varName := fmt.Sprintf("out%d", i)
		variable := g.AddWireVariable(varName)
		g.OutputVars = append(g.OutputVars, variable)
		isOutput[variable] = struct{}{}
	}

	assignExpressions := make([]*AssignExpression, 0)
	for i := 0; i < g.AssignCount; i++ {
		assignExpressions = append(assignExpressions, g.GenerateLoopFreeAssignment(nil))
	}
	for _, assign := range assignExpressions {
		g.CurrentDefinedVars = append(g.CurrentDefinedVars, assign.Operand1)
	}

	alwaysBlocks := make([]*AlwaysBlock, 0)
	randomPick := func(slice []*Variable, count int) []*Variable {
		indices := rand.Perm(len(slice))[:count]
		result := make([]*Variable, count)
		for i, idx := range indices {
			result[i] = slice[idx]
		}
		return result
	}
	for i := 0; i < g.AlwaysCount; i++ {
		alwaysClocks := randomPick(g.ClockVars, 2)
		alwaysBlocks = append(alwaysBlocks, RandomAlwaysBlockWithTargets(g, alwaysClocks, g.MaxDepth, g.MaxWidth))
	}

	outputVar := g.OutputVars[0]
	outputStr := fmt.Sprintf("    assign %s = ", outputVar.Name)
	flag := false
	for _, v := range g.CurrentDefinedVars {
		if _, ok := isInput[v]; ok {
			continue
		}
		if flag {
			outputStr += fmt.Sprintf("+ %s ", v.Name)
		} else {
			outputStr += fmt.Sprintf("%s ", v.Name)
			flag = true
		}
	}
	outputStr += ";\n"
	g.CurrentDefinedVars = append(g.CurrentDefinedVars, outputVar)

	return &legacyModuleParts{
		assignExpressions: assignExpressions,
		alwaysBlocks:      alwaysBlocks,
		outputStr:         outputStr,
		isInput:           isInput,
		isOutput:          isOutput,
	}
}

func (g *ExpressionGenerator) generateLegacyLoopFreeModule() string {
	parts := g.generateLegacyModuleParts()

	moduleStr := fmt.Sprintf("`timescale 1ns/1ps\nmodule %s (", g.Name)

	for i, input := range g.InputVars {
		if i == len(g.InputVars)-1 {
			moduleStr += fmt.Sprintf("%s, ", input.GetName())
		} else {
			moduleStr += fmt.Sprintf("%s, ", input.GetName())
		}
	}

	for i, output := range g.OutputVars {
		if i == len(g.OutputVars)-1 {
			moduleStr += fmt.Sprintf("%s ", output.GetName())
		} else {
			moduleStr += fmt.Sprintf("%s, ", output.GetName())
		}
	}

	moduleStr += ");\n\n"

	for _, v := range g.CurrentDefinedVars {
		signedStr := ""
		if v.isSigned {
			signedStr = "signed"
		}
		if v.Type == VarTypeWire {
			var s string
			if v.hasRange {
				s = fmt.Sprintf("wire %s [%d:%d] %s;\n", signedStr, v.Range.r, v.Range.l, v.GetName())
			} else {
				s = fmt.Sprintf("wire %s %s;\n", signedStr, v.GetName())
			}
			if _, ok := parts.isInput[v]; ok {
				s = "input " + s
			} else if _, ok := parts.isOutput[v]; ok {
				s = "output " + s
			}
			moduleStr += s
		} else if v.Type == VarTypeReg {
			var s string
			if v.hasRange {
				s = fmt.Sprintf("reg %s [%d:%d] %s;\n", signedStr, v.Range.r, v.Range.l, v.GetName())
			} else {
				s = fmt.Sprintf("reg %s %s;\n", signedStr, v.GetName())
			}
			if _, ok := parts.isInput[v]; ok {
				s = "input " + s
			} else if _, ok := parts.isOutput[v]; ok {
				s = "output " + s
			}
			moduleStr += s
		}
	}

	for _, v := range g.ClockVars {
		moduleStr += fmt.Sprintf("input %s;\n", v.GetName())
		moduleStr += fmt.Sprintf("wire %s;\n", v.GetName())
	}

	moduleStr += "\n"

	for _, assign := range parts.assignExpressions {
		moduleStr += assign.GenerateString() + "\n"
	}

	moduleStr += parts.outputStr

	for _, always := range parts.alwaysBlocks {
		moduleStr += always.GenerateString() + "\n"
	}

	moduleStr += "endmodule\n"

	return moduleStr
}

func (g *ExpressionGenerator) generateLegacyLoopFreeEquivalentModules(equalNumber int) string {
	parts := g.generateLegacyModuleParts()

	for i := range parts.assignExpressions {
		parts.assignExpressions[i].GetBitWidth()
		parts.assignExpressions[i].GetSignedness()
		parts.assignExpressions[i].PropagateType(0, false)
	}
	baseAssigns := cloneAssignExpressions(parts.assignExpressions)

	baseAlwaysStr := g.buildAlwaysBlocksString(parts.alwaysBlocks, false)

	modules := make([]string, 0, equalNumber)

	for eqIdx := 0; eqIdx < equalNumber; eqIdx++ {
		currentAssigns := cloneAssignExpressions(baseAssigns)
		if eqIdx > 0 {
			currentAssigns = g.ApplyEquivalenceTransforms(currentAssigns)
		}

		moduleStr := ""
		if eqIdx == 0 {
			moduleStr += "`timescale 1ns/1ps\n"
		}
		moduleStr += fmt.Sprintf("module %s_eq%d (", g.Name, eqIdx)

		for i, input := range g.InputVars {
			moduleStr += input.GetName()
			if i != len(g.InputVars)-1 || len(g.OutputVars) > 0 {
				moduleStr += ", "
			}
		}
		for i, output := range g.OutputVars {
			moduleStr += output.GetName()
			if i != len(g.OutputVars)-1 {
				moduleStr += ", "
			}
		}
		moduleStr += ");\n\n"

		for _, v := range g.CurrentDefinedVars {
			signedStr := ""
			if v.isSigned {
				signedStr = "signed"
			}
			var decl string
			if v.Type == VarTypeWire {
				if v.hasRange {
					decl = fmt.Sprintf("wire %s [%d:%d] %s;\n", signedStr, v.Range.r, v.Range.l, v.GetName())
				} else {
					decl = fmt.Sprintf("wire %s %s;\n", signedStr, v.GetName())
				}
			} else if v.Type == VarTypeReg {
				if v.hasRange {
					decl = fmt.Sprintf("reg %s [%d:%d] %s;\n", signedStr, v.Range.r, v.Range.l, v.GetName())
				} else {
					decl = fmt.Sprintf("reg %s %s;\n", signedStr, v.GetName())
				}
			}
			if _, ok := parts.isInput[v]; ok {
				decl = "input " + decl
			} else if _, ok := parts.isOutput[v]; ok {
				decl = "output " + decl
			}
			moduleStr += decl
		}

		for _, clk := range g.ClockVars {
			moduleStr += fmt.Sprintf("input %s;\n", clk.GetName())
			moduleStr += fmt.Sprintf("wire %s;\n", clk.GetName())
		}

		moduleStr += "\n"

		for _, assign := range currentAssigns {
			moduleStr += assign.GenerateString() + "\n"
		}
		moduleStr += parts.outputStr

		alwaysStr := baseAlwaysStr
		if eqIdx > 0 && g.EnableControlFlowEquiv {
			alwaysStr = g.buildAlwaysBlocksString(parts.alwaysBlocks, true)
		}
		moduleStr += alwaysStr

		moduleStr += "endmodule\n"
		modules = append(modules, moduleStr)
	}

	var combinedModule strings.Builder
	for _, module := range modules {
		combinedModule.WriteString(module)
		combinedModule.WriteString("\n\n")
	}

	return combinedModule.String()
}

func (g *ExpressionGenerator) generateLegacyEquivalentModulesWithOneTop(equalNumber int) string {
	parts := g.generateLegacyModuleParts()

	for i := range parts.assignExpressions {
		parts.assignExpressions[i].GetBitWidth()
		parts.assignExpressions[i].GetSignedness()
		parts.assignExpressions[i].PropagateType(0, false)
	}
	baseAssigns := cloneAssignExpressions(parts.assignExpressions)

	baseAlwaysStr := g.buildAlwaysBlocksString(parts.alwaysBlocks, false)

	outputVar := g.OutputVars[0]

	modules := make([]string, 0, equalNumber)
	moduleNames := make([]string, 0, equalNumber)

	for eqIdx := 0; eqIdx < equalNumber; eqIdx++ {
		currentAssigns := cloneAssignExpressions(baseAssigns)
		if eqIdx > 0 {
			currentAssigns = g.ApplyEquivalenceTransforms(currentAssigns)
		}
		moduleName := fmt.Sprintf("%s_eq%d", g.Name, eqIdx)
		moduleNames = append(moduleNames, moduleName)
		moduleStr := ""
		moduleStr += fmt.Sprintf("module %s_eq%d (", g.Name, eqIdx)

		for i, input := range g.InputVars {
			moduleStr += input.GetName()
			if i != len(g.InputVars)-1 || len(g.OutputVars) > 0 {
				moduleStr += ", "
			}
		}
		for i, output := range g.OutputVars {
			moduleStr += output.GetName()
			if i != len(g.OutputVars)-1 {
				moduleStr += ", "
			}
		}
		moduleStr += ");\n\n"

		for _, v := range g.CurrentDefinedVars {
			signedStr := ""
			if v.isSigned {
				signedStr = "signed"
			}
			var decl string
			if v.Type == VarTypeWire {
				if v.hasRange {
					decl = fmt.Sprintf("wire %s [%d:%d] %s;\n", signedStr, v.Range.r, v.Range.l, v.GetName())
				} else {
					decl = fmt.Sprintf("wire %s %s;\n", signedStr, v.GetName())
				}
			} else if v.Type == VarTypeReg {
				if v.hasRange {
					decl = fmt.Sprintf("reg %s [%d:%d] %s;\n", signedStr, v.Range.r, v.Range.l, v.GetName())
				} else {
					decl = fmt.Sprintf("reg %s %s;\n", signedStr, v.GetName())
				}
			}
			if _, ok := parts.isInput[v]; ok {
				decl = "input " + decl
			} else if _, ok := parts.isOutput[v]; ok {
				decl = "output " + decl
			}
			moduleStr += decl
		}

		for _, clk := range g.ClockVars {
			moduleStr += fmt.Sprintf("input %s;\n", clk.GetName())
			moduleStr += fmt.Sprintf("wire %s;\n", clk.GetName())
		}

		moduleStr += "\n"

		for _, assign := range currentAssigns {
			moduleStr += assign.GenerateString() + "\n"
		}
		moduleStr += parts.outputStr

		alwaysStr := baseAlwaysStr
		if eqIdx > 0 && g.EnableControlFlowEquiv {
			alwaysStr = g.buildAlwaysBlocksString(parts.alwaysBlocks, true)
		}
		moduleStr += alwaysStr

		moduleStr += "endmodule\n"
		modules = append(modules, moduleStr)
	}

	var topBuilder strings.Builder
	topBuilder.WriteString("`timescale 1ns/1ps\nmodule top(\n")
	for i, input := range g.InputVars {
		topBuilder.WriteString(input.GetName())
		if i != len(g.InputVars)-1 || len(g.OutputVars) > 0 {
			topBuilder.WriteString(", ")
		}
	}
	for i, output := range g.OutputVars {
		topBuilder.WriteString(output.GetName())
		if i != len(g.OutputVars)-1 {
			topBuilder.WriteString(", ")
		}
	}
	topBuilder.WriteString(");\n\n")
	for _, v := range g.CurrentDefinedVars {
		signedStr := ""
		if v.isSigned {
			signedStr = "signed"
		}
		var decl string
		if v.Type == VarTypeWire {
			if v.hasRange {
				decl = fmt.Sprintf("wire %s [%d:%d] %s;\n", signedStr, v.Range.r, v.Range.l, v.GetName())
			} else {
				decl = fmt.Sprintf("wire %s %s;\n", signedStr, v.GetName())
			}
		} else if v.Type == VarTypeReg {
			if v.hasRange {
				decl = fmt.Sprintf("reg %s [%d:%d] %s;\n", signedStr, v.Range.r, v.Range.l, v.GetName())
			} else {
				decl = fmt.Sprintf("reg %s %s;\n", signedStr, v.GetName())
			}
		}
		if _, ok := parts.isInput[v]; ok {
			decl = "input " + decl
		} else {
			continue
		}
		topBuilder.WriteString(decl)
	}

	for _, clk := range g.ClockVars {
		topBuilder.WriteString(fmt.Sprintf("input %s;\n", clk.GetName()))
		topBuilder.WriteString(fmt.Sprintf("wire %s;\n", clk.GetName()))
	}
	for i := 0; i < equalNumber; i++ {
		v := outputVar
		var decl string
		signedStr := ""
		if v.isSigned {
			signedStr = "signed"
		}
		if v.hasRange {
			decl = fmt.Sprintf("wire %s [%d:%d] %s_%d;\n", signedStr, v.Range.r, v.Range.l, v.GetName(), i)
		} else {
			decl = fmt.Sprintf("wire %s %s_%d;\n", signedStr, v.GetName(), i)
		}
		topBuilder.WriteString(decl)
	}
	topBuilder.WriteString("    output wire out0;\n")

	for i := 0; i < equalNumber; i++ {
		moduleName := moduleNames[i]
		topBuilder.WriteString(fmt.Sprintf("%s inst_eq%d (\n", moduleName, i))
		for _, v := range g.InputVars {
			topBuilder.WriteString(fmt.Sprintf("    .%s(%s),\n", v.GetName(), v.GetName()))
		}
		for _, clk := range g.ClockVars {
			topBuilder.WriteString(fmt.Sprintf("    .%s(%s),\n", clk.GetName(), clk.GetName()))
		}
		topBuilder.WriteString(fmt.Sprintf("    .%s(%s_%d)\n", outputVar.GetName(), outputVar.GetName(), i))
		topBuilder.WriteString(");\n\n")
	}

	if equalNumber > 1 {
		compareExpr := fmt.Sprintf("assign out0 = (%s_0 == %s_1)", outputVar.GetName(), outputVar.GetName())
		for i := 2; i < equalNumber; i++ {
			compareExpr += fmt.Sprintf(" && (%s_0 == %s_%d)", outputVar.GetName(), outputVar.GetName(), i)
		}
		compareExpr += ";\n"
		topBuilder.WriteString(compareExpr)
	} else {
		topBuilder.WriteString("assign out0 = 1'b1;\n")
	}

	topBuilder.WriteString("endmodule\n\n")

	var finalBuilder strings.Builder
	finalBuilder.WriteString(topBuilder.String())
	for _, mod := range modules {
		finalBuilder.WriteString(mod)
		finalBuilder.WriteString("\n\n")
	}
	return finalBuilder.String()
}
