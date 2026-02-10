package main

import (
	"VeriEQ/CodeGenerator"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func RunVerilator(option string, inputData string, generator *CodeGenerator.ExpressionGenerator, realSubDir, tmpFileName, tbFileName string) ([]byte, error) {
	verilatorPath := toolConfig.VerilatorPath
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

func (f *Fuzzer) RunVerilator(inputData string, generator *CodeGenerator.ExpressionGenerator, realSubDir, tmpFileName, tbFileName, topModule string) ([]byte, error) {
	verilatorPath := toolConfig.VerilatorPath
	verilatorOutputDir := filepath.Join(realSubDir, "obj_dir")
	verilatorLogFileName := filepath.Join(realSubDir, GetRandomFileName("verilator", ".log", ""))

	args := []string{
		"--binary",
		"-Wno-lint",
		"--timing",
	}
	if topModule != "" {
		args = append(args, "--top-module", topModule)
	}
	args = append(args, tmpFileName, tbFileName)
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
	binaryName := "./Vtest"
	if topModule != "" {
		binaryName = "./V" + topModule
	}
	cmd = exec.Command(binaryName)
	cmd.Dir = verilatorOutputDir
	cmd.Stdout = logFile
	cmd.Stderr = &stderrBuffer
	stderrBuffer.Reset()
	if err := cmd.Run(); err != nil {
		handleFailure(f.CrashDir, realSubDir, verilatorLogFileName, stderrBuffer.String()+err.Error())
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
