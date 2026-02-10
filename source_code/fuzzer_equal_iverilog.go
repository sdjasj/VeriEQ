package main

import (
	"VeriEQ/CodeGenerator"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

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
		fmt.Println(err.Error())
		return
	}

	sameMismatch := strings.Contains(string(iverilogData), "NO")
	if f.EnableDiffSim {
		iverilogEquivOut := filepath.Join(realSubDir, "iverilog_equiv_output.txt")
		_ = os.WriteFile(iverilogEquivOut, iverilogData, 0o644)
	}

	crossMismatch := false
	if f.EnableDiffSim {
		tbDiffFile := filepath.Join(realSubDir, "tb_diff.v")
		tbDiffData := generateEq0Tb(generator)
		if err := os.WriteFile(tbDiffFile, []byte(tbDiffData), 0644); err != nil {
			return
		}
		iverilogDiffData, err := f.RunIVerilog(inputData,
			generator, realSubDir, tmpFileName, tbDiffFile)
		if err != nil {
			return
		}
		verilatorDiffData, err := f.RunVerilator(inputData,
			generator, realSubDir, tmpFileName, tbDiffFile, "tb_dut_module")
		if err != nil {
			return
		}
		if !bytes.Equal(verilatorDiffData, iverilogDiffData) {
			crossMismatch = true
			diffContent := "==== Verilator vs Icarus Diff ====\n" +
				diffLines(verilatorDiffData, iverilogDiffData)
			diffFile := filepath.Join(realSubDir, "diff_verilator_vs_iverilog.txt")
			_ = os.WriteFile(diffFile, []byte(diffContent), 0o644)
		}
	}

	shouldReport := false
	if f.EnableDiffSim {
		shouldReport = crossMismatch
	} else {
		shouldReport = sameMismatch
	}

	if !shouldReport {
		if err := os.RemoveAll(realSubDir); err != nil {
			fmt.Printf(" %v\n", err)
		}
		PrettyOK("iverilog", "finish")
		return
	}
	if f.EnableDiffSim {
		if crossMismatch {
			PrettyBug("iverilog", "bug detected")
		}
	} else if sameMismatch {
		PrettyBug("iverilog", "bug detected")
	}
	uniqueCrashDir := filepath.Join(
		f.CrashDir,
		"bug_"+curTimeStr+"_"+GetRandomFileName("", "", ""),
	)
	_ = copyCrashArtifacts(realSubDir, uniqueCrashDir)
}
