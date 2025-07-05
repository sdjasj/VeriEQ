package CodeGenerator

import (
	"fmt"
)

type VerilogVarType int

const (
	VarTypeInput VerilogVarType = iota
	VarTypeOutput
	VarTypeWire
	VarTypeReg
)

type VerilogVariable interface {
	GetName() string
	GetType() VerilogVarType
	GenerateString() string
	GetUsageCount() int
	IncrementUsageCount()
}

type BitRange struct {
	l, r int
}

func (b *BitRange) GetWidth() int {
	return b.r - b.l + 1
}

type Variable struct {
	Name       string
	Type       VerilogVarType
	Range      *BitRange
	hasRange   bool
	UsageCount int
	isSigned   bool
}

func NewVar(varType VerilogVarType) *Variable {
	name := GenerateIdentifier(varType)
	return &Variable{
		Name:       name,
		Type:       varType,
		UsageCount: 0,
	}
}

func (v *Variable) GetWidth() int {
	if v.hasRange {
		return v.Range.GetWidth()
	}
	return 1
}

func (v *Variable) GetName() string {
	return v.Name
}

func (v *Variable) GetType() VerilogVarType {
	return v.Type
}

func (v *Variable) GetUsageCount() int {
	return v.UsageCount
}

func (v *Variable) IncrementUsageCount() {
	v.UsageCount++
}

func (v *Variable) GenerateString() string {
	if v.hasRange && v.Range != nil {
		if v.Type == VarTypeWire {
			return fmt.Sprintf("wire [%d:%d] %s", v.Range.l, v.Range.r, v.Name)
		} else if v.Type == VarTypeReg {
			return fmt.Sprintf("reg [%d:%d] %s", v.Range.l, v.Range.r, v.Name)
		} else if v.Type == VarTypeInput {
			return fmt.Sprintf("input [%d:%d] %s", v.Range.l, v.Range.r, v.Name)
		} else if v.Type == VarTypeOutput {
			return fmt.Sprintf("output [%d:%d] %s", v.Range.l, v.Range.r, v.Name)
		}
	} else {
		if v.Type == VarTypeWire {
			return fmt.Sprintf("wire %s", v.Name)
		} else if v.Type == VarTypeReg {
			return fmt.Sprintf("reg %s", v.Name)
		} else if v.Type == VarTypeInput {
			return fmt.Sprintf("input %s", v.Name)
		} else if v.Type == VarTypeOutput {
			return fmt.Sprintf("output %s", v.Name)
		}
	}
	return v.Name
}

func (v *Variable) EquivalentTrans() bool {
	return true
}
