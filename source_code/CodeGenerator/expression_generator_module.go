package CodeGenerator

import (
	"fmt"
	"strings"
)

func (g *ExpressionGenerator) GenerateLoopFreeModule() string {
	if !g.UsePaperInitGen {
		return g.generateLegacyLoopFreeModule()
	}

	parts := g.generateInitialModuleParts()

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

	moduleStr += "\n"

	for _, assign := range parts.combAssigns {
		moduleStr += assign.GenerateString() + "\n"
	}

	moduleStr += parts.outputStr

	if parts.seqBlock != nil {
		moduleStr += parts.seqBlock.GenerateString() + "\n"
	}

	moduleStr += "endmodule\n"

	return moduleStr

}

func (g *ExpressionGenerator) GenerateLoopFreeEquivalentModules(equalNumber int) string {
	if !g.UsePaperInitGen {
		return g.generateLegacyLoopFreeEquivalentModules(equalNumber)
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

	var combinedModule strings.Builder
	for _, module := range modules {
		combinedModule.WriteString(module)
		combinedModule.WriteString("\n\n")
	}

	return combinedModule.String()
}
