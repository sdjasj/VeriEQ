package CodeGenerator

import (
	"fmt"
	"math/rand"
)

const undefinedInputProbability = 0.2

func pickUndefinedInputs(vars []*Variable, enable bool) map[*Variable]struct{} {
	undefined := make(map[*Variable]struct{})
	if !enable || len(vars) == 0 {
		return undefined
	}
	for _, v := range vars {
		if rand.Float64() < undefinedInputProbability {
			undefined[v] = struct{}{}
		}
	}
	if len(undefined) == 0 && undefinedInputProbability > 0 {
		undefined[vars[rand.Intn(len(vars))]] = struct{}{}
	}
	return undefined
}

func dummyInputName(v *Variable) string {
	return "dummy_" + v.Name
}

func xLiteralForVar(v *Variable) string {
	width := 1
	if v.hasRange && v.Range != nil {
		width = v.Range.GetWidth()
	}
	if width <= 1 {
		return "1'bx"
	}
	return fmt.Sprintf("{%d{1'bx}}", width)
}

func (g *ExpressionGenerator) GenerateTb() string {
	tbStr := fmt.Sprintf("`timescale 1ns/1ps\n\nmodule tb_dut_module;\n\n    parameter NUM_VECTORS = %d;  // 你想读取的行数\n\n",
		g.TestBenchTestTime)
	undefinedInputs := pickUndefinedInputs(g.InputPortVars, g.EnableXInputs)
	xAssignInit := ""
	xAssignLoop := ""
	for _, v := range g.InputPortVars {
		if _, ok := undefinedInputs[v]; ok {
			lit := xLiteralForVar(v)
			xAssignInit += fmt.Sprintf("        %s = %s; ", v.Name, lit)
			xAssignLoop += fmt.Sprintf("            %s = %s;\n", v.Name, lit)
		}
	}
	if xAssignInit != "" {
		xAssignInit += "\n"
	}
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
	for _, v := range g.InputPortVars {
		if _, ok := undefinedInputs[v]; ok {
			tbStr += fmt.Sprintf("    integer %s;\n", dummyInputName(v))
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
		if _, ok := undefinedInputs[v]; ok {
			continue
		}
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
		name := v.Name
		if _, ok := undefinedInputs[v]; ok {
			name = dummyInputName(v)
		}
		if i != len(g.InputPortVars)-1 {
			scanStr += fmt.Sprintf("%s ,", name)
		} else {
			scanStr += fmt.Sprintf("%s ", name)
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


%s
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
		initInput, initRegStr, xAssignInit, scanStr, scanClockStr, len(g.InputVars),
		xAssignLoop, outfmtStr, outVarStr)
	return tbStr
}

func (g *ExpressionGenerator) GenerateEquivalenceCheckTb(equalNumber int) string {
	tbStr := fmt.Sprintf("`timescale 1ns/1ps\n\nmodule tb_equiv_check;\n\nparameter NUM_VECTORS = %d;\n\n", g.TestBenchTestTime)

	undefinedInputs := pickUndefinedInputs(g.InputPortVars, g.EnableXInputs)
	xAssignInit := ""
	xAssignLoop := ""
	for _, v := range g.InputPortVars {
		if _, ok := undefinedInputs[v]; ok {
			lit := xLiteralForVar(v)
			xAssignInit += fmt.Sprintf("    %s = %s;\n", v.Name, lit)
			xAssignLoop += fmt.Sprintf("        %s = %s;\n", v.Name, lit)
		}
	}
	for _, v := range g.InputVars {
		signed := ""
		if v.isSigned {
			signed = "signed "
		}
		if v.hasRange {
			tbStr += fmt.Sprintf("reg %s [%d:%d] %s;\n", signed, v.Range.r, v.Range.l, v.Name)
		} else {
			tbStr += fmt.Sprintf("reg %s%s;\n", signed, v.Name)
		}
	}

	for i := 0; i < equalNumber; i++ {
		for _, v := range g.OutputVars {
			signed := ""
			if v.isSigned {
				signed = "signed "
			}
			name := fmt.Sprintf("%s_eq%d", v.Name, i)
			if v.hasRange {
				tbStr += fmt.Sprintf("wire %s [%d:%d] %s;\n", signed, v.Range.r, v.Range.l, name)
			} else {
				tbStr += fmt.Sprintf("wire %s%s;\n", signed, name)
			}
		}
	}

	for _, v := range g.InputPortVars {
		if _, ok := undefinedInputs[v]; ok {
			tbStr += fmt.Sprintf("integer %s;\n", dummyInputName(v))
		}
	}

	for i := 0; i < equalNumber; i++ {
		instName := fmt.Sprintf("uut_eq%d", i)
		moduleName := fmt.Sprintf("%s_eq%d", g.Name, i)
		tbStr += fmt.Sprintf("\n%s %s (\n", moduleName, instName)
		for _, v := range g.InputVars {
			tbStr += fmt.Sprintf("    .%s(%s),\n", v.Name, v.Name)
		}
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

	totalVars := len(g.InputVars)
	signalVars := g.InputPortVars
	clockVars := g.ClockVars
	signalCount := len(signalVars)
	clockCount := len(clockVars)

	buildScan := func(vars []*Variable, endWithNewline bool, dummy map[*Variable]struct{}) (string, string) {
		if len(vars) == 0 {
			return "", ""
		}
		format := `"`
		names := ""
		for i, v := range vars {
			name := v.Name
			if _, ok := dummy[v]; ok {
				name = dummyInputName(v)
			}
			if i == len(vars)-1 {
				if endWithNewline {
					format += "%d\\n"
				} else {
					format += "%d "
				}
			} else {
				format += "%d "
			}
			names += name
			if i != len(vars)-1 {
				names += ", "
			}
		}
		format += `", `
		return format, names
	}

	signalFormat, signalNames := buildScan(signalVars, clockCount == 0, undefinedInputs)
	clockFormat, clockNames := buildScan(clockVars, true, nil)

	signalScanStmt := "        status = 0;\n"
	if signalCount > 0 {
		signalScanStmt = fmt.Sprintf("        status = $fscanf(fin, %s%s);\n", signalFormat, signalNames)
	}
	clockScanStmt := ""
	if clockCount > 0 {
		clockScanStmt = fmt.Sprintf("        status = status + $fscanf(fin, %s%s);\n", clockFormat, clockNames)
	}

	tbStr += fmt.Sprintf(`
integer i, fin, fout, status;
initial begin
    fin = $fopen("%s", "r");
    if (fin == 0) begin
        $display("Cannot open input.txt");
        $finish;
    end
    fout = $fopen("output.txt", "w");
    if (fout == 0) begin
        $display("Cannot open output.txt");
        $finish;
    end

%s
    for (i = 0; i < NUM_VECTORS; i = i + 1) begin
%s        #20;
%s        if (status < %d) begin
            $display("WARNING: input.txt format error at line %%0d", i);
            $finish;
        end
%s
        #20;
`, g.TestBenchInputFileName, xAssignInit, signalScanStmt, clockScanStmt, totalVars, xAssignLoop)

	tbStr += `        if (
`

	for i := 1; i < equalNumber; i++ {
		for j, v := range g.OutputVars {
			tbStr += fmt.Sprintf("            %s_eq0 !== %s_eq%d", v.Name, v.Name, i)
			if !(i == equalNumber-1 && j == len(g.OutputVars)-1) {
				tbStr += " ||\n"
			} else {
				tbStr += "\n"
			}
		}
	}

	tbStr += `        ) begin
            $fwrite(fout, "NO\n");
        end else begin
            $fwrite(fout, "YES\n");
        end
`

	tbStr += `
    end

    $fclose(fin);
    $fclose(fout);
    $display("Test completed.");
    $finish;
end

endmodule
`
	return tbStr
}
