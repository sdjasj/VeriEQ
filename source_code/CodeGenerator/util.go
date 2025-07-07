package CodeGenerator

import (
	"math/rand"
	"strconv"
)

var wireCnt int
var regCnt int

func GenerateIdentifier(varType VerilogVarType) string {
	if varType == VarTypeWire {
		name := "wire_" + strconv.Itoa(wireCnt)
		wireCnt++
		return name
	} else if varType == VarTypeReg {
		name := "reg_" + strconv.Itoa(regCnt)
		regCnt++
		return name
	}
	return ""
}

func GetRandomRangeFromVar(v *Variable) *BitRange {
	if !v.hasRange {
		return nil
	}
	r := v.Range.l + rand.Intn(v.Range.r-v.Range.l+1)
	l := v.Range.l + rand.Intn(r-v.Range.l+1)
	return &BitRange{
		r: r,
		l: l,
	}
}
