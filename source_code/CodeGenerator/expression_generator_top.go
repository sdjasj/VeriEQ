package CodeGenerator

import (
	"fmt"
	"strings"
)

func (g *ExpressionGenerator) GenerateEquivalentModulesWithOneTop(equalNumber int) string {
	if !g.UsePaperInitGen {
		return g.generateLegacyEquivalentModulesWithOneTop(equalNumber)
	}

	parts := g.generateInitialModuleParts()

	for i := range parts.combAssigns {
		parts.combAssigns[i].GetBitWidth()
		parts.combAssigns[i].GetSignedness()
		parts.combAssigns[i].PropagateType(0, false)
	}
	baseAssigns := cloneAssignExpressions(parts.combAssigns)

	baseSeqStr := ""
	if parts.seqBlock != nil {
		baseSeqStr = parts.seqBlock.GenerateString() + "\n"
	}

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

		moduleStr += "\n"

		for _, assign := range currentAssigns {
			moduleStr += assign.GenerateString() + "\n"
		}
		moduleStr += parts.outputStr

		seqStr := baseSeqStr
		if eqIdx > 0 && g.EnableControlFlowEquiv {
			seqStr = g.buildSeqBlockString(parts.seqBlock, true)
		}
		moduleStr += seqStr

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
