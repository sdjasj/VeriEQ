package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"goFuzzer/CodeGenerator"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Fuzzer struct {
	StartTime        int64
	LogDir           string
	TmpDir           string
	CrashDir         string
	TestBenchName    string
	TestFileName     string
	VerilatorOptions []string
}

func printDirTree(dir string) {
	cmd := exec.Command("tree", dir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("%v\n", err)
	}
}

func (f *Fuzzer) Init() {
	baseTime := strconv.FormatInt(f.StartTime, 10)
	curDir, _ := os.Getwd()
	logDir := curDir + "/" + LOGDIR + baseTime + "/"
	f.LogDir = logDir
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		err := os.MkdirAll(logDir, 0755)
		if err != nil {
			panic(err)
		}
	}

	tmpDir := curDir + "/" + TMPDIR + baseTime + "/"
	f.TmpDir = tmpDir
	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		err := os.MkdirAll(tmpDir, 0755)
		if err != nil {
			panic(err)
		}
	}

	crashDir := curDir + "/" + CRASHDIR + baseTime + "/"
	f.CrashDir = crashDir
	if _, err := os.Stat(crashDir); os.IsNotExist(err) {
		err := os.MkdirAll(crashDir, 0755)
		if err != nil {
			panic(err)
		}
	}
	f.TestBenchName = "tb.v"
	f.TestFileName = "test.v"
	f.VerilatorOptions = []string{
		"-fno-acyc-simp",
		"-fno-assemble",
		"-fno-case",
		"-fno-combine",
		"-fno-const",
		"-fno-const-bit-op-tree",
		"-fno-dedup",
		"-fno-dfg",
		"-fno-dfg-peephole",
		"-fno-dfg-pre-inline",
		"-fno-dfg-post-inline",
		"-fno-expand",
		"-fno-func-opt",
		"-fno-func-opt-balance-cat",
		"-fno-func-opt-split-cat",
		"-fno-gate",
		"-fno-inline",
		"-fno-life",
		"-fno-life-post",
		"-fno-localize",
		"-fno-merge-cond",
		"-fno-merge-cond-motion",
		"-fno-merge-const-pool",
		"-fno-reloop",
		"-fno-reorder",
		"-fno-split",
		"-fno-subst",
		"-fno-subst-const",
		"-fno-table",
	}
}

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

	verilatorPath := "/root/hardware-test/verilator/verilator/bin/verilator"
	verilatorOutputDir := filepath.Join(realSubDir, "obj_dir")
	verilatorLogFileName := filepath.Join(realSubDir, GetRandomFileName("verilator", ".log", ""))

	args := []string{
		"--binary",
		"-Wno-lint",
		"--timing",
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
	cmd = exec.Command("iverilog", args...)
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
		"yosys",
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

	compileCmd := "clang++ -w -g -O3 -std=c++14 " +
		"-I $(yosys-config --datdir)/include/backends/cxxrtl/runtime " +
		"main.cpp -o cxxsim"
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
	fmt.Println(diffStr + "bug occur!!!!!!!!!!!!!!!!!")

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
		"crash_"+curTimeStr+"_"+GetRandomFileName("", "", ""),
	)
	_ = copyDir(realSubDir, uniqueCrashDir)

}

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
	cmd := exec.Command("yosys", "-p", "read_verilog test.v; opt; proc; write_verilog opt.v")
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
	verilatorPath := "/root/hardware-test/verilator/verilator/bin/verilator"
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
	fmt.Println("bug occur!!!!!!!!!!!!!!!!!")

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
		"crash_"+curTimeStr+"_"+GetRandomFileName("", "", ""),
	)
	_ = copyDir(realSubDir, uniqueCrashDir)
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
	cmd := exec.Command("yosys", "-p", "read_verilog test.v; opt; proc; write_verilog opt.v")
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
	cmd = exec.Command("iverilog", args...)
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
	cmd = exec.Command("iverilog", args...)
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
	fmt.Println("bug occur!!!!!!!!!!!!!!!!!")

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
		"crash_"+curTimeStr+"_"+GetRandomFileName("", "", ""),
	)
	_ = copyDir(realSubDir, uniqueCrashDir)
}

func diffLines(verilogData, iverilogData []byte) string {
	verilogLines := strings.Split(string(verilogData), "\n")
	iverilogLines := strings.Split(string(iverilogData), "\n")

	var sb strings.Builder

	maxLines := len(verilogLines)
	if len(iverilogLines) > maxLines {
		maxLines = len(iverilogLines)
	}

	for i := 0; i < maxLines; i++ {
		var vLine, iLine string
		if i < len(verilogLines) {
			vLine = verilogLines[i]
		}
		if i < len(iverilogLines) {
			iLine = iverilogLines[i]
		}

		if vLine != iLine {
			sb.WriteString(fmt.Sprintf("Line %d:\n", i+1))
			sb.WriteString(fmt.Sprintf("  Verilator: %s\n", vLine))
			sb.WriteString(fmt.Sprintf("  Icarus   : %s\n", iLine))
		}
	}

	return sb.String()
}

func copyDir(src string, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		dstFile, err := os.Create(dstPath)
		if err != nil {
			return err
		}
		defer dstFile.Close()

		_, err = io.Copy(dstFile, srcFile)
		if err != nil {
			return err
		}

		return dstFile.Chmod(info.Mode())
	})
}

func processCrash(logFile string, stderr string) {
	if checkSanitizerErrorFromStderr(stderr) {
		return
	}
	appendToLogFile(logFile, stderr)
}

func handleFailure(crashDir, realSubDir, logFile string, stderr string) {
	if logFile != "" {
		processCrash(logFile, stderr)
	}
	curTimeStr := strconv.FormatInt(time.Now().UnixMilli(), 10)
	crashSubdir := filepath.Join(crashDir, "crash_"+curTimeStr+"_"+GetRandomFileName("", "", ""))
	if err := copyDir(realSubDir, crashSubdir); err != nil {
		fmt.Printf("%v\n", err)
	}

	// 删除测试目录
	if err := os.RemoveAll(realSubDir); err != nil {
		fmt.Printf("%v\n", err)
	}
}

// 将 ASan/UBSan 错误输出追加到日志文件末尾
func appendToLogFile(logFileName, stderrContent string) {
	file, err := os.OpenFile(logFileName, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer file.Close()

	// 追加 sanitizer 错误到日志文件
	_, err = file.WriteString("\n==== Sanitizer====\n" + stderrContent + "\n")
	if err != nil {
		return
	}
}

// copyFile 把 src 文件内容拷贝到 dst 文件
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()

	_, err = io.Copy(out, in)
	return err
}

func checkSanitizerErrorFromStderr(stderr string) bool {
	sanitizerKeywords := []string{
		"AddressSanitizer", "heap-buffer-overflow", "stack-buffer-overflow",
		"use-after-free", "global-buffer-overflow", "double-free",
		"invalid-pointer", "shadow memory",

		"UndefinedBehaviorSanitizer", "runtime error:", "signed integer overflow",
		"shift exponent", "out of bounds", "alignment assumption", "integer divide by zero",
		"invalid shift", "unreachable code", "type mismatch",
	}

	for _, keyword := range sanitizerKeywords {
		if strings.Contains(stderr, keyword) {
			return true
		}
	}
	return false
}

const logo = `
__     __        _       _____ ___  
\ \   / /__ _ __(_)     | ____/ _ \ 
 \ \ / / _ \ '__| |_____|  _|| | | |
  \ V /  __/ |  | |_____| |__| |_| |
   \_/ \___|_|  |_|     |_____\__\_\

`

func main() {
	//TestAllEquivalence()
	fmt.Println(logo)
	fuzzer := flag.String("fuzzer", "verilator", "Which fuzzer to run: iverilog | verilator | yosys | cxxrtl")
	threads := flag.Int("threads", 10, "Number of threads")
	count := flag.Int("count", 50, "Number of equivalent test cases")

	flag.Parse()

	RunSelectedFuzzer(*fuzzer, *count, *threads)
}

func RunSelectedFuzzer(name string, count, threads int) {
	switch name {
	case "iverilog":
		go EqualFuzzIverilog(threads, count)
	case "verilator":
		go EqualFuzzVerilator(threads, count)
	case "yosys":
		go EqualFuzzYosysOpt(threads, count)
	case "cxxrtl":
		go EqualFuzzCXXRTL(threads, count)
	default:
		fmt.Fprintf(os.Stderr, "Unknown fuzzer: %s\n", name)
		os.Exit(1)
	}
}

func TestSimpleCXXRTL() error {

	tmpDir, err := os.MkdirTemp("", "cxxrtl_simple_")
	fmt.Println(tmpDir)
	//defer os.RemoveAll(tmpDir)
	generator := CodeGenerator.NewExpressionGenerator()
	dut := generator.GenerateLoopFreeModule()
	tb := generator.GenerateCXXRTLTestBench()
	inputStr := generator.GenerateInputFile()

	dutFile := filepath.Join(tmpDir, "dut.v")
	tbFile := filepath.Join(tmpDir, "main.cpp")
	inputFile := filepath.Join(tmpDir, "input.txt")
	if err := os.WriteFile(dutFile, []byte(dut), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(tbFile, []byte(tb), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(inputFile, []byte(inputStr), 0644); err != nil {
		return err
	}

	yosysCmd := exec.Command(
		"yosys",
		"-p", fmt.Sprintf(
			"read_verilog %s; write_cxxrtl test.cpp",
			dutFile),
	)
	yosysCmd.Dir = tmpDir
	if out, err := yosysCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("Yosys 失败: %v\n%s", err, out)
	}

	buildCmd := `clang++ -w -g -O3 -std=c++14 ` +
		`-I $(yosys-config --datdir)/include/backends/cxxrtl/runtime ` +
		`main.cpp -o cxxsim`
	compile := exec.Command("bash", "-c", buildCmd)
	compile.Dir = tmpDir
	if out, err := compile.CombinedOutput(); err != nil {
		return fmt.Errorf("clang++ 失败: %v\n%s", err, out)
	}

	run := exec.Command("./cxxsim")
	run.Dir = tmpDir
	if out, err := run.CombinedOutput(); err != nil {
		return fmt.Errorf("CXXRTL 运行失败: %v\n%s", err, out)
	} else if len(out) != 0 {

	}

	gotBytes, err := os.ReadFile(filepath.Join(tmpDir, "output.txt"))
	if err != nil {
		return fmt.Errorf("读取 output.txt 失败: %v", err)
	}

	fmt.Println(gotBytes)
	return nil
}

func TestEqualCXXRTL() error {
	tmpDir, err := os.MkdirTemp("", "cxxrtl_equal_")
	fmt.Println(tmpDir)
	//defer os.RemoveAll(tmpDir)
	if err != nil {
		return fmt.Errorf("%v", err)
	}
	equalNum := 5
	generator := CodeGenerator.NewExpressionGenerator()
	dut := generator.GenerateEquivalentModulesWithOneTop(equalNum)
	tb := generator.GenerateCXXRTLMultiModuleTestBench(equalNum)
	inputStr := generator.GenerateInputFile()

	dutFile := filepath.Join(tmpDir, "dut.v")
	tbFile := filepath.Join(tmpDir, "main.cpp")
	inputFile := filepath.Join(tmpDir, "input.txt")
	if err := os.WriteFile(dutFile, []byte(dut), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(tbFile, []byte(tb), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(inputFile, []byte(inputStr), 0644); err != nil {
		return err
	}

	var wg sync.WaitGroup
	errCh := make(chan error, equalNum)

	for i := 0; i < equalNum; i++ {
		i := i
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			outputFile := fmt.Sprintf("test%d.cpp", i)
			outputPath := filepath.Join(tmpDir, outputFile)

			yosysCmd := exec.Command(
				"yosys",
				"-p", fmt.Sprintf(
					"read_verilog %s; hierarchy -top top_eq%d; write_cxxrtl %s",
					dutFile, i, outputFile),
			)
			yosysCmd.Dir = tmpDir

			if out, err := yosysCmd.CombinedOutput(); err != nil {
				errCh <- fmt.Errorf("Yosys failed for top_eq%d: %v\n%s", i, err, string(out))
				return
			}

			data, err := os.ReadFile(outputPath)
			if err != nil {
				errCh <- fmt.Errorf("读取 %s 失败: %v", outputFile, err)
				return
			}
			lines := strings.Split(string(data), "\n")
			if len(lines) > 5 {
				lines = lines[:len(lines)-5]
			}
			newContent := strings.Join(lines, "\n")
			if err := os.WriteFile(outputPath, []byte(newContent), 0644); err != nil {
				errCh <- fmt.Errorf("写回 %s 失败: %v", outputFile, err)
			}
		}(i)

	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			fmt.Println("Error:", err)
			return err
		}
	}

	buildCmd := `clang++ -w -g -O3 -std=c++14 ` +
		`-I $(yosys-config --datdir)/include/backends/cxxrtl/runtime ` +
		`main.cpp -o cxxsim`
	compile := exec.Command("bash", "-c", buildCmd)
	compile.Dir = tmpDir
	if out, err := compile.CombinedOutput(); err != nil {
		return fmt.Errorf("clang++ 失败: %v\n%s", err, out)
	}

	run := exec.Command("./cxxsim")
	run.Dir = tmpDir
	if out, err := run.CombinedOutput(); err != nil {
		return fmt.Errorf("CXXRTL 运行失败: %v\n%s", err, out)
	} else if len(out) != 0 {
	}

	gotBytes, err := os.ReadFile(filepath.Join(tmpDir, "output.txt"))
	if err != nil {
		return fmt.Errorf("读取 output.txt 失败: %v", err)
	}

	fmt.Println(gotBytes)
	return nil
}

func TestExpressionGenerator() {
	g := CodeGenerator.NewExpressionGenerator()
	data := []byte(g.GenerateLoopFreeModule())
	err := os.WriteFile("test.v", data, 0644)
	if err != nil {
		panic(err)
	}
	tb := []byte(g.GenerateTb())
	err = os.WriteFile("tb.v", tb, 0644)
	if err != nil {
		panic(err)
	}
	g.GenerateInputFile()

}

func saveCrashArtifacts(f *Fuzzer, logFileName, tmpFileName, reason string) {
	crashLogPath := filepath.Join(f.CrashDir, filepath.Base(logFileName))
	errMoveLog := os.Rename(logFileName, crashLogPath)
	if errMoveLog != nil {
		fmt.Printf("%v\n", errMoveLog)
	}

	appendToLogFile(crashLogPath, "\n==== 崩溃原因 ====\n"+reason+"\n")

	dstTmp := filepath.Join(f.CrashDir, filepath.Base(tmpFileName))
	if errCopy := copyFile(tmpFileName, dstTmp); errCopy != nil {
		fmt.Printf("拷贝临时文件到 crash 文件夹时出错: %v\n", errCopy)
	}
}

var Commands []string

func (f *Fuzzer) TestSynth() {
	generator := CodeGenerator.NewExpressionGenerator()
	generator.Name = "top"
	tmpFileName := f.TmpDir + GetRandomFileName("tmp", ".v", "")

	file, err := os.Create(tmpFileName)
	if err != nil {
		fmt.Println("Error creating Verilog file:", err)
		return
	}

	verilogCode := generator.GenerateLoopFreeModule()
	file.Write([]byte(verilogCode))
	file.Close()

	var wg sync.WaitGroup
	wg.Add(len(Commands))

	for i := 0; i < len(Commands); i++ {
		i := i
		command := Commands[i]

		go func(command string) {
			defer wg.Done()

			logFileName := f.LogDir + GetRandomFileName(command+"-", ".log", "")
			realCmd := fmt.Sprintf("read_verilog %s; %s", tmpFileName, command)

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
			defer cancel()

			cmd := exec.CommandContext(ctx, "yosys", "-l", logFileName, "-p", realCmd)

			var stderrBuffer bytes.Buffer
			cmd.Stderr = &stderrBuffer

			err := cmd.Run()

			exitErr, isExitErr := err.(*exec.ExitError)

			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				reason := "Yosys timeout"
				saveCrashArtifacts(f, logFileName, tmpFileName, reason)
				return
			}

			if isExitErr && exitErr.ExitCode() == 137 {
				reason := "Yosys OOM Kill"
				saveCrashArtifacts(f, logFileName, tmpFileName, reason)
				return
			}

			if err != nil {
				sanitizerError := checkSanitizerErrorFromStderr(stderrBuffer.String())

				if sanitizerError {
					appendToLogFile(logFileName, stderrBuffer.String())
					reason := "Sanitizer (ASan/UBSan)"
					saveCrashArtifacts(f, logFileName, tmpFileName, reason)
				}

			} else {
				os.Remove(logFileName)
			}
		}(command)

	}

	wg.Wait()
}

func TestOfSynth() {
	Commands = []string{
		"synth",
		"synth_achronix",
		"synth_anlogic",
		"synth_easic",
		"synth_ecp5",
		"synth_efinix",
		"synth_fabulous",
		"synth_gatemate",
		"synth_gowin",
		"synth_ice40",
		"synth_intel",
		"synth_intel_alm",
		"synth_lattice",
		"synth_nanoxplore",
		"synth_nexus",
		"synth_sf2",
		"synthprop",
		"synth_coolrunner2",
		"synth_greenpak4",
		"synth_quicklogic",
		"synth_xilinx",
	}
	fuzzer := &Fuzzer{
		StartTime: time.Now().UnixMilli(),
	}
	fuzzer.Init()
	tasks := make(chan struct{})

	for i := 0; i < 10; i++ {
		go func() {
			for range tasks {
				fuzzer.TestSynth()
			}
		}()
	}

	for {
		tasks <- struct{}{}
		// time.Sleep(time.Millisecond * 10)
	}
}

func TestOfSimulator() {
	rand.Seed(time.Now().UnixNano())
	fuzzer := &Fuzzer{
		StartTime: time.Now().UnixMilli(),
	}
	fuzzer.Init()
	tasks := make(chan struct{})

	for i := 0; i < 100; i++ {
		go func() {
			for range tasks {
				fuzzer.Fuzz()
			}
		}()
	}

	for {
		tasks <- struct{}{}
		// time.Sleep(time.Millisecond * 10)
	}
}

func TestWidthAndDepth() {
	generator := CodeGenerator.NewExpressionGenerator()
	generator.OutputNums = 1
	generator.MaxDepth = 3
	generator.AssignCount = 32
	widthContent := generator.GenerateLoopFreeModule()
	generator = CodeGenerator.NewExpressionGenerator()
	generator.OutputNums = 1
	generator.MaxDepth = 8
	generator.AssignCount = 1
	depthContent := generator.GenerateLoopFreeModule()
	file, _ := os.Create("width.v")
	file.Write([]byte(widthContent))
	file.Close()
	file, _ = os.Create("depth.v")
	file.Write([]byte(depthContent))
	file.Close()
	fmt.Println("finish")
}

func TestEqualExpressionGenerator() {
	g := CodeGenerator.NewExpressionGenerator()
	g.Name = "top"

	equalNumber := 3
	modules := g.GenerateEquivalentModulesWithOneTop(equalNumber)

	if err := os.WriteFile("test.v", []byte(modules), 0644); err != nil {
		panic(err)
	}

	tb := g.GenerateEquivalenceCheckTb(equalNumber)
	if err := os.WriteFile("tb.v", []byte(tb), 0644); err != nil {
		panic(err)
	}

	input := g.GenerateInputFile()
	if err := os.WriteFile("input.txt", []byte(input), 0644); err != nil {
		panic(err)
	}

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
				fmt.Printf("[%s] vs [%s]\n", f.VerilatorOptions[i], f.VerilatorOptions[j])

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
		fmt.Println("bug occur!!!!!!!!!")

		handleFailure(f.CrashDir, realSubDir, "", "")
	} else {
		if err := os.RemoveAll(realSubDir); err != nil {
			fmt.Printf("%v\n", err)
		}
	}

}

func (f *Fuzzer) TestEqualModulesCXXRTL(equalNumber int) {
	generator := CodeGenerator.NewExpressionGenerator()
	curMillis := time.Now().UnixMilli()
	curTimeStr := strconv.FormatInt(curMillis, 10)
	subDir := strconv.FormatInt(curMillis%1000, 10)
	tmpSubDir := filepath.Join(f.TmpDir, subDir)

	if err := os.MkdirAll(tmpSubDir, 0755); err != nil {
		fmt.Println(err)
		return
	}
	realSubDir := filepath.Join(tmpSubDir, GetRandomFileName("tmp_cxxrtl", "", ""))
	if err := os.MkdirAll(realSubDir, 0755); err != nil {
		fmt.Println(err)
		return
	}

	tmpFileName := filepath.Join(realSubDir, f.TestFileName)
	tbFileName := filepath.Join(realSubDir, "main.cpp")
	tb := generator.GenerateCXXRTLMultiModuleTestBench(equalNumber)

	if err := os.WriteFile(tbFileName, []byte(tb), 0644); err != nil {
		return
	}

	if err := os.WriteFile(tmpFileName, []byte(generator.GenerateLoopFreeEquivalentModules(equalNumber)), 0644); err != nil {
		return
	}

	var wg sync.WaitGroup
	errCh := make(chan error, equalNumber)

	for i := 0; i < equalNumber; i++ {
		i := i
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			outputFile := fmt.Sprintf("test%d.cpp", i)
			outputPath := filepath.Join(realSubDir, outputFile)

			yosysCmd := exec.Command(
				"yosys",
				"-p", fmt.Sprintf(
					"read_verilog %s; hierarchy -top top_eq%d; write_cxxrtl %s",
					tmpFileName, i, outputFile),
			)
			yosysCmd.Dir = realSubDir

			if out, err := yosysCmd.CombinedOutput(); err != nil {
				errCh <- fmt.Errorf("Yosys failed for top_eq%d: %v\n%s", i, err, string(out))
				return
			}

			data, err := os.ReadFile(outputPath)
			if err != nil {
				errCh <- fmt.Errorf(" %s  %v", outputFile, err)
				return
			}
			lines := strings.Split(string(data), "\n")
			if len(lines) > 5 {
				lines = lines[:len(lines)-5]
			}
			newContent := strings.Join(lines, "\n")
			if err := os.WriteFile(outputPath, []byte(newContent), 0644); err != nil {
				errCh <- fmt.Errorf("%s %v", outputFile, err)
			}
		}(i)

	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
	}

	inputData := generator.GenerateInputFile()

	cxxrtlData, err := f.RunCXXRTL(inputData,
		generator, realSubDir, tmpFileName, tbFileName)
	if err != nil {
		return
	}
	if !strings.Contains(string(cxxrtlData), "NO") {
		if err := os.RemoveAll(realSubDir); err != nil {
			fmt.Printf("%v\n", err)
		}
		//fmt.Println("cxxrtl finish!!!")
		return
	}
	//fmt.Println("cxxrtl bug occur!!!!!!!!!")
	uniqueCrashDir := filepath.Join(
		f.CrashDir,
		"crash_"+curTimeStr+"_"+GetRandomFileName("", "", ""),
	)
	_ = copyDir(realSubDir, uniqueCrashDir)
}

func (f *Fuzzer) TestEqualModulesIcarus(equalNumber int) {
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
	realSubDir := filepath.Join(tmpSubDir, GetRandomFileName("tmp_iverilog", "", ""))
	if err := os.MkdirAll(realSubDir, 0755); err != nil {
		fmt.Println(err)
		return
	}

	tmpFileName := filepath.Join(realSubDir, f.TestFileName)
	tbFileName := filepath.Join(realSubDir, f.TestBenchName)

	if err := os.WriteFile(tmpFileName, []byte(generator.GenerateLoopFreeEquivalentModules(equalNumber)), 0644); err != nil {
		return
	}

	tbData := generator.GenerateEquivalenceCheckTb(equalNumber)
	if err := os.WriteFile(tbFileName, []byte(tbData), 0644); err != nil {
		return
	}

	inputData := generator.GenerateInputFile()

	iverilogData, err := f.RunIVerilog(inputData,
		generator, realSubDir, tmpFileName, tbFileName)
	if err != nil {
		return
	}
	if !strings.Contains(string(iverilogData), "NO") {
		if err := os.RemoveAll(realSubDir); err != nil {
			fmt.Printf(" %v\n", err)
		}
		//fmt.Println("iverilog finish!!!")
		return
	}
	//fmt.Println("iverilog bug occur!!!!!!!!!")
	uniqueCrashDir := filepath.Join(
		f.CrashDir,
		"crash_"+curTimeStr+"_"+GetRandomFileName("", "", ""),
	)
	_ = copyDir(realSubDir, uniqueCrashDir)
}

func (f *Fuzzer) TestEqualModulesVerilator(equalNumber int) {
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
	realSubDir := filepath.Join(tmpSubDir, GetRandomFileName("tmp_verilator", "", ""))
	if err := os.MkdirAll(realSubDir, 0755); err != nil {
		fmt.Println(err)
		return
	}

	tmpFileName := filepath.Join(realSubDir, f.TestFileName)
	tbFileName := filepath.Join(realSubDir, f.TestBenchName)

	if err := os.WriteFile(tmpFileName, []byte(generator.GenerateLoopFreeEquivalentModules(equalNumber)), 0644); err != nil {
		return
	}

	tbData := generator.GenerateEquivalenceCheckTb(equalNumber)
	if err := os.WriteFile(tbFileName, []byte(tbData), 0644); err != nil {
		return
	}

	inputData := generator.GenerateInputFile()

	verilatorData, err := f.RunVerilator(inputData,
		generator, realSubDir, tmpFileName, tbFileName)
	if err != nil {
		return
	}
	if !strings.Contains(string(verilatorData), "NO") {
		if err := os.RemoveAll(realSubDir); err != nil {
			fmt.Printf(" %v\n", err)
		}
		//fmt.Println("verilator finish!!!")
		return
	}
	//fmt.Println("verilator bug occur!!!!!!!!!")
	uniqueCrashDir := filepath.Join(
		f.CrashDir,
		"crash_"+curTimeStr+"_"+GetRandomFileName("", "", ""),
	)
	_ = copyDir(realSubDir, uniqueCrashDir)
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

	if err := os.WriteFile(tmpFileName, []byte(generator.GenerateLoopFreeEquivalentModules(equalNumber)), 0644); err != nil {
		return
	}

	tbData := generator.GenerateEquivalenceCheckTb(equalNumber)
	if err := os.WriteFile(tbFileName, []byte(tbData), 0644); err != nil {
		return
	}

	inputData := generator.GenerateInputFile()

	preData, optData, err := f.RunYosysOptAndSim(inputData,
		generator, realSubDir, tmpFileName, tbFileName)
	if err != nil {
		return
	}
	if !strings.Contains(string(preData), "NO") && !strings.Contains(string(optData), "NO") {
		if err := os.RemoveAll(realSubDir); err != nil {
			fmt.Printf("%v\n", err)
		}
		//fmt.Println("yosysOpt finish!!!")
		return
	}
	//fmt.Println("yosysOpt bug occur!!!!!!!!!")
	uniqueCrashDir := filepath.Join(
		f.CrashDir,
		"crash_"+curTimeStr+"_"+GetRandomFileName("", "", ""),
	)
	_ = copyDir(realSubDir, uniqueCrashDir)
}

func RunVerilator(option string, inputData string, generator *CodeGenerator.ExpressionGenerator, realSubDir, tmpFileName, tbFileName string) ([]byte, error) {
	verilatorPath := "/root/hardware-test/verilator/verilator/bin/verilator"
	subDirName := "obj_dir"
	safeOption := strings.ReplaceAll(option, "-", "_")
	if option != "" {
		subDirName += "_" + safeOption
	}
	verilatorOutputDir := filepath.Join(realSubDir, subDirName)
	verilatorLogFileName := filepath.Join(realSubDir, fmt.Sprintf("verlator_%s.log", safeOption))
	//verilatorLogFileName := filepath.Join(realSubDir, "verilator.log")
	args := []string{
		"--binary",
		"-Wno-lint",
		"--timing",
		"--top-module", "tb_dut_module",
	}
	if option != "" {
		args = append(args, option)
	}
	args = append(args, []string{"-Mdir", verilatorOutputDir}...)

	args = append(args, tmpFileName)
	args = append(args, tbFileName)
	cmd := exec.Command(verilatorPath, args...)
	cmd.Dir = realSubDir

	logFile, err := os.Create(verilatorLogFileName)
	defer logFile.Close()
	if err != nil {
		fmt.Println("Error creating log file:", err)
		return nil, nil
	}

	var stderrBuffer bytes.Buffer
	cmd.Stdout = logFile
	cmd.Stderr = &stderrBuffer
	//fmt.Println(cmd.String())

	if err := cmd.Run(); err != nil {
		//handleFailure(f.CrashDir, realSubDir, verilatorLogFileName, stderrBuffer.String())
		fmt.Println(err)
		processCrash(verilatorLogFileName, stderrBuffer.String())
		return nil, err
	}

	testbenchInputPath := filepath.Join(verilatorOutputDir, generator.TestBenchInputFileName)
	if err := os.WriteFile(testbenchInputPath, []byte(inputData), 0644); err != nil {
		return nil, err
	}
	cmd = exec.Command("./Vtb_dut_module")
	cmd.Dir = verilatorOutputDir
	cmd.Stdout = logFile
	cmd.Stderr = &stderrBuffer
	stderrBuffer.Reset()
	if err := cmd.Run(); err != nil {
		//handleFailure(f.CrashDir, realSubDir, verilatorLogFileName, stderrBuffer.String())
		processCrash(verilatorLogFileName, stderrBuffer.String())
		return nil, err
	}

	verilatorData, err := os.ReadFile(filepath.Join(verilatorOutputDir, generator.TestBenchOutputFileName))
	if err != nil {
		//handleFailure(f.CrashDir, realSubDir, verilatorLogFileName, err.Error())
		processCrash(verilatorLogFileName, err.Error())
	}
	verilatorOut := filepath.Join(realSubDir, fmt.Sprintf("verilator_%s_output.txt", safeOption))
	_ = os.WriteFile(verilatorOut, verilatorData, 0o644)
	return verilatorData, err
}

func (f *Fuzzer) RunVerilator(inputData string, generator *CodeGenerator.ExpressionGenerator, realSubDir, tmpFileName, tbFileName string) ([]byte, error) {
	verilatorPath := "/root/hardware-test/verilator/verilator/bin/verilator"
	verilatorOutputDir := filepath.Join(realSubDir, "obj_dir")
	verilatorLogFileName := filepath.Join(realSubDir, GetRandomFileName("verilator", ".log", ""))

	args := []string{
		"--binary",
		"-Wno-lint",
		"--timing",
		tmpFileName,
		tbFileName,
	}
	cmd := exec.Command(verilatorPath, args...)
	cmd.Dir = realSubDir

	logFile, err := os.Create(verilatorLogFileName)
	defer logFile.Close()
	if err != nil {
		fmt.Println("Error creating log file:", err)
		return nil, err
	}

	var stderrBuffer bytes.Buffer
	cmd.Stdout = logFile
	cmd.Stderr = &stderrBuffer

	if err := cmd.Run(); err != nil {
		handleFailure(f.CrashDir, realSubDir, verilatorLogFileName, stderrBuffer.String())
		return nil, err
	}

	testbenchInputPath := filepath.Join(verilatorOutputDir, generator.TestBenchInputFileName)
	if err := os.WriteFile(testbenchInputPath, []byte(inputData), 0644); err != nil {
		return nil, err
	}
	cmd = exec.Command("./Vtest")
	cmd.Dir = verilatorOutputDir
	cmd.Stdout = logFile
	cmd.Stderr = &stderrBuffer
	stderrBuffer.Reset()
	if err := cmd.Run(); err != nil {
		handleFailure(f.CrashDir, realSubDir, verilatorLogFileName, stderrBuffer.String())
		return nil, err
	}

	verilatorData, err := os.ReadFile(filepath.Join(verilatorOutputDir, generator.TestBenchOutputFileName))
	if err != nil {
		handleFailure(f.CrashDir, realSubDir, verilatorLogFileName, err.Error())
		return nil, err
	}
	verilatorOut := filepath.Join(realSubDir, "verilator_output.txt")
	_ = os.WriteFile(verilatorOut, verilatorData, 0o644)
	return verilatorData, nil
}

func (f *Fuzzer) RunYosysOptAndSim(inputData string, generator *CodeGenerator.ExpressionGenerator, realSubDir, tmpFileName, tbFileName string) ([]byte, []byte, error) {
	optFileName := filepath.Join(realSubDir, "opt.v")
	yosysOptFile := filepath.Join(realSubDir, "yosys_opt.log")

	logFile, err := os.Create(yosysOptFile)
	if err != nil {
		return nil, nil, err
	}
	defer logFile.Close()

	var stderrBuffer bytes.Buffer
	cmd := exec.Command("yosys", "-p", "read_verilog test.v; opt; proc; write_verilog opt.v")
	cmd.Dir = realSubDir
	cmd.Stdout = logFile
	cmd.Stderr = &stderrBuffer
	if err := cmd.Run(); err != nil {
		handleFailure(f.CrashDir, realSubDir, yosysOptFile, stderrBuffer.String()+err.Error())
		return nil, nil, err
	}

	dataNoOpt, err := f.RunIVerilog(inputData, generator, realSubDir, tmpFileName, tbFileName)
	if err != nil {
		return nil, nil, err
	}
	noOptOutPath := filepath.Join(realSubDir, "iverilog_output_noOpt.txt")
	_ = os.WriteFile(noOptOutPath, dataNoOpt, 0o644)

	optIVerilogDir := filepath.Join(realSubDir, "iverilog_opt")
	if err := os.MkdirAll(optIVerilogDir, 0755); err != nil {
		return dataNoOpt, nil, err
	}

	aoutPath := filepath.Join(optIVerilogDir, "a.out")
	args := []string{optFileName, tbFileName, "-o", aoutPath}
	cmd = exec.Command("iverilog", args...)
	cmd.Env = append(os.Environ(), "ASAN_OPTIONS=detect_leaks=0")
	cmd.Dir = realSubDir

	optLogFile := filepath.Join(realSubDir, GetRandomFileName("iverilog_opt_", ".log", ""))
	logFile, err = os.Create(optLogFile)
	if err != nil {
		return dataNoOpt, nil, err
	}
	defer logFile.Close()

	cmd.Stdout = logFile
	cmd.Stderr = &stderrBuffer
	stderrBuffer.Reset()

	if err := cmd.Run(); err != nil {
		handleFailure(f.CrashDir, realSubDir, optLogFile, stderrBuffer.String()+err.Error())
		return dataNoOpt, nil, err
	}

	testbenchInputPath := filepath.Join(optIVerilogDir, generator.TestBenchInputFileName)
	if err := os.WriteFile(testbenchInputPath, []byte(inputData), 0644); err != nil {
		return dataNoOpt, nil, err
	}

	cmd = exec.Command("./a.out")
	cmd.Env = append(os.Environ(), "ASAN_OPTIONS=detect_leaks=0")
	cmd.Dir = optIVerilogDir
	cmd.Stdout = logFile
	cmd.Stderr = &stderrBuffer
	stderrBuffer.Reset()
	if err := cmd.Run(); err != nil {
		handleFailure(f.CrashDir, realSubDir, optLogFile, stderrBuffer.String()+err.Error())
		return dataNoOpt, nil, err
	}

	dataOpt, err := os.ReadFile(filepath.Join(optIVerilogDir, generator.TestBenchOutputFileName))
	if err != nil {
		handleFailure(f.CrashDir, realSubDir, optLogFile, err.Error())
		return dataNoOpt, nil, err
	}
	optOutPath := filepath.Join(realSubDir, "iverilog_output_Opt.txt")
	_ = os.WriteFile(optOutPath, dataOpt, 0o644)

	return dataNoOpt, dataOpt, nil
}

func (f *Fuzzer) RunIVerilog(inputData string, generator *CodeGenerator.ExpressionGenerator, realSubDir, tmpFileName, tbFileName string) ([]byte, error) {
	iverilogDir := filepath.Join(realSubDir, "iverilog")
	if err := os.MkdirAll(iverilogDir, 0755); err != nil {
		return nil, err
	}

	aoutPath := filepath.Join(iverilogDir, "a.out")
	args := []string{tmpFileName, tbFileName, "-o", aoutPath}
	cmd := exec.Command("iverilog", args...)
	cmd.Env = append(os.Environ(), "ASAN_OPTIONS=detect_leaks=0")
	cmd.Dir = realSubDir

	iverilogLogFileName := filepath.Join(realSubDir, GetRandomFileName("iverilog", ".log", ""))
	logFile, err := os.Create(iverilogLogFileName)
	defer logFile.Close()
	if err != nil {
		return nil, err
	}
	var stderrBuffer bytes.Buffer
	cmd.Stdout = logFile
	cmd.Stderr = &stderrBuffer

	if err := cmd.Run(); err != nil {
		handleFailure(f.CrashDir, iverilogLogFileName, stderrBuffer.String(), err.Error())
		return nil, err
	}

	testbenchInputPath := filepath.Join(iverilogDir, generator.TestBenchInputFileName)
	if err := os.WriteFile(testbenchInputPath, []byte(inputData), 0644); err != nil {
		return nil, err
	}
	cmd = exec.Command("./a.out")
	cmd.Env = append(os.Environ(), "ASAN_OPTIONS=detect_leaks=0")
	cmd.Dir = iverilogDir
	cmd.Stdout = logFile
	cmd.Stderr = &stderrBuffer
	stderrBuffer.Reset()
	if err := cmd.Run(); err != nil {
		handleFailure(f.CrashDir, iverilogLogFileName, stderrBuffer.String(), err.Error())
		return nil, err
	}

	iverilogData, err := os.ReadFile(filepath.Join(iverilogDir, generator.TestBenchOutputFileName))
	if err != nil {
		handleFailure(f.CrashDir, realSubDir, iverilogLogFileName, err.Error())
		return nil, err
	}

	iverilogOut := filepath.Join(realSubDir, "iverilog_output.txt")
	_ = os.WriteFile(iverilogOut, iverilogData, 0o644)
	return iverilogData, nil
}

func (f *Fuzzer) RunCXXRTL(inputData string, generator *CodeGenerator.ExpressionGenerator, realSubDir, tmpFileName, tbFileName string) ([]byte, error) {
	//printDirTree(realSubDir)
	//_ = os.Remove(generator.TestBenchOutputFileName)

	cxxrtlLog := filepath.Join(realSubDir, GetRandomFileName("cxxrtl", ".log", ""))

	logFile, err := os.Create(cxxrtlLog)
	defer logFile.Close()
	if err != nil {
		return nil, err
	}

	var cxxrtlStderr bytes.Buffer

	compileCmd := "clang++ -w -g -O3 -std=c++14 " +
		"-I $(yosys-config --datdir)/include/backends/cxxrtl/runtime " +
		"main.cpp -o cxxsim"
	build := exec.Command("bash", "-c", compileCmd)
	build.Dir = realSubDir
	build.Stdout = logFile
	build.Stderr = &cxxrtlStderr
	cxxrtlStderr.Reset()

	if err := build.Run(); err != nil {
		handleFailure(f.CrashDir, realSubDir, cxxrtlLog, cxxrtlStderr.String())
		return nil, err
	}

	tbInputPath := filepath.Join(realSubDir, generator.TestBenchInputFileName)
	if err := os.WriteFile(tbInputPath, []byte(inputData), 0o644); err != nil {
		fmt.Println("写入 CXXRTL testbench 输入失败:", err)
		return nil, err
	}

	sim := exec.Command("./cxxsim")
	sim.Dir = realSubDir
	sim.Stdout = logFile
	sim.Stderr = &cxxrtlStderr
	cxxrtlStderr.Reset()

	if err := sim.Run(); err != nil {
		handleFailure(f.CrashDir, realSubDir, cxxrtlLog, cxxrtlStderr.String())
		return nil, err
	}

	cxxrtlData, err := os.ReadFile(filepath.Join(realSubDir, "output.txt"))
	if err != nil {
		handleFailure(f.CrashDir, realSubDir, cxxrtlLog, err.Error())
		return nil, err
	}
	//printDirTree(realSubDir)
	//_ = os.Remove(generator.TestBenchOutputFileName)

	cxxrtlOut := filepath.Join(realSubDir, "cxxrtl_output.txt")
	_ = os.WriteFile(cxxrtlOut, cxxrtlData, 0o644)
	return cxxrtlData, nil
}

func (f *Fuzzer) RunYosysOpt(realSubDir, OptFileName string) error {
	var stderrBuffer bytes.Buffer
	yosysOptFile := filepath.Join(realSubDir, "yosys_opt.log")
	logFile, err := os.Create(yosysOptFile)
	if err != nil {
		return err
	}
	cmd := exec.Command("yosys", "-p", "read_verilog test.v; opt; proc; write_verilog opt.v")
	cmd.Dir = realSubDir
	cmd.Stdout = logFile
	cmd.Stderr = &stderrBuffer
	if err := cmd.Run(); err != nil {
		handleFailure(f.CrashDir, realSubDir, yosysOptFile, stderrBuffer.String()+err.Error())
		return err
	}

	optFileContent, err := os.ReadFile(OptFileName)
	if err != nil {
		return err
	}
	newContent := []byte("`timescale 1ns/1ps\n")
	newContent = append(newContent, optFileContent...)
	if err := os.WriteFile(OptFileName, newContent, 0644); err != nil {
		return err
	}
	return nil
}

var countIverilog int64 = 0
var countVerilator int64 = 0
var countYosysOpt int64 = 0
var countCXXRTL int64 = 0

var outputFile = "task_counter.txt"

func StartCounterLogger(outputFile string) {
	startTimestamp := time.Now().Unix() // 起始时间戳（秒）

	go func() {
		for {
			time.Sleep(30 * time.Second)

			currentTime := time.Now().Format("2006-01-02 15:04:05")
			currentTimestamp := time.Now().Unix()
			elapsed := currentTimestamp - startTimestamp

			iverilog := atomic.LoadInt64(&countIverilog)
			verilator := atomic.LoadInt64(&countVerilator)
			yosys := atomic.LoadInt64(&countYosysOpt)
			cxxrtl := atomic.LoadInt64(&countCXXRTL)

			line := fmt.Sprintf("[%s] Icarus=%d Verilator=%d YosysOpt=%d CXXRTL=%d Elapsed=%ds\n",
				currentTime, iverilog, verilator, yosys, cxxrtl, elapsed)

			f, err := os.OpenFile(outputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err == nil {
				defer f.Close()
				f.WriteString(line)
			}
		}
	}()
}

func EqualFuzzIverilog(workersPerType, equalNumber int) {
	fuzzerIcarus := &Fuzzer{StartTime: time.Now().UnixMilli()}
	fuzzerIcarus.Init()
	tasks := make(chan struct{})
	for i := 0; i < workersPerType; i++ {
		go func() {
			for range tasks {
				fuzzerIcarus.TestEqualModulesIcarus(equalNumber)
				atomic.AddInt64(&countIverilog, int64(equalNumber))
			}
		}()
	}
	for {
		tasks <- struct{}{}
	}
}

func EqualFuzzVerilator(workersPerType, equalNumber int) {
	fuzzerVerilator := &Fuzzer{StartTime: time.Now().UnixMilli()}
	fuzzerVerilator.Init()
	tasks := make(chan struct{})
	for i := 0; i < workersPerType; i++ {
		go func() {
			for range tasks {
				fuzzerVerilator.TestEqualModulesVerilator(equalNumber)
				atomic.AddInt64(&countVerilator, int64(equalNumber))
			}
		}()
	}
	for {
		tasks <- struct{}{}
	}
}

func EqualFuzzYosysOpt(workersPerType, equalNumber int) {

	fuzzerYosysOpt := &Fuzzer{StartTime: time.Now().UnixMilli()}
	fuzzerYosysOpt.Init()

	tasks := make(chan struct{})
	for i := 0; i < workersPerType; i++ {
		go func() {
			for range tasks {
				fuzzerYosysOpt.TestEqualModulesYosysOpt(equalNumber)
				atomic.AddInt64(&countYosysOpt, int64(equalNumber))
			}
		}()
	}
	for {
		tasks <- struct{}{}
	}
}

func EqualFuzzCXXRTL(workersPerType, equalNumber int) {
	fuzzerCXXRTL := &Fuzzer{StartTime: time.Now().UnixMilli()}
	fuzzerCXXRTL.Init()

	tasks := make(chan struct{})
	for i := 0; i < workersPerType; i++ {
		go func() {
			for range tasks {
				fuzzerCXXRTL.TestEqualModulesCXXRTL(equalNumber)
				atomic.AddInt64(&countCXXRTL, int64(equalNumber))
			}
		}()
	}
	for {
		tasks <- struct{}{}
	}
}

func TestAllEquivalence() {
	StartCounterLogger("cxxrtl_verilator_task_counter.txt")

	//go EqualFuzzIverilog(50, 10)
	go EqualFuzzVerilator(50, 10)
	//go EqualFuzzYosysOpt(30, 10)
	go EqualFuzzCXXRTL(50, 10)

	select {}
}
