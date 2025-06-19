package CodeGenerator

import (
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

func GetRandomName() (ans string) {
	for i := 0; i < 10; i++ {
		ans += string(rune('a' + rand.Intn(26)))
	}
	return ans
}

// ExpressionGenerator 表达式生成器
type ExpressionGenerator struct {
	Variables               map[string]*Variable // 已定义的变量
	InputVars               []*Variable
	OutputVars              []*Variable
	ClockVars               []*Variable
	InputPortVars           []*Variable
	InputNums               int
	OutputNums              int
	ClockNums               int
	MaxDepth                int // 表达式最大深度
	MaxWidth                int // 表达式最大宽度
	CurrentDefinedVars      []*Variable
	AssignCount             int
	AlwaysCount             int
	ProbabilityOfRange      float64
	ProbabilityOfSigned     float64
	MaxRangeWidth           int
	MinRangeWidth           int
	WireIndex               int
	RegIndex                int
	Name                    string
	TestBenchInputFileName  string
	TestBenchOutputFileName string
	TestBenchTestTime       int
	MaxInputValue           int
}

// NewExpressionGenerator 创建一个新的表达式生成器
func NewExpressionGenerator() *ExpressionGenerator {
	return &ExpressionGenerator{
		Variables:               make(map[string]*Variable),
		MaxDepth:                3, // 默认最大深度
		MaxWidth:                3, // 默认最大宽度
		AssignCount:             20,
		AlwaysCount:             1,
		ProbabilityOfRange:      0.8,
		ProbabilityOfSigned:     0.5,
		MaxRangeWidth:           30,
		MinRangeWidth:           1,
		InputNums:               10,
		OutputNums:              1,
		ClockNums:               2,
		Name:                    "top",
		TestBenchInputFileName:  "input.txt",
		TestBenchOutputFileName: "output.txt",
		TestBenchTestTime:       20,
	}
	//return &ExpressionGenerator{
	//	Variables:               make(map[string]*Variable),
	//	MaxDepth:                5, // 默认最大深度
	//	MaxWidth:                1, // 默认最大宽度
	//	AssignCount:             1,
	//	AlwaysCount:             1,
	//	ProbabilityOfRange:      0.7,
	//	MaxRangeWidth:           30,
	//	InputNums:               2,
	//	OutputNums:              2,
	//	MinRangeWidth:           20,
	//	ClockNums:               2,
	//	Name:                    GetRandomName(),
	//	TestBenchInputFileName:  "input.txt",
	//	TestBenchOutputFileName: "output.txt",
	//	TestBenchTestTime:       20,
	//}
}

// SetMaxDepth 设置表达式最大深度
func (g *ExpressionGenerator) SetMaxDepth(depth int) {
	g.MaxDepth = depth
}

// SetMaxWidth 设置表达式最大宽度
func (g *ExpressionGenerator) SetMaxWidth(width int) {
	g.MaxWidth = width
}

// AddVariable 添加一个变量
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

// AddWireVariable 添加一个线网变量
func (g *ExpressionGenerator) AddWireVariable(name string) *Variable {
	if name == "" {
		name = fmt.Sprintf("wire_%d", g.WireIndex)
		g.WireIndex++
	}
	return g.AddVariable(name, VarTypeWire)
}

// AddRegVariable 添加一个寄存器变量
func (g *ExpressionGenerator) AddRegVariable(name string) *Variable {
	if name == "" {
		name = fmt.Sprintf("reg_%d", g.RegIndex)
		g.RegIndex++
	}
	return g.AddVariable(name, VarTypeReg)
}

// GenerateExpression 生成一个表达式
func (g *ExpressionGenerator) GenerateExpression(depth int) Expression {
	// 重置已使用变量

	// 如果深度为0，生成一个基本表达式
	if depth <= 0 {
		return g.generateBasicExpression()
	}

	// 随机选择表达式类型
	exprType := rand.Intn(6) // 增加位拼接表达式的选项

	switch exprType {
	case 0:
		// 生成二元表达式
		return g.generateBinaryExpression(depth)
	case 1:
		// 生成一元表达式
		return g.generateUnaryExpression(depth)
	case 2:
		// 生成三元表达式
		return g.generateTernaryExpression(depth)
	case 3:
		// 生成位拼接表达式
		return g.generateConcatenationExpression(depth)
	case 4:
		// 生成变量表达式
		return g.generateVariableExpression()
	case 5:
		// 生成数字表达式
		return g.generateNumberExpression()

	default:
		return g.generateBasicExpression()
	}
}

// generateBasicExpression 生成基本表达式（变量或数字）
func (g *ExpressionGenerator) generateBasicExpression() Expression {
	// 50%概率生成变量，50%概率生成数字
	if rand.Intn(2) == 0 && len(g.CurrentDefinedVars) > 0 {
		return g.generateVariableExpression()
	}
	return g.generateNumberExpression()
}

// generateBinaryExpression 生成二元表达式
func (g *ExpressionGenerator) generateBinaryExpression(depth int) Expression {
	// 可用的二元运算符
	operators := []string{
		// 算术运算符
		"+", "-", "*", "/", "%",
		// 逻辑运算符
		"&&", "||",
		// 位运算符
		"&", "|", "^", "~^", "~&", "~|",
		// 比较运算符
		"==", "!=", "===", "!==", "<", "<=", ">", ">=",
		// 移位运算符
		"<<", ">>", "<<<", ">>>",
	}
	// 随机选择一个运算符
	operator := operators[rand.Intn(len(operators))]

	// 生成左右操作数
	left := g.GenerateExpression(depth - 1)
	right := g.GenerateExpression(depth - 1)

	return &BinaryExpression{
		Left:     left,
		Right:    right,
		Operator: operator,
	}
}

// generateUnaryExpression 生成一元表达式
func (g *ExpressionGenerator) generateUnaryExpression(depth int) Expression {
	// 可用的一元运算符
	operators := []string{"!", "~", "-"}

	// 随机选择一个运算符
	operator := operators[rand.Intn(len(operators))]

	// 生成操作数
	operand := g.GenerateExpression(depth - 1)

	return &UnaryExpression{
		Operand:  operand,
		Operator: operator,
	}
}

// generateTernaryExpression 生成三元表达式
func (g *ExpressionGenerator) generateTernaryExpression(depth int) Expression {
	// 生成条件、真表达式和假表达式
	condition := g.GenerateExpression(depth - 1)
	trueExpr := g.GenerateExpression(depth - 1)
	falseExpr := g.GenerateExpression(depth - 1)

	return &TernaryExpression{
		Condition: condition,
		TrueExpr:  trueExpr,
		FalseExpr: falseExpr,
	}
}

// generateVariableExpression 生成变量表达式
func (g *ExpressionGenerator) generateVariableExpression() Expression {
	// 获取所有可用变量

	if len(g.CurrentDefinedVars) == 0 {
		// 如果没有可用变量，生成一个数字表达式
		return g.generateNumberExpression()
	}

	// 随机选择一个变量
	variable := g.CurrentDefinedVars[rand.Intn(len(g.CurrentDefinedVars))]
	if variable.hasRange && rand.Float64() > 0.8 {
		r := variable.Range.l + rand.Intn(variable.Range.r-variable.Range.l+1)
		l := variable.Range.l + rand.Intn(r-variable.Range.l+1)
		return &VariableExpression{
			Var:      variable,
			hasRange: true,
			Range: &BitRange{
				l: l,
				r: r,
			},
		}
	}

	return &VariableExpression{
		Var:      variable,
		hasRange: false,
	}
}

// bitsNeeded 返回 value 在二进制中所需的最少位数
func bitsNeeded(value int) int {
	if value == 0 {
		return 1
	}
	return int(math.Floor(math.Log2(float64(value)))) + 1
}

// parseInt 返回给定进制字符串的十进制值
func parseInt(valueStr string, base int) int {
	val, _ := strconv.ParseInt(valueStr, base, 64)
	return int(val)
}

func GenerateRandomNumber() string {
	format := rand.Intn(4)

	var numStr string
	switch format {
	case 0:
		// 十进制
		value := rand.Intn(1000)
		width := bitsNeeded(value)
		numStr = fmt.Sprintf("%d'd%d", width, value)

	case 1:
		// 二进制
		length := rand.Intn(10) + 1 // 1 到 10 位
		binStr := ""
		for i := 0; i < length; i++ {
			binStr += fmt.Sprintf("%d", rand.Intn(2))
		}
		numStr = fmt.Sprintf("%d'b%s", length+rand.Intn(6), binStr)

	case 2:
		// 八进制
		digits := rand.Intn(4) + 1
		octStr := ""
		for i := 0; i < digits; i++ {
			octStr += fmt.Sprintf("%o", rand.Intn(8))
		}
		value := parseInt(octStr, 8)
		width := bitsNeeded(value)
		numStr = fmt.Sprintf("%d'o%s", width, octStr)

	case 3:
		// 十六进制
		digits := rand.Intn(3) + 1
		hexStr := ""
		for i := 0; i < digits; i++ {
			hexStr += fmt.Sprintf("%x", rand.Intn(16))
		}
		value := parseInt(hexStr, 16)
		width := bitsNeeded(value)
		numStr = fmt.Sprintf("%d'h%s", width, hexStr)
	}
	return numStr
}

// generateNumberExpression 生成 Verilog 数字表达式，位宽统一使用实际的二进制位数
func (g *ExpressionGenerator) generateNumberExpression() Expression {
	return &NumberExpression{
		Value: RandomConstNumber(),
	}
}

// GenerateLoopFreeExpression 生成一个无环路的表达式
func (g *ExpressionGenerator) GenerateLoopFreeExpression(depth int) Expression {
	// 尝试生成表达式，直到找到一个无环路的表达式
	return g.GenerateExpression(depth)

}

// GenerateLoopFreeAssignment 生成一个无环路的赋值语句
func (g *ExpressionGenerator) GenerateLoopFreeAssignment(target *Variable) *AssignExpression {
	// 获取所有可用变量

	if len(g.CurrentDefinedVars) == 0 {
		return nil
	}

	// 生成无环路表达式
	expr := g.GenerateLoopFreeExpression(g.MaxDepth)
	if target == nil {
		target = g.AddWireVariable("")
	}
	//g.CurrentDefinedVars = append(g.CurrentDefinedVars, target)

	return &AssignExpression{
		Operand1: target,
		Right:    expr,
	}
}

func (g *ExpressionGenerator) GenerateLoopFreeOutputAssignment(target *Variable) *AssignExpression {
	// 获取所有可用变量

	if len(g.CurrentDefinedVars) == 0 {
		return nil
	}

	// 生成无环路表达式
	expr := g.GenerateLoopFreeExpression(g.MaxDepth)
	if target == nil {
		target = g.AddWireVariable("")
	}

	return &AssignExpression{
		Operand1: target,
		Right:    expr,
	}
}

func (g *ExpressionGenerator) Clear() {
	g.CurrentDefinedVars = make([]*Variable, 0)
	g.OutputVars = make([]*Variable, 0)
	g.InputVars = make([]*Variable, 0)
	g.ClockVars = make([]*Variable, 0)

}

func (g *ExpressionGenerator) ApplyEquivalenceTransforms(assigns []*AssignExpression) []*AssignExpression {
	transformed := make([]*AssignExpression, len(assigns))
	for i, a := range assigns {
		transformed[i] = a.EquivalentTrans().(*AssignExpression)
	}
	return transformed
}

// GenerateLoopFreeModule 生成一个无环路的完整模块
func (g *ExpressionGenerator) GenerateLoopFreeModule() string {
	// 清空变量
	g.Clear()

	isInput := make(map[*Variable]struct{})
	isOutput := make(map[*Variable]struct{})
	// 生成输入变量
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

	// 生成输出变量
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
	for i := 0; i < g.AssignCount; i++ {
		g.CurrentDefinedVars = append(g.CurrentDefinedVars, assignExpressions[i].Operand1)
	}

	alwaysBlocks := make([]*AlwaysBlock, 0)

	randomPick := func(slice []*Variable, count int) []*Variable {

		indices := rand.Perm(len(slice))[:count] // 生成打乱后的索引，然后取前count个
		result := make([]*Variable, count)
		for i, idx := range indices {
			result[i] = slice[idx]
		}
		return result
	}
	for i := 0; i < g.AlwaysCount; i++ {
		//alwaysClocks := randomPick(g.ClockVars, max(2, rand.Intn(len(g.ClockVars))))
		alwaysClocks := randomPick(g.ClockVars, 2)
		alwaysBlocks = append(alwaysBlocks, RandomAlwaysBlockWithTargets(g, alwaysClocks, g.MaxDepth, g.MaxWidth))
	}

	//for i := 0; i < g.OutputNums; i++ {
	//	assignExpressions = append(assignExpressions, g.GenerateLoopFreeOutputAssignment(g.OutputVars[i]))
	//}
	outputVar := g.OutputVars[0]
	outputStr := fmt.Sprintf("    assign %s = ", outputVar.Name)
	flag := false
	for i := 0; i < len(g.CurrentDefinedVars); i++ {
		v := g.CurrentDefinedVars[i]
		if _, ok := isInput[v]; ok {
			continue
		} else {
			if flag {
				outputStr += fmt.Sprintf("+ %s ", v.Name)
			} else {
				outputStr += fmt.Sprintf("%s ", v.Name)
				flag = true
			}
		}
	}
	outputStr += ";\n"
	g.CurrentDefinedVars = append(g.CurrentDefinedVars, outputVar)
	//g.CurrentDefinedVars = append(g.CurrentDefinedVars, g.OutputVars...)

	//生成整个模块
	// 生成模块声明
	moduleStr := fmt.Sprintf("`timescale 1ns/1ps\nmodule %s (", g.Name)

	// 生成输入端口
	for i, input := range g.InputVars {
		if i == len(g.InputVars)-1 {
			moduleStr += fmt.Sprintf("%s, ", input.GetName())
		} else {
			moduleStr += fmt.Sprintf("%s, ", input.GetName())
		}
	}

	// 生成输出端口
	for i, output := range g.OutputVars {
		if i == len(g.OutputVars)-1 {
			moduleStr += fmt.Sprintf("%s ", output.GetName())
		} else {
			moduleStr += fmt.Sprintf("%s, ", output.GetName())
		}
	}

	moduleStr += ");\n\n"

	// 生成wire和reg声明
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
			if _, ok := isInput[v]; ok {
				s = "input " + s
			} else if _, ok := isOutput[v]; ok {
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
			if _, ok := isInput[v]; ok {
				s = "input " + s
			} else if _, ok := isOutput[v]; ok {
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

	for _, assign := range assignExpressions {
		moduleStr += assign.GenerateString() + "\n"
	}
	// 生成assign语句

	moduleStr += outputStr

	// 生成always块
	for _, always := range alwaysBlocks {
		moduleStr += always.GenerateString() + "\n"
	}

	// 生成模块结束
	moduleStr += "endmodule\n"

	return moduleStr

}

// GenerateLoopFreeEquivalentModules 生成多个等价的完整无环路模块
func (g *ExpressionGenerator) GenerateLoopFreeEquivalentModules(equalNumber int) []string {
	// 清空并初始化变量
	g.Clear()
	isInput := make(map[*Variable]struct{})
	isOutput := make(map[*Variable]struct{})

	// 生成输入变量
	for i := 0; i < g.InputNums; i++ {
		varName := fmt.Sprintf("in%d", i)
		variable := g.AddWireVariable(varName)
		g.CurrentDefinedVars = append(g.CurrentDefinedVars, variable)
		g.InputVars = append(g.InputVars, variable)
		g.InputPortVars = append(g.InputPortVars, variable)
		isInput[variable] = struct{}{}
	}

	// 生成时钟变量
	for i := 0; i < g.ClockNums; i++ {
		varName := fmt.Sprintf("clock_%d", i)
		variable := g.AddVariableNotArray(varName, VarTypeWire)
		g.InputVars = append(g.InputVars, variable)
		g.ClockVars = append(g.ClockVars, variable)
		isInput[variable] = struct{}{}
	}

	// 生成输出变量
	for i := 0; i < g.OutputNums; i++ {
		varName := fmt.Sprintf("out%d", i)
		variable := g.AddWireVariable(varName)
		g.OutputVars = append(g.OutputVars, variable)
		isOutput[variable] = struct{}{}
	}

	// 生成初始 assign 表达式（未做等价变换）
	assignExpressions := make([]*AssignExpression, 0)
	for i := 0; i < g.AssignCount; i++ {
		assignExpressions = append(assignExpressions, g.GenerateLoopFreeAssignment(nil))
	}
	for _, assign := range assignExpressions {
		g.CurrentDefinedVars = append(g.CurrentDefinedVars, assign.Operand1)
	}

	// 生成 always 块
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

	// 统一输出表达式
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

	// 每个等价版本的模块都保存在这个切片中
	modules := make([]string, 0, equalNumber)

	for i, _ := range assignExpressions {
		assignExpressions[i].GetBitWidth()
		assignExpressions[i].GetSignedness()
	}

	alwaysStr := ""
	for _, v := range alwaysBlocks {
		alwaysStr += v.GenerateString() + "\n"
	}

	for eqIdx := 0; eqIdx < equalNumber; eqIdx++ {
		// 对 assign 表达式做等价变换
		if eqIdx > 0 {
			g.ApplyEquivalenceTransforms(assignExpressions)
		}

		// 模块声明头
		moduleStr := fmt.Sprintf("`timescale 1ns/1ps\nmodule %s_eq%d (", g.Name, eqIdx)

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

		// 端口声明
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
			if _, ok := isInput[v]; ok {
				decl = "input " + decl
			} else if _, ok := isOutput[v]; ok {
				decl = "output " + decl
			}
			moduleStr += decl
		}

		for _, clk := range g.ClockVars {
			moduleStr += fmt.Sprintf("input %s;\n", clk.GetName())
			moduleStr += fmt.Sprintf("wire %s;\n", clk.GetName())
		}

		moduleStr += "\n"

		// assign 表达式部分
		for _, assign := range assignExpressions {
			moduleStr += assign.GenerateString() + "\n"
		}
		moduleStr += outputStr

		// always 块
		moduleStr += alwaysStr

		moduleStr += "endmodule\n"
		modules = append(modules, moduleStr)
	}

	return modules
}

func (g *ExpressionGenerator) GenerateTb() string {
	tbStr := fmt.Sprintf("`timescale 1ns/1ps\n\nmodule tb_dut_module;\n\n    parameter NUM_VECTORS = %d;  // 你想读取的行数\n\n",
		g.TestBenchTestTime)
	for _, v := range g.InputVars {
		signedStr := ""
		if v.isSigned {
			signedStr = "signed"
		}
		if v.hasRange {
			tbStr += fmt.Sprintf("    reg %s [%d:%d] %s;\n", signedStr, v.Range.r, v.Range.l, v.Name)
		} else {
			tbStr += fmt.Sprintf("reg %s %s;\n", signedStr, v.Name)
		}
	}
	for _, v := range g.OutputVars {
		signedStr := ""
		if v.isSigned {
			signedStr = "signed"
		}
		if v.hasRange {
			tbStr += fmt.Sprintf("    wire %s [%d:%d] %s;\n", signedStr, v.Range.r, v.Range.l, v.Name)
		} else {
			tbStr += fmt.Sprintf("wire %s %s;\n", signedStr, v.Name)
		}
	}
	inputPort := ""
	for _, v := range g.InputVars {
		inputPort += fmt.Sprintf(".%s(%s), ", v.Name, v.Name)
	}
	outputPort := ""
	for i := 0; i < len(g.OutputVars); i++ {
		v := g.OutputVars[i]
		if i != len(g.OutputVars)-1 {
			outputPort += fmt.Sprintf(".%s(%s), ", v.Name, v.Name)
		} else {
			outputPort += fmt.Sprintf(".%s(%s)", v.Name, v.Name)
		}
	}
	initInput := "#2000;\n        "
	for _, v := range g.InputPortVars {
		initInput += fmt.Sprintf("%s = 0; ", v.Name)
	}
	initInput += "\n        #2000;\n        "
	for _, v := range g.ClockVars {
		initInput += fmt.Sprintf("%s = 0; ", v.Name)
	}

	initRegStr := ""
	for i := 0; i < len(g.CurrentDefinedVars); i++ {
		v := g.CurrentDefinedVars[i]
		if v.Type != VarTypeReg {
			continue
		}
		initRegStr += fmt.Sprintf("    uut.%s = 0;\n", v.Name)
	}

	scanStr := "\""
	for i := 0; i < len(g.InputPortVars); i++ {
		if i != len(g.InputPortVars)-1 {
			scanStr += "%d "
		} else {
			scanStr += "%d "
		}
	}
	scanStr += "\", "
	for i := 0; i < len(g.InputPortVars); i++ {
		v := g.InputPortVars[i]
		if i != len(g.InputPortVars)-1 {
			scanStr += fmt.Sprintf("%s ,", v.Name)
		} else {
			scanStr += fmt.Sprintf("%s ", v.Name)
		}

	}
	scanClockStr := "\""
	for i := 0; i < len(g.ClockVars); i++ {
		if i != len(g.ClockVars)-1 {
			scanClockStr += "%d "
		} else {
			scanClockStr += "%d\\n"
		}
	}
	scanClockStr += "\", "
	for i := 0; i < len(g.ClockVars); i++ {
		v := g.ClockVars[i]
		if i != len(g.ClockVars)-1 {
			scanClockStr += fmt.Sprintf("%s ,", v.Name)
		} else {
			scanClockStr += fmt.Sprintf("%s ", v.Name)
		}

	}

	hashStr := ""
	for i := 0; i < len(g.OutputVars); i++ {
		v := g.OutputVars[i]
		if i != len(g.OutputVars)-1 {
			hashStr += fmt.Sprintf("%s + ", v.Name)
		} else {
			hashStr += fmt.Sprintf("%s;", v.Name)
		}

	}
	outfmtStr := ""
	for i := 0; i < len(g.OutputVars); i++ {
		if i != len(g.OutputVars)-1 {
			outfmtStr += fmt.Sprintf("%s:%%0d ", g.OutputVars[i].Name)
		} else {
			outfmtStr += fmt.Sprintf("%s:%%0d\\n", g.OutputVars[i].Name)
		}
	}
	outVarStr := ""
	for i := 0; i < len(g.OutputVars); i++ {
		if i != len(g.OutputVars)-1 {
			outVarStr += fmt.Sprintf("%s, ", g.OutputVars[i].Name)
		} else {
			outVarStr += fmt.Sprintf("%s", g.OutputVars[i].Name)
		}
	}

	tbStr += fmt.Sprintf(`
    %s uut  (
		%s
		%s
    );
    integer fin, fout;
    integer i, status;
    reg [31:0] output_hash;
    initial begin

        fin = $fopen("%s", "r");
        if (fin == 0) begin
            $display("ERROR: Cannot open %s");
            $finish;
        end

        fout = $fopen("%s", "w");
        if (fout == 0) begin
            $display("ERROR: Cannot open %s for writing");
            $finish;
        end

		%s
		#2000;
		%s

		#2000;
        for (i = 0; i < NUM_VECTORS; i = i + 1) begin
            status = $fscanf(fin, %s);
			#2000
			status = status + $fscanf(fin, %s);

            if (status < %d) begin
                $display("WARNING: File doesn't have enough lines or format error at line %%0d", i);
   
                $finish;
            end


            #2000;
			$fwrite(fout, "%s", %s);
        end

        $fclose(fin);
        $fclose(fout);
        $display("Simulation finished after %%0d vectors.", NUM_VECTORS);
        $finish;
    end

endmodule
`, g.Name, inputPort, outputPort, g.TestBenchInputFileName,
		g.TestBenchInputFileName, g.TestBenchOutputFileName, g.TestBenchOutputFileName,
		initInput, initRegStr, scanStr, scanClockStr, len(g.InputVars), outfmtStr, outVarStr)
	return tbStr
}

func genAllOnesHex(n int) string {
	if n <= 0 {
		return "0x0"
	}

	// 构造 big.Int 值：(1 << n) - 1
	ones := new(big.Int).Lsh(big.NewInt(1), uint(n))
	ones.Sub(ones, big.NewInt(1))

	return fmt.Sprintf("0x%x", ones)
}

func (g *ExpressionGenerator) GenerateCXXRTLTestBench() string {
	valueIdx := 0
	inStr := ""
	initStr := ""
	for i := 0; i < len(g.InputPortVars); i++ {
		v := g.InputPortVars[i]
		var width int
		if v.hasRange {
			width = v.Range.GetWidth()
		} else {
			width = 1
		}
		inStr += fmt.Sprintf("mod->p_in%d  = cxxrtl::value<%d>(values[%d]);\n", i, width, valueIdx)
		initStr += fmt.Sprintf("mod->p_in%d  = cxxrtl::value<%d>(0u);\n", i, width)
		valueIdx++
	}
	inStr += "\nmod->step();\n"
	//clock vars
	for i := 0; i < len(g.ClockVars); i++ {
		v := g.ClockVars[i]
		var width int
		if v.hasRange {
			width = v.Range.GetWidth()
		} else {
			width = 1
		}

		inStr += fmt.Sprintf("mod->p_clock__%d  = cxxrtl::value<%d>(values[%d]);\n", i, width, valueIdx)
		initStr += fmt.Sprintf("mod->p_clock__%d  = cxxrtl::value<%d>(0u);\n", i, width)
		valueIdx++
	}
	outStr := ""
	for i := 0; i < len(g.OutputVars); i++ {
		v := g.OutputVars[i]
		var width int
		if v.hasRange {
			width = v.Range.GetWidth()
		} else {
			width = 1
		}
		bitWidth := genAllOnesHex(width)
		if v.isSigned {
			if i != len(g.OutputVars)-1 {
				outStr += fmt.Sprintf("<< \"%s:\" << truncate_signed(mod->p_out%d) << \" \"", g.OutputVars[i].Name, i)
			} else {
				outStr += fmt.Sprintf("<< \"%s:\" << truncate_signed(mod->p_out%d) << std::endl;", g.OutputVars[i].Name, i)
			}

		} else {
			if i != len(g.OutputVars)-1 {
				outStr += fmt.Sprintf("<< \"%s:\" << (mod->p_out%d.get<uint32_t>() & %s)  << \" \"", g.OutputVars[i].Name, i, bitWidth)
			} else {
				outStr += fmt.Sprintf("<< \"%s:\" << (mod->p_out%d.get<uint32_t>() & %s) << std::endl;", g.OutputVars[i].Name, i, bitWidth)
			}
		}

	}
	tbStr := fmt.Sprintf(`
#include <fstream>
#include <iostream>
#include <sstream>
#include <vector>
#include <string>
#include "test.cpp"
    
using namespace std;
using namespace cxxrtl_design;

template <std::size_t N>
int32_t truncate_signed(cxxrtl::value<N> val) {
    int32_t raw = val.template get<int32_t>();
    return (raw << (32 - N)) >> (32 - N);
}


std::vector<uint32_t> parse_line(const std::string &line) {
	std::stringstream ss(line);
	std::vector<uint32_t> values;
	uint32_t val;
	while (ss >> val) {
		values.push_back(val);
	}
	return values;
}

int main() {
	// 创建模块
	std::unique_ptr<p_%s> mod = std::make_unique<p_%s>();
	std::ofstream ofs("output.txt");
	// 打开输入文件
	std::ifstream infile("input.txt");
	if (!infile) {
		std::cerr << "Cannot open input.txt" << std::endl;
		return 1;
	}

	std::string line;
	int line_num = 0;

	%s

	mod->step();
	mod->step();
	mod->step();
	while (std::getline(infile, line)) {
		line_num++;
		auto values = parse_line(line);

		// 前 21 个输入端口（位宽根据你之前的错误信息确定）
		%s

		// 模拟一次
		mod->step();
		mod->step();
		mod->step();

		// 输出结果（仅示例，按实际输出端口定）

		if (!ofs) return 1;
		ofs %s
	}

	return 0;
}

`, g.Name, g.Name, initStr, inStr, outStr)
	return tbStr
}

func (g *ExpressionGenerator) GenerateInputFile() string {
	res := ""
	for i := 0; i < g.TestBenchTestTime; i++ {
		inputLine := ""
		for j := 0; j < len(g.InputVars); j++ {
			r := 2
			v := g.InputVars[j]
			if v.hasRange {
				r = 1 << (v.Range.GetWidth())
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

// 初始化随机数生成器
func init() {
	rand.Seed(time.Now().UnixNano())
}

func (g *ExpressionGenerator) GetRandomVariable(regOnly bool) *Variable {
	// 获取所有变量
	var allVars []*Variable
	for _, v := range g.CurrentDefinedVars {
		allVars = append(allVars, v)
	}

	// 如果没有变量，返回nil
	if len(allVars) == 0 {
		return nil
	}

	// 如果只需要寄存器变量，过滤出寄存器变量
	if regOnly {
		var regVars []*Variable
		for _, v := range allVars {
			if v.Type == VarTypeReg {
				regVars = append(regVars, v)
			}
		}

		// 如果没有寄存器变量，返回nil
		if len(regVars) == 0 {
			return nil
		}

		// 随机返回一个寄存器变量
		return regVars[rand.Intn(len(regVars))]
	}

	// 随机返回一个变量
	return allVars[rand.Intn(len(allVars))]
}

// generateConcatenationExpression 生成位拼接表达式
func (g *ExpressionGenerator) generateConcatenationExpression(depth int) Expression {
	// 随机生成2-4个表达式进行拼接
	numExprs := rand.Intn(3) + 2
	exprs := make([]Expression, 0, numExprs)

	for i := 0; i < numExprs; i++ {
		// 20%的概率生成重复拼接表达式
		if rand.Float64() < 0.2 {
			// 生成重复次数(1-8)
			count := rand.Intn(8) + 1
			countExpr := &NumberExpression{
				Value: RandomConstNumberWithBitWidth(count, false),
			}
			// 生成要重复的表达式
			expr := g.GenerateExpression(depth - 1)
			exprs = append(exprs, &ReplicationExpression{
				Count:      countExpr,
				Expression: expr,
			})
		} else {
			expr := g.GenerateExpression(depth - 1)
			exprs = append(exprs, expr)
		}
	}

	return &ConcatenationExpression{
		Expressions: exprs,
	}
}

func (g *ExpressionGenerator) GenerateEquivalenceCheckTb(equalNumber int) string {
	tbStr := fmt.Sprintf("`timescale 1ns/1ps\n\nmodule tb_equiv_check;\n\nparameter NUM_VECTORS = %d;\n\n", g.TestBenchTestTime)

	// 输入端口声明
	for _, v := range g.InputVars {
		signed := ""
		if v.isSigned {
			signed = "signed "
		}
		if v.hasRange {
			tbStr += fmt.Sprintf("reg %s[%d:%d] %s;\n", signed, v.Range.r, v.Range.l, v.Name)
		} else {
			tbStr += fmt.Sprintf("reg %s%s;\n", signed, v.Name)
		}
	}
	// 输出端口声明（每个模块一组）
	for i := 0; i < equalNumber; i++ {
		for _, v := range g.OutputVars {
			signed := ""
			if v.isSigned {
				signed = "signed "
			}
			name := fmt.Sprintf("%s_eq%d", v.Name, i)
			if v.hasRange {
				tbStr += fmt.Sprintf("wire %s[%d:%d] %s;\n", signed, v.Range.r, v.Range.l, name)
			} else {
				tbStr += fmt.Sprintf("wire %s%s;\n", signed, name)
			}
		}
	}

	// 实例化所有等价模块
	for i := 0; i < equalNumber; i++ {
		instName := fmt.Sprintf("uut_eq%d", i)
		moduleName := fmt.Sprintf("%s_eq%d", g.Name, i)
		tbStr += fmt.Sprintf("\n%s %s (\n", moduleName, instName)
		// 输入连接
		for _, v := range g.InputVars {
			tbStr += fmt.Sprintf("    .%s(%s),\n", v.Name, v.Name)
		}
		// 输出连接
		for j, v := range g.OutputVars {
			suffix := ""
			if j == len(g.OutputVars)-1 {
				suffix = ""
			} else {
				suffix = ","
			}
			tbStr += fmt.Sprintf("    .%s(%s_eq%d)%s\n", v.Name, v.Name, i, suffix)
		}
		tbStr += ");\n"
	}

	// 测试逻辑（读取 input.txt，执行并比较输出）
	tbStr += `
integer i, fin;
initial begin
    fin = $fopen("input.txt", "r");
    if (fin == 0) begin
        $display("Cannot open input.txt");
        $finish;
    end

    for (i = 0; i < NUM_VECTORS; i = i + 1) begin
        // 输入扫描
`
	for _, v := range g.InputVars {
		tbStr += fmt.Sprintf("        $fscanf(fin, \"%%d\", %s);\n", v.Name)
	}

	tbStr += "        #10;\n"

	// 输出比较逻辑
	for i := 1; i < equalNumber; i++ {
		for _, v := range g.OutputVars {
			tbStr += fmt.Sprintf("        if (%s_eq0 !== %s_eq%d) $display(\"Mismatch on %%s at vector %%0d: %%0d vs %%0d\", \"%s\", i, %s_eq0, %s_eq%d);\n",
				v.Name, v.Name, i, v.Name, v.Name, v.Name, i)
		}
	}
	tbStr += `
    end
    $fclose(fin);
    $display("Test completed.");
    $finish;
end

endmodule
`
	return tbStr
}

func (g *ExpressionGenerator) GenerateCXXRTLEquivalenceCheck(equalNumber int) string {
	// 预备初始化代码、输入设置代码、输出比较代码
	initStrs := make([]string, equalNumber)
	inStrs := make([]string, equalNumber)
	outStrs := make([]string, equalNumber)
	modNames := make([]string, equalNumber)
	for i := 0; i < equalNumber; i++ {
		modNames[i] = fmt.Sprintf("mod%d", i)
	}

	valueIdx := 0
	for i, mod := range modNames {
		for _, v := range g.InputPortVars {
			width := 1
			if v.hasRange {
				width = v.Range.GetWidth()
			}
			inStrs[i] += fmt.Sprintf("%s->p_%s = cxxrtl::value<%d>(values[%d]);\n", mod, v.Name, width, valueIdx)
			initStrs[i] += fmt.Sprintf("%s->p_%s = cxxrtl::value<%d>(0u);\n", mod, v.Name, width)
			valueIdx++
		}
		for j, clk := range g.ClockVars {
			width := 1
			if clk.hasRange {
				width = clk.Range.GetWidth()
			}
			inStrs[i] += fmt.Sprintf("%s->p_clock__%d = cxxrtl::value<%d>(values[%d]);\n", mod, j, width, valueIdx)
			initStrs[i] += fmt.Sprintf("%s->p_clock__%d = cxxrtl::value<%d>(0u);\n", mod, j, width)
			valueIdx++
		}
	}

	// 输出比较
	for _, v := range g.OutputVars {
		for i := 1; i < equalNumber; i++ {
			outStrs[0] += fmt.Sprintf("if (%s->p_%s != %s->p_%s) std::cerr << \"Mismatch on %s: \" << %s->p_%s << \" vs \" << %s->p_%s << std::endl;\n",
				modNames[0], v.Name, modNames[i], v.Name,
				v.Name,
				modNames[0], v.Name, modNames[i], v.Name)
		}
	}

	tb := "#include <iostream>\n#include <fstream>\n#include <sstream>\n#include <vector>\n#include <memory>\n#include \"test.cpp\"\n\n"
	tb += "using namespace std;\nusing namespace cxxrtl_design;\n\n"
	tb += "std::vector<uint32_t> parse_line(const std::string &line) {\n"
	tb += "    std::stringstream ss(line);\n    std::vector<uint32_t> values;\n    uint32_t val;\n"
	tb += "    while (ss >> val) values.push_back(val);\n    return values;\n}\n\n"
	tb += "int main() {\n"
	for _, mod := range modNames {
		tb += fmt.Sprintf("    std::unique_ptr<p_%s> %s = std::make_unique<p_%s>();\n", g.Name, mod, g.Name)
	}
	tb += "\n    std::ifstream infile(\"input.txt\");\n    if (!infile) return 1;\n"
	tb += "    std::string line;\n    int line_num = 0;\n\n"

	// 初始化所有模块
	for _, s := range initStrs {
		tb += s
	}
	tb += "    for (int i = 0; std::getline(infile, line); ++i) {\n"
	tb += "        auto values = parse_line(line);\n"
	for _, s := range inStrs {
		tb += s
		tb += "        " + strings.Split(s, "\n")[0][:5] + "->step();\n"
	}
	for _, s := range outStrs {
		tb += s
	}
	tb += "    }\n    return 0;\n}\n"

	return tb
}
