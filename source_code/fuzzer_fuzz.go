package main

import (
	"VeriEQ/CodeGenerator"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

func (f *Fuzzer) Fuzz() {
	generator := CodeGenerator.NewExpressionGenerator()
	curMillis := time.Now().UnixMilli()
	curTimeStr := strconv.FormatInt(curMillis, 10)

	subDir := strconv.FormatInt(curMillis%1000, 10)
	tmpSubDir := filepath.Join(f.TmpDir, subDir)

	if err := os.MkdirAll(tmpSubDir, 0755); err != nil {
		fmt.Println(err)
		return
	}
	realSubDir := filepath.Join(tmpSubDir, GetRandomFileName("tmp_", "", ""))
	if err := os.MkdirAll(realSubDir, 0755); err != nil {
		fmt.Println(err)
		return
	}

	tmpFileName := filepath.Join(realSubDir, f.TestFileName)
	tbFileName := filepath.Join(realSubDir, f.TestBenchName)

	if err := os.WriteFile(tmpFileName, []byte(generator.GenerateLoopFreeModule()), 0644); err != nil {
		fmt.Println(err)
		return
	}

	tbData := generator.GenerateTb()
	if err := os.WriteFile(tbFileName, []byte(tbData), 0644); err != nil {
		fmt.Println(err)
		return
	}

	inputData := generator.GenerateInputFile()

	verilatorPath := toolConfig.VerilatorPath
	verilatorOutputDir := filepath.Join(realSubDir, "obj_dir")
	verilatorLogFileName := filepath.Join(realSubDir, GetRandomFileName("verilator", ".log", ""))

	topModule := "tb_dut_module"
	args := []string{
		"--binary",
		"-Wno-lint",
		"--timing",
		"--top-module",
		topModule,
		tmpFileName,
		tbFileName,
	}
	cmd := exec.Command(verilatorPath, args...)
	cmd.Dir = realSubDir

	logFile, err := os.Create(verilatorLogFileName)
	if err != nil {
		fmt.Println("Error creating log file:", err)
		return
	}

	var stderrBuffer bytes.Buffer
	cmd.Stdout = logFile
	cmd.Stderr = &stderrBuffer

	if err := cmd.Run(); err != nil {
		handleFailure(f.CrashDir, realSubDir, verilatorLogFileName, stderrBuffer.String())
		return
	}

	testbenchInputPath := filepath.Join(verilatorOutputDir, generator.TestBenchInputFileName)
	if err := os.WriteFile(testbenchInputPath, []byte(inputData), 0644); err != nil {
		fmt.Println(err)
		return
	}
	binaryName := "./V" + topModule
	cmd = exec.Command(binaryName)
	cmd.Dir = verilatorOutputDir
	cmd.Stdout = logFile
	cmd.Stderr = &stderrBuffer
	stderrBuffer.Reset()
	if err := cmd.Run(); err != nil {
		handleFailure(f.CrashDir, realSubDir, verilatorLogFileName, stderrBuffer.String())
		return
	}

	verilatorData, err := os.ReadFile(filepath.Join(verilatorOutputDir, generator.TestBenchOutputFileName))
	if err != nil {
		fmt.Printf(" %v\n", err)
		handleFailure(f.CrashDir, realSubDir, verilatorLogFileName, err.Error())
	}

	logFile.Close()

	verilatorOut := filepath.Join(realSubDir, "verilator_output.txt")
	_ = os.WriteFile(verilatorOut, verilatorData, 0o644)
	//verilatorData, err := f.RunVerilator(inputData, generator, realSubDir, tmpSubDir, tbFileName)
	//if err != nil {
	//	return
	//}
	//printDirTree(realSubDir)
	//_ = os.Remove(generator.TestBenchOutputFileName)
	iverilogDir := filepath.Join(realSubDir, "iverilog")
	if err := os.MkdirAll(iverilogDir, 0755); err != nil {
		fmt.Println(err)
		return
	}

	aoutPath := filepath.Join(iverilogDir, "a.out")
	args = []string{tmpFileName, tbFileName, "-o", aoutPath}
	cmd = exec.Command(toolConfig.IverilogPath, args...)
	cmd.Env = append(os.Environ(), "ASAN_OPTIONS=detect_leaks=0")
	cmd.Dir = realSubDir

	iverilogLogFileName := filepath.Join(realSubDir, GetRandomFileName("iverilog", ".log", ""))
	logFile, err = os.Create(iverilogLogFileName)
	if err != nil {
		fmt.Println(err)
		return
	}

	cmd.Stdout = logFile
	cmd.Stderr = &stderrBuffer
	stderrBuffer.Reset()

	if err := cmd.Run(); err != nil {
		handleFailure(f.CrashDir, iverilogLogFileName, stderrBuffer.String(), err.Error())
		return
	}

	testbenchInputPath = filepath.Join(iverilogDir, generator.TestBenchInputFileName)
	if err := os.WriteFile(testbenchInputPath, []byte(inputData), 0644); err != nil {
		fmt.Println(err)
		return
	}
	cmd = exec.Command("./a.out")
	cmd.Env = append(os.Environ(), "ASAN_OPTIONS=detect_leaks=0")
	cmd.Dir = iverilogDir
	cmd.Stdout = logFile
	cmd.Stderr = &stderrBuffer
	stderrBuffer.Reset()
	if err := cmd.Run(); err != nil {
		handleFailure(f.CrashDir, iverilogLogFileName, stderrBuffer.String(), err.Error())
		return
	}

	iverilogData, err := os.ReadFile(filepath.Join(iverilogDir, generator.TestBenchOutputFileName))
	if err != nil {
		fmt.Println(err)
		handleFailure(f.CrashDir, realSubDir, iverilogLogFileName, err.Error())
	}

	logFile.Close()

	iverilogOut := filepath.Join(realSubDir, "iverilog_output.txt")
	_ = os.WriteFile(iverilogOut, iverilogData, 0o644)

	//printDirTree(realSubDir)
	//_ = os.Remove(generator.TestBenchOutputFileName)
	cxxrtlDir := filepath.Join(realSubDir, "cxxrtl")
	if err := os.MkdirAll(cxxrtlDir, 0o755); err != nil {
		fmt.Println(err)
		return
	}
	CXXTestBenchData := generator.GenerateCXXRTLTestBench()
	CXXTestBenchFile := filepath.Join(cxxrtlDir, "main.cpp")
	if err := os.WriteFile(CXXTestBenchFile, []byte(CXXTestBenchData), 0644); err != nil {
		fmt.Println(err)
	}

	yosysCmd := exec.Command(
		toolConfig.YosysPath,
		"-p",
		fmt.Sprintf(
			"read_verilog %s; write_cxxrtl test.cpp",
			tmpFileName),
	)
	yosysCmd.Dir = cxxrtlDir

	cxxrtlLog := filepath.Join(realSubDir, GetRandomFileName("cxxrtl", ".log", ""))

	logFile, err = os.Create(cxxrtlLog)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer logFile.Close()

	var cxxrtlStderr bytes.Buffer
	yosysCmd.Stdout = logFile
	yosysCmd.Stderr = &cxxrtlStderr

	if err := yosysCmd.Run(); err != nil {
		handleFailure(f.CrashDir, realSubDir, cxxrtlLog, cxxrtlStderr.String())
		return
	}

	compileCmd := fmt.Sprintf("%s -w -g -O3 -std=c++14 -I $(%s --datdir)/include/backends/cxxrtl/runtime main.cpp -o cxxsim",
		toolConfig.ClangXXPath, toolConfig.YosysConfigPath)
	build := exec.Command("bash", "-c", compileCmd)
	build.Dir = cxxrtlDir
	build.Stdout = logFile
	build.Stderr = &cxxrtlStderr
	cxxrtlStderr.Reset()

	if err := build.Run(); err != nil {
		handleFailure(f.CrashDir, realSubDir, cxxrtlLog, cxxrtlStderr.String())
		return
	}

	tbInputPath := filepath.Join(cxxrtlDir, generator.TestBenchInputFileName)
	if err := os.WriteFile(tbInputPath, []byte(inputData), 0o644); err != nil {
		return
	}

	sim := exec.Command("./cxxsim")
	sim.Dir = cxxrtlDir
	sim.Stdout = logFile
	sim.Stderr = &cxxrtlStderr
	cxxrtlStderr.Reset()

	if err := sim.Run(); err != nil {
		handleFailure(f.CrashDir, realSubDir, cxxrtlLog, cxxrtlStderr.String())
		return
	}

	cxxrtlData, err := os.ReadFile(filepath.Join(cxxrtlDir, "output.txt"))
	if err != nil {
		handleFailure(f.CrashDir, realSubDir, cxxrtlLog, err.Error())
	}

	logFile.Close()
	//printDirTree(realSubDir)
	//_ = os.Remove(generator.TestBenchOutputFileName)

	cxxrtlOut := filepath.Join(realSubDir, "cxxrtl_output.txt")
	_ = os.WriteFile(cxxrtlOut, cxxrtlData, 0o644)

	equalVI := bytes.Equal(verilatorData, iverilogData)
	equalVC := bytes.Equal(verilatorData, cxxrtlData)
	equalIC := bytes.Equal(iverilogData, cxxrtlData)

	if equalVI && equalVC && equalIC {
		if err := os.RemoveAll(realSubDir); err != nil {
			fmt.Printf("%v\n", err)
		}
		return
	}
	diffStr := ""
	if !equalVI {
		diffStr += "verilator is not equal with iverilog\n"
	}
	if !equalVC {
		diffStr += "verilator is not equal with cxxrtl\n"
	}
	if !equalIC {
		diffStr += "iverilog is not equal with cxxrtl\n"
	}
	PrettyBug("fuzz", "bug detected")

	diffContent := diffStr + "\n==== Verilator vs CXXRTL Diff ====\n" + diffLines(verilatorData, cxxrtlData) +
		"\n==== Verilator vs Icarus Diff ====\n" +
		diffLines(verilatorData, iverilogData) +
		"\n==== Icarus vs CXXRTL Diff ====\n" +
		diffLines(iverilogData, cxxrtlData)
	diffFile := filepath.Join(realSubDir, "diff.txt")
	_ = os.WriteFile(diffFile, []byte(diffContent), 0o644)

	_ = os.WriteFile(
		filepath.Join(f.TmpDir, "panic.log"),
		[]byte("bug occur!!!!!!!!!"),
		0o644,
	)

	uniqueCrashDir := filepath.Join(
		f.CrashDir,
		"bug_"+curTimeStr+"_"+GetRandomFileName("", "", ""),
	)
	_ = copyCrashArtifacts(realSubDir, uniqueCrashDir)

}
