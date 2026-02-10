package main

import (
	"VeriEQ/CodeGenerator"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

func (f *Fuzzer) TestYosysOptUsingVerilator() {
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
	OptFileName := filepath.Join(realSubDir, "opt.v")
	tbFileName := filepath.Join(realSubDir, f.TestBenchName)

	if err := os.WriteFile(tmpFileName, []byte(generator.GenerateLoopFreeModule()), 0644); err != nil {
		fmt.Println("写 test.v 出错:", err)
		return
	}

	tbData := generator.GenerateTb()
	if err := os.WriteFile(tbFileName, []byte(tbData), 0644); err != nil {
		fmt.Println("写 tb.v 出错:", err)
		return
	}

	inputData := generator.GenerateInputFile()

	var stderrBuffer bytes.Buffer

	yosysOptFile := filepath.Join(realSubDir, "yosys_opt.log")
	logFile, err := os.Create(yosysOptFile)
	if err != nil {
		return
	}
	cmd := exec.Command(toolConfig.YosysPath, "-p", "read_verilog test.v; opt; proc; write_verilog opt.v")
	cmd.Dir = realSubDir
	cmd.Stdout = logFile
	cmd.Stderr = &stderrBuffer
	if err := cmd.Run(); err != nil {
		handleFailure(f.CrashDir, realSubDir, yosysOptFile, stderrBuffer.String()+err.Error())
		return
	}

	optFileContent, err := os.ReadFile(OptFileName)
	if err != nil {
		return
	}
	newContent := []byte("`timescale 1ns/1ps\n")
	newContent = append(newContent, optFileContent...)
	if err := os.WriteFile(OptFileName, newContent, 0644); err != nil {
		return
	}

	//printDirTree(realSubDir)
	//_ = os.Remove(generator.TestBenchOutputFileName)
	verilatorPath := toolConfig.VerilatorPath
	verilatorOutputDir := filepath.Join(realSubDir, "obj_dir")
	verilatorLogFileName := filepath.Join(realSubDir, GetRandomFileName("verilator", ".log", ""))

	args := []string{
		"--binary",
		"-Wno-lint",
		"--timing",
		tmpFileName,
		tbFileName,
	}
	cmd = exec.Command(verilatorPath, args...)
	cmd.Dir = realSubDir

	logFile, err = os.Create(verilatorLogFileName)
	if err != nil {
		fmt.Println("Error creating log file:", err)
		return
	}

	stderrBuffer.Reset()
	cmd.Stdout = logFile
	cmd.Stderr = &stderrBuffer

	if err := cmd.Run(); err != nil {
		handleFailure(f.CrashDir, realSubDir, verilatorLogFileName, stderrBuffer.String())
		return
	}

	testbenchInputPath := filepath.Join(verilatorOutputDir, generator.TestBenchInputFileName)
	if err := os.WriteFile(testbenchInputPath, []byte(inputData), 0644); err != nil {
		fmt.Println("写 testbench 输入出错:", err)
		return
	}
	cmd = exec.Command("./Vtest")
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
		handleFailure(f.CrashDir, realSubDir, verilatorLogFileName, err.Error())
	}

	logFile.Close()

	verilatorOut := filepath.Join(realSubDir, "verilator_output.txt")
	_ = os.WriteFile(verilatorOut, verilatorData, 0o644)

	//opt

	verilatorOutputDir = filepath.Join(realSubDir, "obj_dir_opt")
	verilatorLogFileName = filepath.Join(realSubDir, GetRandomFileName("verilator_opt", ".log", ""))

	args = []string{
		"--binary",
		"-Wno-lint",
		"--timing",
		"-Mdir",
		verilatorOutputDir,
		OptFileName,
		tbFileName,
	}
	cmd = exec.Command(verilatorPath, args...)
	cmd.Dir = realSubDir

	logFile, err = os.Create(verilatorLogFileName)
	if err != nil {
		fmt.Println("Error creating log file:", err)
		return
	}

	stderrBuffer.Reset()
	cmd.Stdout = logFile
	cmd.Stderr = &stderrBuffer

	if err := cmd.Run(); err != nil {
		handleFailure(f.CrashDir, realSubDir, verilatorLogFileName, stderrBuffer.String())
		return
	}

	testbenchInputPath = filepath.Join(verilatorOutputDir, generator.TestBenchInputFileName)
	if err := os.WriteFile(testbenchInputPath, []byte(inputData), 0644); err != nil {
		fmt.Println("写 testbench 输入出错:", err)
		return
	}
	cmd = exec.Command("./Vopt")
	cmd.Dir = verilatorOutputDir
	cmd.Stdout = logFile
	cmd.Stderr = &stderrBuffer
	stderrBuffer.Reset()
	if err := cmd.Run(); err != nil {
		handleFailure(f.CrashDir, realSubDir, verilatorLogFileName, stderrBuffer.String())
		return
	}

	verilatorOptData, err := os.ReadFile(filepath.Join(verilatorOutputDir, generator.TestBenchOutputFileName))
	if err != nil {
		handleFailure(f.CrashDir, realSubDir, verilatorLogFileName, err.Error())
	}

	logFile.Close()

	verilatorOut = filepath.Join(realSubDir, "verilator_opt_output.txt")
	_ = os.WriteFile(verilatorOut, verilatorOptData, 0o644)

	equalOpt := bytes.Equal(verilatorData, verilatorOptData)

	if equalOpt {
		if err := os.RemoveAll(realSubDir); err != nil {
			fmt.Printf("删除临时测试目录失败: %v\n", err)
		}
		return
	}
	PrettyBug("yosys", "bug detected", "saved: diff.txt")

	diffContent := "\n==== NoOpt vs Opt Diff ====\n" + diffLines(verilatorData, verilatorOptData)
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

func (f *Fuzzer) TestYosysOpt() {
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
	OptFileName := filepath.Join(realSubDir, "opt.v")
	tbFileName := filepath.Join(realSubDir, f.TestBenchName)

	if err := os.WriteFile(tmpFileName, []byte(generator.GenerateLoopFreeModule()), 0644); err != nil {
		return
	}

	tbData := generator.GenerateTb()
	if err := os.WriteFile(tbFileName, []byte(tbData), 0644); err != nil {
		return
	}

	var stderrBuffer bytes.Buffer
	inputData := generator.GenerateInputFile()
	yosysOptFile := filepath.Join(realSubDir, "yosys_opt.log")
	logFile, err := os.Create(yosysOptFile)
	if err != nil {
		return
	}
	cmd := exec.Command(toolConfig.YosysPath, "-p", "read_verilog test.v; opt; proc; write_verilog opt.v")
	cmd.Dir = realSubDir
	cmd.Stdout = logFile
	cmd.Stderr = &stderrBuffer
	if err := cmd.Run(); err != nil {
		handleFailure(f.CrashDir, realSubDir, yosysOptFile, stderrBuffer.String()+err.Error())
		return
	}

	//printDirTree(realSubDir)
	//_ = os.Remove(generator.TestBenchOutputFileName)
	iverilogDir := filepath.Join(realSubDir, "iverilog")
	if err := os.MkdirAll(iverilogDir, 0755); err != nil {
		return
	}

	aoutPath := filepath.Join(iverilogDir, "a.out")
	args := []string{tmpFileName, tbFileName, "-o", aoutPath}
	cmd = exec.Command(toolConfig.IverilogPath, args...)
	cmd.Env = append(os.Environ(), "ASAN_OPTIONS=detect_leaks=0")
	cmd.Dir = realSubDir

	iverilogLogFileName := filepath.Join(realSubDir, GetRandomFileName("iverilog", ".log", ""))
	logFile, err = os.Create(iverilogLogFileName)
	if err != nil {
		return
	}

	cmd.Stdout = logFile
	cmd.Stderr = &stderrBuffer
	stderrBuffer.Reset()

	if err := cmd.Run(); err != nil {
		handleFailure(f.CrashDir, realSubDir, iverilogLogFileName, stderrBuffer.String()+err.Error())
		return
	}

	testbenchInputPath := filepath.Join(iverilogDir, generator.TestBenchInputFileName)
	if err := os.WriteFile(testbenchInputPath, []byte(inputData), 0644); err != nil {
		return
	}
	cmd = exec.Command("./a.out")
	cmd.Env = append(os.Environ(), "ASAN_OPTIONS=detect_leaks=0")
	cmd.Dir = iverilogDir
	cmd.Stdout = logFile
	cmd.Stderr = &stderrBuffer
	stderrBuffer.Reset()
	if err := cmd.Run(); err != nil {
		handleFailure(f.CrashDir, realSubDir, iverilogLogFileName, stderrBuffer.String()+err.Error())
		return
	}

	iverilogData, err := os.ReadFile(filepath.Join(iverilogDir, generator.TestBenchOutputFileName))
	if err != nil {
		handleFailure(f.CrashDir, realSubDir, iverilogLogFileName, err.Error())
	}

	logFile.Close()

	iverilogOut := filepath.Join(realSubDir, "iverilog_output_noOpt.txt")
	_ = os.WriteFile(iverilogOut, iverilogData, 0o644)

	//opt

	iverilogDir = filepath.Join(realSubDir, "iverilog_opt")
	if err := os.MkdirAll(iverilogDir, 0755); err != nil {
		return
	}

	aoutPath = filepath.Join(iverilogDir, "a.out")
	args = []string{OptFileName, tbFileName, "-o", aoutPath}
	cmd = exec.Command(toolConfig.IverilogPath, args...)
	cmd.Env = append(os.Environ(), "ASAN_OPTIONS=detect_leaks=0")
	cmd.Dir = realSubDir

	iverilogLogFileName = filepath.Join(realSubDir, GetRandomFileName("iverilog_opt_", ".log", ""))
	logFile, err = os.Create(iverilogLogFileName)
	if err != nil {
		return
	}

	cmd.Stdout = logFile
	cmd.Stderr = &stderrBuffer
	stderrBuffer.Reset()

	if err := cmd.Run(); err != nil {
		handleFailure(f.CrashDir, realSubDir, iverilogLogFileName, stderrBuffer.String()+err.Error())
		return
	}

	testbenchInputPath = filepath.Join(iverilogDir, generator.TestBenchInputFileName)
	if err := os.WriteFile(testbenchInputPath, []byte(inputData), 0644); err != nil {
		return
	}
	cmd = exec.Command("./a.out")
	cmd.Env = append(os.Environ(), "ASAN_OPTIONS=detect_leaks=0")
	cmd.Dir = iverilogDir
	cmd.Stdout = logFile
	cmd.Stderr = &stderrBuffer
	stderrBuffer.Reset()
	if err := cmd.Run(); err != nil {
		handleFailure(f.CrashDir, realSubDir, iverilogLogFileName, stderrBuffer.String()+err.Error())
		return
	}

	iverilogOptData, err := os.ReadFile(filepath.Join(iverilogDir, generator.TestBenchOutputFileName))
	if err != nil {
		handleFailure(f.CrashDir, realSubDir, iverilogLogFileName, err.Error())
	}

	logFile.Close()

	iverilogOut = filepath.Join(realSubDir, "iverilog_output_Opt.txt")
	_ = os.WriteFile(iverilogOut, iverilogOptData, 0o644)

	equalOpt := bytes.Equal(iverilogOptData, iverilogData)

	if equalOpt {
		if err := os.RemoveAll(realSubDir); err != nil {
			fmt.Printf("%v\n", err)
		}
		return
	}
	PrettyBug("yosys", "bug detected", "saved: diff.txt")

	diffContent := "\n==== NoOpt vs Opt Diff ====\n" + diffLines(iverilogData, iverilogOptData)
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

func (f *Fuzzer) TestYosysOptUsingVerilatorWithManyOptions() {
	generator := CodeGenerator.NewExpressionGenerator()
	curMillis := time.Now().UnixMilli()

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
	OptFileName := filepath.Join(realSubDir, "opt.v")
	tbFileName := filepath.Join(realSubDir, f.TestBenchName)

	if err := os.WriteFile(tmpFileName, []byte(generator.GenerateLoopFreeModule()), 0644); err != nil {
		return
	}

	tbData := generator.GenerateTb()
	if err := os.WriteFile(tbFileName, []byte(tbData), 0644); err != nil {
		return
	}

	inputData := generator.GenerateInputFile()

	err := f.RunYosysOpt(realSubDir, OptFileName)
	if err != nil {
		return
	}
	optionLength := len(f.VerilatorOptions)
	verilatorOutput := make([][]byte, optionLength)
	verilatorTaskState := make([]bool, optionLength)
	var wg sync.WaitGroup
	wg.Add(optionLength)
	for i := 0; i < optionLength; i++ {
		i := i
		go func() {
			defer wg.Done()
			verilatorData, err := RunVerilator(f.VerilatorOptions[i], inputData,
				generator, realSubDir, OptFileName, tbFileName)
			if err != nil {
				fmt.Println("error of ", f.VerilatorOptions[i])
				return
			}
			//fmt.Println(f.VerilatorOptions[i])
			verilatorTaskState[i] = true
			verilatorOutput[i] = verilatorData
		}()
		//verilatorData, err := RunVerilator(f.VerilatorOptions[i], inputData,
		//	generator, realSubDir, tmpFileName, tbFileName)
		//if err != nil {
		//	fmt.Println("error of ", f.VerilatorOptions[i])
		//	return
		//}
		//verilatorTaskState[i] = true
		//verilatorOutput[i] = verilatorData
	}
	wg.Wait()
	for i := 0; i < optionLength; i++ {
		if !verilatorTaskState[i] {
			fmt.Println("verilator compile or test fail in " + f.VerilatorOptions[i])
			handleFailure(f.CrashDir, realSubDir, "", "")
			return
		}
	}

	inconsistentFound := false
	for i := 0; i < optionLength; i++ {
		for j := i + 1; j < optionLength; j++ {
			if !bytes.Equal(verilatorOutput[i], verilatorOutput[j]) {
				inconsistentFound = true

				diffContent := fmt.Sprintf(
					"==== %s vs %s ====\n%s\n",
					f.VerilatorOptions[i], f.VerilatorOptions[j],
					diffLines(verilatorOutput[i], verilatorOutput[j]),
				)

				diffFile := filepath.Join(realSubDir, fmt.Sprintf("diff_%d_vs_%d.txt", i, j))
				_ = os.WriteFile(diffFile, []byte(diffContent), 0o644)
			}
		}
	}

	if inconsistentFound {
		PrettyBug("yosys", "bug detected")

		handleFailure(f.CrashDir, realSubDir, "", "")
	} else {
		if err := os.RemoveAll(realSubDir); err != nil {
			fmt.Printf("%v\n", err)
		}
	}

}

func (f *Fuzzer) TestEqualModulesYosysOpt(equalNumber int) {
	generator := CodeGenerator.NewExpressionGenerator()
	if equalNumber == 0 {
		equalNumber = 10
	}
	curMillis := time.Now().UnixMilli()
	curTimeStr := strconv.FormatInt(curMillis, 10)
	subDir := strconv.FormatInt(curMillis%1000, 10)
	tmpSubDir := filepath.Join(f.TmpDir, subDir)

	if err := os.MkdirAll(tmpSubDir, 0755); err != nil {
		fmt.Println(err)
		return
	}
	realSubDir := filepath.Join(tmpSubDir, GetRandomFileName("tmp_yosysOpt", "", ""))
	if err := os.MkdirAll(realSubDir, 0755); err != nil {
		fmt.Println(err)
		return
	}

	tmpFileName := filepath.Join(realSubDir, f.TestFileName)
	tbFileName := filepath.Join(realSubDir, f.TestBenchName)
	tbDiffFileName := filepath.Join(realSubDir, "tb_diff.v")

	moduleData := generator.GenerateLoopFreeEquivalentModules(equalNumber)
	tbData := generator.GenerateEquivalenceCheckTb(equalNumber)

	if err := os.WriteFile(tmpFileName, []byte(moduleData), 0644); err != nil {
		return
	}
	if err := os.WriteFile(tbFileName, []byte(tbData), 0644); err != nil {
		return
	}

	tbFileForSim := tbFileName
	if f.EnableDiffSim {
		tbDiffData := generateEq0Tb(generator)
		if err := os.WriteFile(tbDiffFileName, []byte(tbDiffData), 0644); err != nil {
			return
		}
		tbFileForSim = tbDiffFileName
	}

	inputData := generator.GenerateInputFile()

	preData, optData, err := f.RunYosysOptAndSim(inputData,
		generator, realSubDir, tmpFileName, tbFileForSim)
	if err != nil {
		return
	}
	if f.EnableDiffSim {
		if bytes.Equal(preData, optData) {
			if err := os.RemoveAll(realSubDir); err != nil {
				fmt.Printf("%v\n", err)
			}
			PrettyOK("yosys", "finish")
			return
		}
		PrettyBug("yosys", "bug detected")
		diffContent := "\n==== NoOpt vs Opt Diff ====\n" + diffLines(preData, optData)
		diffFile := filepath.Join(realSubDir, "diff.txt")
		_ = os.WriteFile(diffFile, []byte(diffContent), 0o644)
	} else if !strings.Contains(string(preData), "NO") && !strings.Contains(string(optData), "NO") {
		if err := os.RemoveAll(realSubDir); err != nil {
			fmt.Printf("%v\n", err)
		}
		PrettyOK("yosys", "finish")
		return
	}
	if !f.EnableDiffSim {
		PrettyBug("yosys", "bug detected")
	}
	uniqueCrashDir := filepath.Join(
		f.CrashDir,
		"bug_"+curTimeStr+"_"+GetRandomFileName("", "", ""),
	)
	_ = copyCrashArtifacts(realSubDir, uniqueCrashDir)
}
