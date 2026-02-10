package CodeGenerator

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"
)

func (g *ExpressionGenerator) GenerateInputFile() string {
	res := ""
	for i := 0; i < g.TestBenchTestTime; i++ {
		inputLine := ""
		for j := 0; j < len(g.InputVars); j++ {
			r := 2
			v := g.InputVars[j]
			if v.hasRange {
				r = maxInputRange(v.Range.GetWidth())
			}
			if j != len(g.InputVars)-1 {
				inputLine += fmt.Sprintf("%d ", rand.Intn(r))
			} else {
				inputLine += fmt.Sprintf("%d\n", rand.Intn(r))
			}
		}
		res += inputLine
	}
	return res
}

func maxInputRange(width int) int {
	if width < 1 {
		return 2
	}
	maxShift := strconv.IntSize - 2
	if maxShift < 1 {
		return 2
	}
	if width > maxShift {
		width = maxShift
	}
	r := 1 << width
	if r < 2 {
		return 2
	}
	return r
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func (g *ExpressionGenerator) GetRandomVariable(regOnly bool) *Variable {
	var allVars []*Variable
	for _, v := range g.CurrentDefinedVars {
		allVars = append(allVars, v)
	}

	if len(allVars) == 0 {
		return nil
	}

	if regOnly {
		var regVars []*Variable
		for _, v := range allVars {
			if v.Type == VarTypeReg {
				regVars = append(regVars, v)
			}
		}

		if len(regVars) == 0 {
			return nil
		}

		return regVars[rand.Intn(len(regVars))]
	}

	return allVars[rand.Intn(len(allVars))]
}
