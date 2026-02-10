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
		toolConfig.YosysPath,
		"-p", fmt.Sprintf(
			"read_verilog %s; write_cxxrtl test.cpp",
			dutFile),
	)
	yosysCmd.Dir = tmpDir
	if out, err := yosysCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("Yosys 失败: %v\n%s", err, out)
	}

	buildCmd := fmt.Sprintf("%s -w -g -O3 -std=c++14 -I $(%s --datdir)/include/backends/cxxrtl/runtime main.cpp -o cxxsim",
		toolConfig.ClangXXPath, toolConfig.YosysConfigPath)
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
				toolConfig.YosysPath,
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

	buildCmd := fmt.Sprintf("%s -w -g -O3 -std=c++14 -I $(%s --datdir)/include/backends/cxxrtl/runtime main.cpp -o cxxsim",
		toolConfig.ClangXXPath, toolConfig.YosysConfigPath)
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

	if f.EnableDiffSim {
		modules := generator.GenerateLoopFreeEquivalentModules(equalNumber)
		if err := os.WriteFile(tmpFileName, []byte(modules), 0644); err != nil {
			return
		}

		eqTb := generator.GenerateCXXRTLMultiModuleTestBench(equalNumber)
		if err := os.WriteFile(filepath.Join(realSubDir, "main_eq.cpp"), []byte(eqTb), 0644); err != nil {
			return
		}

		originalName := generator.Name
		topEq0Name := fmt.Sprintf("%s_eq0", originalName)
		generator.Name = fmt.Sprintf("%s__eq0", originalName)
		tb := generator.GenerateCXXRTLTestBench()
		generator.Name = originalName
		if err := os.WriteFile(tbFileName, []byte(tb), 0644); err != nil {
			return
		}

		outputPath := filepath.Join(realSubDir, "test.cpp")
		yosysCmd := exec.Command(
			toolConfig.YosysPath,
			"-p", fmt.Sprintf(
				"read_verilog %s; hierarchy -top %s; write_cxxrtl test.cpp",
				tmpFileName, topEq0Name),
		)
		yosysCmd.Dir = realSubDir
		if out, err := yosysCmd.CombinedOutput(); err != nil {
			fmt.Println("Error:", fmt.Errorf("Yosys failed: %v\n%s", err, string(out)))
			return
		}
		data, err := os.ReadFile(outputPath)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		lines := strings.Split(string(data), "\n")
		if len(lines) > 5 {
			lines = lines[:len(lines)-5]
		}
		newContent := strings.Join(lines, "\n")
		if err := os.WriteFile(outputPath, []byte(newContent), 0644); err != nil {
			fmt.Println("Error:", err)
			return
		}

		inputData := generator.GenerateInputFile()
		cxxrtlData, err := f.RunCXXRTL(inputData,
			generator, realSubDir, tmpFileName, tbFileName)
		if err != nil {
			return
		}

		iverilogTbFile := filepath.Join(realSubDir, "tb_diff.v")
		tbData := generateEq0Tb(generator)
		if err := os.WriteFile(iverilogTbFile, []byte(tbData), 0644); err != nil {
			return
		}
		iverilogData, err := f.RunIVerilog(inputData,
			generator, realSubDir, tmpFileName, iverilogTbFile)
		if err != nil {
			return
		}

		shouldReport := !bytes.Equal(cxxrtlData, iverilogData)
		if !shouldReport {
			if err := os.RemoveAll(realSubDir); err != nil {
				fmt.Printf("%v\n", err)
			}
			PrettyOK("cxxrtl", "finish")
			return
		}
		PrettyBug("cxxrtl", "bug detected")
		uniqueCrashDir := filepath.Join(
			f.CrashDir,
			"bug_"+curTimeStr+"_"+GetRandomFileName("", "", ""),
		)
		_ = copyCrashArtifacts(realSubDir, uniqueCrashDir)
		return
	}

	modules := generator.GenerateLoopFreeEquivalentModules(equalNumber)
	if err := os.WriteFile(tmpFileName, []byte(modules), 0644); err != nil {
		return
	}

	tb := generator.GenerateCXXRTLMultiModuleTestBench(equalNumber)
	if err := os.WriteFile(tbFileName, []byte(tb), 0644); err != nil {
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
				toolConfig.YosysPath,
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

	sameMismatch := strings.Contains(string(cxxrtlData), "NO")

	shouldReport := false
	shouldReport = sameMismatch

	if !shouldReport {
		if err := os.RemoveAll(realSubDir); err != nil {
			fmt.Printf("%v\n", err)
		}
		PrettyOK("cxxrtl", "finish")
		return
	}
	if sameMismatch {
		PrettyBug("cxxrtl", "bug detected")
	}
	uniqueCrashDir := filepath.Join(
		f.CrashDir,
		"bug_"+curTimeStr+"_"+GetRandomFileName("", "", ""),
	)
	_ = copyCrashArtifacts(realSubDir, uniqueCrashDir)
}
