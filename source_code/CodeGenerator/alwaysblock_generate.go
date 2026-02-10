package CodeGenerator

import (
	"fmt"
	"strings"
)

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
				edge := "posedge "
				if !a.ForcePosedge && len(a.ClockPosedge) == len(a.ClockVars) && !a.ClockPosedge[i] {
					edge = "negedge "
				}
				sb.WriteString(edge)
				sb.WriteString(clock.Name)
			}
			resetIsClock := false
			if a.ResetVar != nil {
				for _, clock := range a.ClockVars {
					if clock == a.ResetVar {
						resetIsClock = true
						break
					}
				}
			}
			if a.ResetVar != nil && !resetIsClock {
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
