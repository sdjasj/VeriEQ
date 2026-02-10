package main

import (
	"VeriEQ/CodeGenerator"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

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

	compileCmd := fmt.Sprintf("%s -w -g -O3 -std=c++14 -I $(%s --datdir)/include/backends/cxxrtl/runtime main.cpp -o cxxsim",
		toolConfig.ClangXXPath, toolConfig.YosysConfigPath)
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
