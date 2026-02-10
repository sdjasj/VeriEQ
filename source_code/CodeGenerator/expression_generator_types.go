package CodeGenerator

import "math/rand"

func GetRandomName() (ans string) {
	for i := 0; i < 10; i++ {
		ans += string(rune('a' + rand.Intn(26)))
	}
	return ans
}

type ExpressionGenerator struct {
	Variables               map[string]*Variable
	InputVars               []*Variable
	OutputVars              []*Variable
	ClockVars               []*Variable
	InputPortVars           []*Variable
	InputNums               int
	OutputNums              int
	ClockNums               int
	MaxDepth                int
	MaxWidth                int
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
	UsePaperInitGen         bool
	EnableControlFlowEquiv  bool
	EnableXInputs           bool
}

var DefaultUsePaperInitGen = true
var DefaultEnableControlFlowEquiv = false
var DefaultEnableXInputs = false

func NewExpressionGenerator() *ExpressionGenerator {
	return &ExpressionGenerator{
		Variables:               make(map[string]*Variable),
		MaxDepth:                5,
		MaxWidth:                5,
		AssignCount:             20,
		AlwaysCount:             1,
		ProbabilityOfRange:      0.8,
		ProbabilityOfSigned:     0.5,
		MaxRangeWidth:           30,
		MinRangeWidth:           1,
		InputNums:               20,
		OutputNums:              1,
		ClockNums:               2,
		Name:                    "top",
		TestBenchInputFileName:  "input.txt",
		TestBenchOutputFileName: "output.txt",
		TestBenchTestTime:       20,
		UsePaperInitGen:         DefaultUsePaperInitGen,
		EnableControlFlowEquiv:  DefaultEnableControlFlowEquiv,
		EnableXInputs:           DefaultEnableXInputs,
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

func (g *ExpressionGenerator) SetMaxDepth(depth int) {
	g.MaxDepth = depth
}

func (g *ExpressionGenerator) SetMaxWidth(width int) {
	g.MaxWidth = width
}
