package CodeGenerator

import (
	"fmt"
	"math/big"
	"strings"
)

func genAllOnesHex(n int) string {
	if n <= 0 {
		return "0x0"
	}

	ones := new(big.Int).Lsh(big.NewInt(1), uint(n))
	ones.Sub(ones, big.NewInt(1))

	return fmt.Sprintf("0x%x", ones)
}

func (g *ExpressionGenerator) GenerateCXXRTLMultiModuleTestBench(equalNumber int) string {
	valueIdx := 0
	inStr := ""
	initStr := ""
	modDecl := ""
	initMods := ""

	for i := 0; i < equalNumber; i++ {
		modDecl += fmt.Sprintf("std::unique_ptr<p_top__eq%d> mod%d;\n", i, i)
		initMods += fmt.Sprintf("mod%d = std::make_unique<p_top__eq%d>();\n", i, i)
	}

	for i := 0; i < len(g.InputPortVars); i++ {
		v := g.InputPortVars[i]
		width := 1
		if v.hasRange {
			width = v.Range.GetWidth()
		}
		for j := 0; j < equalNumber; j++ {
			inStr += fmt.Sprintf("mod%d->p_in%d  = cxxrtl::value<%d>(values[%d]);\n", j, i, width, valueIdx)
			initStr += fmt.Sprintf("mod%d->p_in%d  = cxxrtl::value<%d>(0u);\n", j, i, width)
		}
		valueIdx++
	}

	for i := 0; i < len(g.ClockVars); i++ {
		v := g.ClockVars[i]
		width := 1
		if v.hasRange {
			width = v.Range.GetWidth()
		}
		for j := 0; j < equalNumber; j++ {
			inStr += fmt.Sprintf("mod%d->p_clock__%d = cxxrtl::value<%d>(values[%d]);\n", j, i, width, valueIdx)
			initStr += fmt.Sprintf("mod%d->p_clock__%d = cxxrtl::value<%d>(0u);\n", j, i, width)
		}
		valueIdx++
	}

	compareStr := ""
	if len(g.OutputVars) > 0 {
		out := g.OutputVars[0]
		width := 1
		if out.hasRange {
			width = out.Range.GetWidth()
		}
		if out.isSigned {
			for i := 1; i < equalNumber; i++ {
				compareStr += fmt.Sprintf("if (truncate_signed(mod0->p_out0) != truncate_signed(mod%d->p_out0)) {\n", i)
				compareStr += "    std::cerr << \"Mismatch at line \" << line_num << std::endl;\n"
				compareStr += "    return 1;\n}\n"
			}
		} else {
			mask := genAllOnesHex(width)
			for i := 1; i < equalNumber; i++ {
				compareStr += fmt.Sprintf("if ((mod0->p_out0.get<uint32_t>() & %s) != (mod%d->p_out0.get<uint32_t>() & %s)) {\n",
					mask, i, mask)
				compareStr += "    std::cerr << \"NO\" << std::endl;\n"
				compareStr += "    return 1;\n}\n"
			}
		}
	}

	outStr := ""
	for i := 0; i < len(g.OutputVars); i++ {
		v := g.OutputVars[i]
		width := 1
		if v.hasRange {
			width = v.Range.GetWidth()
		}
		bitWidth := genAllOnesHex(width)
		if v.isSigned {
			if i != len(g.OutputVars)-1 {
				outStr += fmt.Sprintf("<< \"%s:\" << truncate_signed(mod0->p_out%d) << \" \"", v.Name, i)
			} else {
				outStr += fmt.Sprintf("<< \"%s:\" << truncate_signed(mod0->p_out%d) << std::endl;", v.Name, i)
			}
		} else {
			if i != len(g.OutputVars)-1 {
				outStr += fmt.Sprintf("<< \"%s:\" << (mod0->p_out%d.get<uint32_t>() & %s) << \" \"", v.Name, i, bitWidth)
			} else {
				outStr += fmt.Sprintf("<< \"%s:\" << (mod0->p_out%d.get<uint32_t>() & %s) << std::endl;", v.Name, i, bitWidth)
			}
		}
	}

	includeStr := ""
	for i := 0; i < equalNumber; i++ {
		includeStr += fmt.Sprintf("#include \"test%d.cpp\"\n", i)
	}

	stepStr := ""
	for i := 0; i < equalNumber; i++ {
		stepStr += fmt.Sprintf("mod%d->step();\n", i)
	}

	tbStr := fmt.Sprintf(`%s
#include <fstream>
#include <iostream>
#include <sstream>
#include <vector>
#include <string>

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
%s

%s

	std::ofstream ofs("output.txt");
	std::ifstream infile("input.txt");
	if (!infile) {
		std::cerr << "Cannot open input.txt" << std::endl;
		return 1;
	}

	std::string line;
	int line_num = 0;

%s

%s

%s // warmup

	while (std::getline(infile, line)) {
		line_num++;
		auto values = parse_line(line);

%s

%s

%s

%s
		if (!ofs) return 1;
		ofs %s
	}

	std::cout << "All outputs matched." << std::endl;
	return 0;
}
`, includeStr, modDecl, initMods, initStr, stepStr, stepStr, inStr, stepStr, stepStr, compareStr, outStr)

	return tbStr
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
	std::unique_ptr<p_%s> mod = std::make_unique<p_%s>();
	std::ofstream ofs("output.txt");
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

		%s

		mod->step();
		mod->step();
		mod->step();


		if (!ofs) return 1;
		ofs %s
	}

	return 0;
}

`, g.Name, g.Name, initStr, inStr, outStr)
	return tbStr
}

func (g *ExpressionGenerator) GenerateCXXRTLEquivalenceCheck(equalNumber int) string {
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
