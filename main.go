package main

import (
	"bytes"
	"context"
	"errors"
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
		fmt.Printf("打印目录树失败: %v\n", err)
	}
}

func (f *Fuzzer) Init() {
	baseTime := strconv.FormatInt(f.StartTime, 10)
	curDir, _ := os.Getwd()
	// 日志目录
	logDir := curDir + "/" + LOGDIR + baseTime + "/"
	f.LogDir = logDir
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		err := os.MkdirAll(logDir, 0755)
		if err != nil {
			panic(err)
		}
	}

	// 中间文件目录
	tmpDir := curDir + "/" + TMPDIR + baseTime + "/"
	f.TmpDir = tmpDir
	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		err := os.MkdirAll(tmpDir, 0755)
		if err != nil {
			panic(err)
		}
	}

	// 崩溃日志目录
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

func RunVerilator(option string, inputData string, generator *CodeGenerator.ExpressionGenerator, realSubDir, tmpFileName, tbFileName string) ([]byte, error) {
	// 运行 Verilator 编译
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

	// 运行 Verilator 模型
	testbenchInputPath := filepath.Join(verilatorOutputDir, generator.TestBenchInputFileName)
	if err := os.WriteFile(testbenchInputPath, []byte(inputData), 0644); err != nil {
		fmt.Println("写 testbench 输入出错:", err)
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
		fmt.Printf("读取verilator输出出错, %v\n", err)
		//handleFailure(f.CrashDir, realSubDir, verilatorLogFileName, err.Error())
		processCrash(verilatorLogFileName, err.Error())
	}
	verilatorOut := filepath.Join(realSubDir, fmt.Sprintf("verilator_%s_output.txt", safeOption))
	_ = os.WriteFile(verilatorOut, verilatorData, 0o644)
	return verilatorData, err
}

func (f *Fuzzer) RunIVerilog(inputData string, generator *CodeGenerator.ExpressionGenerator, realSubDir, tmpFileName, tbFileName string) ([]byte, error) {
	iverilogDir := filepath.Join(realSubDir, "iverilog")
	if err := os.MkdirAll(iverilogDir, 0755); err != nil {
		fmt.Println("创建 iverilog 目录失败:", err)
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
		fmt.Println("创建 iverilog 日志失败:", err)
		return nil, err
	}
	var stderrBuffer bytes.Buffer
	cmd.Stdout = logFile
	cmd.Stderr = &stderrBuffer

	if err := cmd.Run(); err != nil {
		handleFailure(f.CrashDir, iverilogLogFileName, stderrBuffer.String(), err.Error())
		return nil, err
	}

	// 执行 Icarus 仿真
	testbenchInputPath := filepath.Join(iverilogDir, generator.TestBenchInputFileName)
	if err := os.WriteFile(testbenchInputPath, []byte(inputData), 0644); err != nil {
		fmt.Println("写入 testbench 输入文件失败:", err)
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
		fmt.Println("读取iverilog输出出错:", err)
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
	// 1. 创建 cxxrtl 工作目录
	cxxrtlDir := filepath.Join(realSubDir, "cxxrtl")
	if err := os.MkdirAll(cxxrtlDir, 0o755); err != nil {
		fmt.Println("创建 cxxrtl 目录失败:", err)
		return nil, err
	}
	CXXTestBenchData := generator.GenerateCXXRTLTestBench()
	CXXTestBenchFile := filepath.Join(cxxrtlDir, "main.cpp")
	if err := os.WriteFile(CXXTestBenchFile, []byte(CXXTestBenchData), 0644); err != nil {
		fmt.Println("创建 cxxrtl 激励文件失败: ", err)
		return nil, err
	}

	// 2. 用 Yosys 把设计转换成 CXXRTL 的 main.cpp
	//    tb.v 依旧作为顶层；若顶层名称不是 tb，自行替换 "-top tb"
	yosysCmd := exec.Command(
		"yosys",
		"-p",
		fmt.Sprintf(
			"read_verilog %s; write_cxxrtl test.cpp",
			tmpFileName),
	)
	yosysCmd.Dir = cxxrtlDir

	cxxrtlLog := filepath.Join(realSubDir, GetRandomFileName("cxxrtl", ".log", ""))

	logFile, err := os.Create(cxxrtlLog)
	defer logFile.Close()
	if err != nil {
		fmt.Println("创建 cxxrtl 日志失败:", err)
		return nil, err
	}

	var cxxrtlStderr bytes.Buffer
	yosysCmd.Stdout = logFile
	yosysCmd.Stderr = &cxxrtlStderr

	if err := yosysCmd.Run(); err != nil {
		fmt.Println("cxxrtl生成失败")
		handleFailure(f.CrashDir, realSubDir, cxxrtlLog, cxxrtlStderr.String())
		return nil, err
	}

	// 3. 编译 CXXRTL 生成的 main.cpp
	compileCmd := "clang++ -w -g -O3 -std=c++14 " +
		"-I $(yosys-config --datdir)/include/backends/cxxrtl/runtime " +
		"main.cpp -o cxxsim"
	// 需要 shell 来展开 $(yosys-config ...)
	build := exec.Command("bash", "-c", compileCmd)
	build.Dir = cxxrtlDir
	build.Stdout = logFile
	build.Stderr = &cxxrtlStderr
	cxxrtlStderr.Reset()

	if err := build.Run(); err != nil {
		fmt.Println("cxxrtl编译失败")
		handleFailure(f.CrashDir, realSubDir, cxxrtlLog, cxxrtlStderr.String())
		return nil, err
	}

	// 4. 把 testbench 输入文件复制到 cxxrtl 目录
	tbInputPath := filepath.Join(cxxrtlDir, generator.TestBenchInputFileName)
	if err := os.WriteFile(tbInputPath, []byte(inputData), 0o644); err != nil {
		fmt.Println("写入 CXXRTL testbench 输入失败:", err)
		return nil, err
	}

	// 5. 运行 CXXRTL 仿真
	sim := exec.Command("./cxxsim")
	sim.Dir = cxxrtlDir
	sim.Stdout = logFile
	sim.Stderr = &cxxrtlStderr
	cxxrtlStderr.Reset()

	if err := sim.Run(); err != nil {
		fmt.Println("cxxrtl运行失败")
		handleFailure(f.CrashDir, realSubDir, cxxrtlLog, cxxrtlStderr.String())
		return nil, err
	}

	// 6. 读取 CXXRTL 输出
	cxxrtlData, err := os.ReadFile(filepath.Join(cxxrtlDir, "output.txt"))
	if err != nil {
		fmt.Println("读取 CXXRTL 输出出错:", err)
		handleFailure(f.CrashDir, realSubDir, cxxrtlLog, err.Error())
		return nil, err
	}
	//printDirTree(realSubDir)
	//_ = os.Remove(generator.TestBenchOutputFileName)
	/* ---------- 三方结果一致性比较 ---------- */

	cxxrtlOut := filepath.Join(realSubDir, "cxxrtl_output.txt")
	_ = os.WriteFile(cxxrtlOut, cxxrtlData, 0o644)
	return cxxrtlData, nil
}

func (f *Fuzzer) RunYosysOpt(realSubDir, OptFileName string) error {
	var stderrBuffer bytes.Buffer
	yosysOptFile := filepath.Join(realSubDir, "yosys_opt.log")
	logFile, err := os.Create(yosysOptFile)
	if err != nil {
		fmt.Println("创建 yosys_opt 日志失败:", err)
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
		fmt.Println("读取 opt.v 出错:", err)
		return err
	}
	newContent := []byte("`timescale 1ns/1ps\n")
	newContent = append(newContent, optFileContent...)
	if err := os.WriteFile(OptFileName, newContent, 0644); err != nil {
		fmt.Println("写回 opt.v 出错:", err)
		return err
	}
	return nil
}

func (f *Fuzzer) Fuzz() {
	// 生成 Verilog 相关
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

	// 生成文件路径
	tmpFileName := filepath.Join(realSubDir, f.TestFileName)
	tbFileName := filepath.Join(realSubDir, f.TestBenchName)

	// 写 test.v
	if err := os.WriteFile(tmpFileName, []byte(generator.GenerateLoopFreeModule()), 0644); err != nil {
		fmt.Println("写 test.v 出错:", err)
		return
	}

	// 写 tb.v
	tbData := generator.GenerateTb()
	if err := os.WriteFile(tbFileName, []byte(tbData), 0644); err != nil {
		fmt.Println("写 tb.v 出错:", err)
		return
	}

	inputData := generator.GenerateInputFile()

	// 运行 Verilator 编译
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

	// 运行 Verilator 模型
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
		fmt.Printf("读取verilator输出出错, %v\n", err)
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
	// 运行 Icarus Verilog
	iverilogDir := filepath.Join(realSubDir, "iverilog")
	if err := os.MkdirAll(iverilogDir, 0755); err != nil {
		fmt.Println("创建 iverilog 目录失败:", err)
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
		fmt.Println("创建 iverilog 日志失败:", err)
		return
	}

	cmd.Stdout = logFile
	cmd.Stderr = &stderrBuffer
	stderrBuffer.Reset()

	if err := cmd.Run(); err != nil {
		handleFailure(f.CrashDir, iverilogLogFileName, stderrBuffer.String(), err.Error())
		return
	}

	// 执行 Icarus 仿真
	testbenchInputPath = filepath.Join(iverilogDir, generator.TestBenchInputFileName)
	if err := os.WriteFile(testbenchInputPath, []byte(inputData), 0644); err != nil {
		fmt.Println("写入 testbench 输入文件失败:", err)
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
		fmt.Println("读取iverilog输出出错:", err)
		handleFailure(f.CrashDir, realSubDir, iverilogLogFileName, err.Error())
	}

	logFile.Close()

	iverilogOut := filepath.Join(realSubDir, "iverilog_output.txt")
	_ = os.WriteFile(iverilogOut, iverilogData, 0o644)

	//printDirTree(realSubDir)
	//_ = os.Remove(generator.TestBenchOutputFileName)
	// 1. 创建 cxxrtl 工作目录
	cxxrtlDir := filepath.Join(realSubDir, "cxxrtl")
	if err := os.MkdirAll(cxxrtlDir, 0o755); err != nil {
		fmt.Println("创建 cxxrtl 目录失败:", err)
		return
	}
	CXXTestBenchData := generator.GenerateCXXRTLTestBench()
	CXXTestBenchFile := filepath.Join(cxxrtlDir, "main.cpp")
	if err := os.WriteFile(CXXTestBenchFile, []byte(CXXTestBenchData), 0644); err != nil {
		fmt.Println("创建 cxxrtl 激励文件失败: ", err)
	}

	// 2. 用 Yosys 把设计转换成 CXXRTL 的 main.cpp
	//    tb.v 依旧作为顶层；若顶层名称不是 tb，自行替换 "-top tb"
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
		fmt.Println("创建 cxxrtl 日志失败:", err)
		return
	}
	defer logFile.Close()

	var cxxrtlStderr bytes.Buffer
	yosysCmd.Stdout = logFile
	yosysCmd.Stderr = &cxxrtlStderr

	if err := yosysCmd.Run(); err != nil {
		fmt.Println("cxxrtl生成失败")
		handleFailure(f.CrashDir, realSubDir, cxxrtlLog, cxxrtlStderr.String())
		return
	}

	// 3. 编译 CXXRTL 生成的 main.cpp
	compileCmd := "clang++ -w -g -O3 -std=c++14 " +
		"-I $(yosys-config --datdir)/include/backends/cxxrtl/runtime " +
		"main.cpp -o cxxsim"
	// 需要 shell 来展开 $(yosys-config ...)
	build := exec.Command("bash", "-c", compileCmd)
	build.Dir = cxxrtlDir
	build.Stdout = logFile
	build.Stderr = &cxxrtlStderr
	cxxrtlStderr.Reset()

	if err := build.Run(); err != nil {
		fmt.Println("cxxrtl编译失败")
		handleFailure(f.CrashDir, realSubDir, cxxrtlLog, cxxrtlStderr.String())
		return
	}

	// 4. 把 testbench 输入文件复制到 cxxrtl 目录
	tbInputPath := filepath.Join(cxxrtlDir, generator.TestBenchInputFileName)
	if err := os.WriteFile(tbInputPath, []byte(inputData), 0o644); err != nil {
		fmt.Println("写入 CXXRTL testbench 输入失败:", err)
		return
	}

	// 5. 运行 CXXRTL 仿真
	sim := exec.Command("./cxxsim")
	sim.Dir = cxxrtlDir
	sim.Stdout = logFile
	sim.Stderr = &cxxrtlStderr
	cxxrtlStderr.Reset()

	if err := sim.Run(); err != nil {
		fmt.Println("cxxrtl运行失败")
		handleFailure(f.CrashDir, realSubDir, cxxrtlLog, cxxrtlStderr.String())
		return
	}

	// 6. 读取 CXXRTL 输出
	cxxrtlData, err := os.ReadFile(filepath.Join(cxxrtlDir, "output.txt"))
	if err != nil {
		fmt.Println("读取 CXXRTL 输出出错:", err)
		handleFailure(f.CrashDir, realSubDir, cxxrtlLog, err.Error())
	}

	logFile.Close()
	//printDirTree(realSubDir)
	//_ = os.Remove(generator.TestBenchOutputFileName)
	/* ---------- 三方结果一致性比较 ---------- */

	cxxrtlOut := filepath.Join(realSubDir, "cxxrtl_output.txt")
	_ = os.WriteFile(cxxrtlOut, cxxrtlData, 0o644)

	equalVI := bytes.Equal(verilatorData, iverilogData)
	equalVC := bytes.Equal(verilatorData, cxxrtlData)
	equalIC := bytes.Equal(iverilogData, cxxrtlData)

	if equalVI && equalVC && equalIC { // 三个完全一致
		fmt.Println(curTimeStr + " 三个结果完全相同")
		if err := os.RemoveAll(realSubDir); err != nil {
			fmt.Printf("删除临时测试目录失败: %v\n", err)
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
	// 任意不一致即判定 bug
	fmt.Println(diffStr + "bug occur!!!!!!!!!!!!!!!!!")

	// ⬇️ 把三个仿真输出写到文件留档

	// 生成 diff.txt（以 Verilator 为基准）
	diffContent := diffStr + "\n==== Verilator vs CXXRTL Diff ====\n" + diffLines(verilatorData, cxxrtlData) +
		"\n==== Verilator vs Icarus Diff ====\n" +
		diffLines(verilatorData, iverilogData) +
		"\n==== Icarus vs CXXRTL Diff ====\n" +
		diffLines(iverilogData, cxxrtlData)
	diffFile := filepath.Join(realSubDir, "diff.txt")
	_ = os.WriteFile(diffFile, []byte(diffContent), 0o644)

	// 标记 panic
	_ = os.WriteFile(
		filepath.Join(f.TmpDir, "panic.log"),
		[]byte("bug occur!!!!!!!!!"),
		0o644,
	)

	// 把现场复制到 crash 目录
	uniqueCrashDir := filepath.Join(
		f.CrashDir,
		"crash_"+curTimeStr+"_"+GetRandomFileName("", "", ""),
	)
	_ = copyDir(realSubDir, uniqueCrashDir)

}

func (f *Fuzzer) TestYosysOptUsingVerilatorWithManyOptions() {
	// 生成 Verilog 相关
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

	// 生成文件路径
	tmpFileName := filepath.Join(realSubDir, f.TestFileName)
	OptFileName := filepath.Join(realSubDir, "opt.v")
	tbFileName := filepath.Join(realSubDir, f.TestBenchName)

	// 写 test.v
	if err := os.WriteFile(tmpFileName, []byte(generator.GenerateLoopFreeModule()), 0644); err != nil {
		fmt.Println("写 test.v 出错:", err)
		return
	}

	// 写 tb.v
	tbData := generator.GenerateTb()
	if err := os.WriteFile(tbFileName, []byte(tbData), 0644); err != nil {
		fmt.Println("写 tb.v 出错:", err)
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
				fmt.Printf("不一致: [%s] vs [%s]\n", f.VerilatorOptions[i], f.VerilatorOptions[j])

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
		// 标记 panic
		fmt.Println("bug occur!!!!!!!!!")

		// 保存 crash 目录
		handleFailure(f.CrashDir, realSubDir, "", "")
		fmt.Println("发现差异，已保存至 crash 目录")
	} else {
		fmt.Println(curTimeStr + " 所有 Verilator 优化选项结果一致")
		if err := os.RemoveAll(realSubDir); err != nil {
			fmt.Printf("删除临时测试目录失败: %v\n", err)
		}
	}

}

func (f *Fuzzer) TestYosysOptUsingVerilator() {
	// 生成 Verilog 相关
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

	// 生成文件路径
	tmpFileName := filepath.Join(realSubDir, f.TestFileName)
	OptFileName := filepath.Join(realSubDir, "opt.v")
	tbFileName := filepath.Join(realSubDir, f.TestBenchName)

	// 写 test.v
	if err := os.WriteFile(tmpFileName, []byte(generator.GenerateLoopFreeModule()), 0644); err != nil {
		fmt.Println("写 test.v 出错:", err)
		return
	}

	// 写 tb.v
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
		fmt.Println("创建 yosys_opt 日志失败:", err)
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
		fmt.Println("读取 opt.v 出错:", err)
		return
	}
	newContent := []byte("`timescale 1ns/1ps\n")
	newContent = append(newContent, optFileContent...)
	if err := os.WriteFile(OptFileName, newContent, 0644); err != nil {
		fmt.Println("写回 opt.v 出错:", err)
		return
	}

	//printDirTree(realSubDir)
	//_ = os.Remove(generator.TestBenchOutputFileName)
	// 运行 Icarus Verilog
	// 运行 Verilator 编译
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

	// 运行 Verilator 模型
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
		fmt.Printf("读取verilator输出出错, %v\n", err)
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

	// 运行 Verilator 模型
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
		fmt.Printf("读取verilator输出出错, %v\n", err)
		handleFailure(f.CrashDir, realSubDir, verilatorLogFileName, err.Error())
	}

	logFile.Close()

	verilatorOut = filepath.Join(realSubDir, "verilator_opt_output.txt")
	_ = os.WriteFile(verilatorOut, verilatorOptData, 0o644)

	equalOpt := bytes.Equal(verilatorData, verilatorOptData)

	if equalOpt { // 三个完全一致
		fmt.Println(curTimeStr + " 结果完全相同")
		if err := os.RemoveAll(realSubDir); err != nil {
			fmt.Printf("删除临时测试目录失败: %v\n", err)
		}
		return
	}
	// 任意不一致即判定 bug
	fmt.Println("bug occur!!!!!!!!!!!!!!!!!")

	// ⬇️ 把三个仿真输出写到文件留档

	// 生成 diff.txt（以 Verilator 为基准）
	diffContent := "\n==== NoOpt vs Opt Diff ====\n" + diffLines(verilatorData, verilatorOptData)
	diffFile := filepath.Join(realSubDir, "diff.txt")
	_ = os.WriteFile(diffFile, []byte(diffContent), 0o644)

	// 标记 panic
	_ = os.WriteFile(
		filepath.Join(f.TmpDir, "panic.log"),
		[]byte("bug occur!!!!!!!!!"),
		0o644,
	)

	// 把现场复制到 crash 目录
	uniqueCrashDir := filepath.Join(
		f.CrashDir,
		"crash_"+curTimeStr+"_"+GetRandomFileName("", "", ""),
	)
	_ = copyDir(realSubDir, uniqueCrashDir)
}

func (f *Fuzzer) TestYosysOpt() {
	// 生成 Verilog 相关
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

	// 生成文件路径
	tmpFileName := filepath.Join(realSubDir, f.TestFileName)
	OptFileName := filepath.Join(realSubDir, "opt.v")
	tbFileName := filepath.Join(realSubDir, f.TestBenchName)

	// 写 test.v
	if err := os.WriteFile(tmpFileName, []byte(generator.GenerateLoopFreeModule()), 0644); err != nil {
		fmt.Println("写 test.v 出错:", err)
		return
	}

	// 写 tb.v
	tbData := generator.GenerateTb()
	if err := os.WriteFile(tbFileName, []byte(tbData), 0644); err != nil {
		fmt.Println("写 tb.v 出错:", err)
		return
	}

	var stderrBuffer bytes.Buffer
	inputData := generator.GenerateInputFile()
	yosysOptFile := filepath.Join(realSubDir, "yosys_opt.log")
	logFile, err := os.Create(yosysOptFile)
	if err != nil {
		fmt.Println("创建 yosys_opt 日志失败:", err)
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
	// 运行 Icarus Verilog
	iverilogDir := filepath.Join(realSubDir, "iverilog")
	if err := os.MkdirAll(iverilogDir, 0755); err != nil {
		fmt.Println("创建 iverilog 目录失败:", err)
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
		fmt.Println("创建 iverilog 日志失败:", err)
		return
	}

	cmd.Stdout = logFile
	cmd.Stderr = &stderrBuffer
	stderrBuffer.Reset()

	if err := cmd.Run(); err != nil {
		handleFailure(f.CrashDir, realSubDir, iverilogLogFileName, stderrBuffer.String()+err.Error())
		return
	}

	// 执行 Icarus 仿真
	testbenchInputPath := filepath.Join(iverilogDir, generator.TestBenchInputFileName)
	if err := os.WriteFile(testbenchInputPath, []byte(inputData), 0644); err != nil {
		fmt.Println("写入 testbench 输入文件失败:", err)
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
		fmt.Println("读取iverilog输出出错:", err)
		handleFailure(f.CrashDir, realSubDir, iverilogLogFileName, err.Error())
	}

	logFile.Close()

	iverilogOut := filepath.Join(realSubDir, "iverilog_output_noOpt.txt")
	_ = os.WriteFile(iverilogOut, iverilogData, 0o644)

	//opt

	iverilogDir = filepath.Join(realSubDir, "iverilog_opt")
	if err := os.MkdirAll(iverilogDir, 0755); err != nil {
		fmt.Println("创建 iverilog_opt 目录失败:", err)
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
		fmt.Println("创建 iverilog_opt 日志失败:", err)
		return
	}

	cmd.Stdout = logFile
	cmd.Stderr = &stderrBuffer
	stderrBuffer.Reset()

	if err := cmd.Run(); err != nil {
		handleFailure(f.CrashDir, realSubDir, iverilogLogFileName, stderrBuffer.String()+err.Error())
		return
	}

	// 执行 Icarus 仿真
	testbenchInputPath = filepath.Join(iverilogDir, generator.TestBenchInputFileName)
	if err := os.WriteFile(testbenchInputPath, []byte(inputData), 0644); err != nil {
		fmt.Println("写入 testbench 输入文件失败:", err)
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
		fmt.Println("读取iverilog_opt输出出错:", err)
		handleFailure(f.CrashDir, realSubDir, iverilogLogFileName, err.Error())
	}

	logFile.Close()

	iverilogOut = filepath.Join(realSubDir, "iverilog_output_Opt.txt")
	_ = os.WriteFile(iverilogOut, iverilogOptData, 0o644)

	equalOpt := bytes.Equal(iverilogOptData, iverilogData)

	if equalOpt { // 三个完全一致
		fmt.Println(curTimeStr + " 结果完全相同")
		if err := os.RemoveAll(realSubDir); err != nil {
			fmt.Printf("删除临时测试目录失败: %v\n", err)
		}
		return
	}
	// 任意不一致即判定 bug
	fmt.Println("bug occur!!!!!!!!!!!!!!!!!")

	// ⬇️ 把三个仿真输出写到文件留档

	// 生成 diff.txt（以 Verilator 为基准）
	diffContent := "\n==== NoOpt vs Opt Diff ====\n" + diffLines(iverilogData, iverilogOptData)
	diffFile := filepath.Join(realSubDir, "diff.txt")
	_ = os.WriteFile(diffFile, []byte(diffContent), 0o644)

	// 标记 panic
	_ = os.WriteFile(
		filepath.Join(f.TmpDir, "panic.log"),
		[]byte("bug occur!!!!!!!!!"),
		0o644,
	)

	// 把现场复制到 crash 目录
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
		fmt.Printf("%s 失败且检测到 ASan 错误，保留日志和 Verilog 文件。\n")
	}
	appendToLogFile(logFile, stderr)
}

func handleFailure(crashDir, realSubDir, logFile string, stderr string) {
	if logFile != "" {
		processCrash(logFile, stderr)
	}

	// 把测试现场复制到 crash 子目录
	curTimeStr := strconv.FormatInt(time.Now().UnixMilli(), 10)
	crashSubdir := filepath.Join(crashDir, "crash_"+curTimeStr+"_"+GetRandomFileName("", "", ""))
	if err := copyDir(realSubDir, crashSubdir); err != nil {
		fmt.Printf("复制出错目录失败：%v\n", err)
	}

	// 删除测试目录
	if err := os.RemoveAll(realSubDir); err != nil {
		fmt.Printf("删除临时测试目录失败: %v\n", err)
	}
}

// 将 ASan/UBSan 错误输出追加到日志文件末尾
func appendToLogFile(logFileName, stderrContent string) {
	file, err := os.OpenFile(logFileName, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("无法打开日志文件 %s 追加 sanitizer 错误: %v\n", logFileName, err)
		return
	}
	defer file.Close()

	// 追加 sanitizer 错误到日志文件
	_, err = file.WriteString("\n==== Sanitizer 错误输出 ====\n" + stderrContent + "\n")
	if err != nil {
		fmt.Printf("写入 sanitizer 错误到日志文件 %s 失败: %v\n", logFileName, err)
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

// 检查 stderr 是否包含 ASan 或 UBSan 相关的错误信息
func checkSanitizerErrorFromStderr(stderr string) bool {
	// ASan 和 UBSan 相关的常见错误关键词
	sanitizerKeywords := []string{
		// ASan 错误
		"AddressSanitizer", "heap-buffer-overflow", "stack-buffer-overflow",
		"use-after-free", "global-buffer-overflow", "double-free",
		"invalid-pointer", "shadow memory",

		// UBSan 错误
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

func main() {
	TestEqualExpressionGenerator()
}

// TestSimpleCXXRTL 用固定电路验证 CXXRTL 流程能否正常工作。
// 返回 nil 表示通过，返回 error 表示流程或结果不正确。
func TestSimpleCXXRTL() error {
	// 1. 创建临时目录并写入 Verilog 源文件
	tmpDir, err := os.MkdirTemp("", "cxxrtl_simple_")
	fmt.Println(tmpDir)
	//defer os.RemoveAll(tmpDir)
	if err != nil {
		return fmt.Errorf("创建临时目录失败: %v", err)
	}
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

	// 2. 调 Yosys 生成 CXXRTL main.cpp
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

	// 3. 编译 main.cpp
	buildCmd := `clang++ -w -g -O3 -std=c++14 ` +
		`-I $(yosys-config --datdir)/include/backends/cxxrtl/runtime ` +
		`main.cpp -o cxxsim`
	compile := exec.Command("bash", "-c", buildCmd)
	compile.Dir = tmpDir
	if out, err := compile.CombinedOutput(); err != nil {
		return fmt.Errorf("clang++ 失败: %v\n%s", err, out)
	}

	// 4. 运行仿真
	run := exec.Command("./cxxsim")
	run.Dir = tmpDir
	if out, err := run.CombinedOutput(); err != nil {
		return fmt.Errorf("CXXRTL 运行失败: %v\n%s", err, out)
	} else if len(out) != 0 {
		// CXXRTL 默认没有 stdout；有输出也无妨
	}

	// 5. 读取并校验 out.txt
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
	// 移动日志文件到 crash 文件夹
	crashLogPath := filepath.Join(f.CrashDir, filepath.Base(logFileName))
	errMoveLog := os.Rename(logFileName, crashLogPath)
	if errMoveLog != nil {
		fmt.Printf("移动日志文件到 crash 文件夹时出错: %v\n", errMoveLog)
	}

	// 追加错误原因到日志
	appendToLogFile(crashLogPath, "\n==== 崩溃原因 ====\n"+reason+"\n")

	// 拷贝 tmp 文件
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

			// 设置2小时超时
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
			defer cancel()

			cmd := exec.CommandContext(ctx, "yosys", "-l", logFileName, "-p", realCmd)

			var stderrBuffer bytes.Buffer
			cmd.Stderr = &stderrBuffer

			err := cmd.Run()

			exitErr, isExitErr := err.(*exec.ExitError)

			// 检查是否超时
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				fmt.Printf("Yosys 命令超时，强制终止，保留日志和 Verilog 文件。\n")
				reason := "Yosys 命令超时 (超过2小时)"
				saveCrashArtifacts(f, logFileName, tmpFileName, reason)
				return
			}

			// 检查 OOM Kill (exit status 137)
			if isExitErr && exitErr.ExitCode() == 137 {
				fmt.Printf("Yosys 被 OOM Kill (退出码 137)，保留日志和 Verilog 文件。\n")
				reason := "Yosys 进程被 OOM Kill (退出码 137)"
				saveCrashArtifacts(f, logFileName, tmpFileName, reason)
				return
			}

			// 非超时且非 OOM 的情况
			if err != nil {
				sanitizerError := checkSanitizerErrorFromStderr(stderrBuffer.String())

				if sanitizerError {
					appendToLogFile(logFileName, stderrBuffer.String())
					fmt.Printf("Yosys 失败且检测到 ASan/UBSan 错误，保留日志和 Verilog 文件。\n")
					reason := "检测到 Sanitizer 错误 (ASan/UBSan)"
					saveCrashArtifacts(f, logFileName, tmpFileName, reason)
				} else {
					fmt.Println("Yosys 失败，但未检测到 ASan/UBSan 相关错误")
				}

			} else {
				fmt.Println("Yosys 命令执行成功，删除日志文件。")
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

	// 设置生成几个等价版本
	equalNumber := 3

	// 生成多个等价模块
	modules := g.GenerateLoopFreeEquivalentModules(equalNumber)

	for i, module := range modules {
		moduleFile := fmt.Sprintf("test_eq%d.v", i)
		err := os.WriteFile(moduleFile, []byte(module), 0644)
		if err != nil {
			panic(err)
		}
	}
	tb := []byte(g.GenerateTb())
	err := os.WriteFile("tb.v", tb, 0644)
	if err != nil {
		panic(err)
	}
	g.GenerateInputFile()

}
