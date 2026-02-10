package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
)

func printDirTree(dir string) {
	cmd := exec.Command(toolConfig.TreePath, dir)
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
