package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

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
	crashSubdir := filepath.Join(crashDir, "bug_"+curTimeStr+"_"+GetRandomFileName("", "", ""))
	if err := copyCrashArtifacts(realSubDir, crashSubdir); err != nil {
		fmt.Printf("%v\n", err)
	}
	PrettyBug("runtime", "bug detected")

	// 删除测试目录
	if err := os.RemoveAll(realSubDir); err != nil {
		fmt.Printf("%v\n", err)
	}
}

func copyCrashArtifacts(srcDir, dstDir string) error {
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return err
	}

	allowed := map[string]bool{
		"input.txt":   true,
		"test.v":      true,
		"tb.v":        true,
		"tb_diff.v":   true,
		"main.cpp":    true,
	}
	seen := map[string]bool{}

	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		base := filepath.Base(path)
		if !allowed[base] {
			return nil
		}
		if seen[base] {
			return nil
		}
		dstPath := filepath.Join(dstDir, base)
		if err := copyFile(path, dstPath); err != nil {
			return err
		}
		seen[base] = true
		return nil
	})
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

func saveCrashArtifacts(f *Fuzzer, logFileName, tmpFileName, reason string) {
	_ = logFileName
	_ = reason
	curTimeStr := strconv.FormatInt(time.Now().UnixMilli(), 10)
	crashSubdir := filepath.Join(f.CrashDir, "bug_"+curTimeStr+"_"+GetRandomFileName("", "", ""))
	if err := copyCrashArtifacts(filepath.Dir(tmpFileName), crashSubdir); err != nil {
		fmt.Printf("拷贝崩溃文件时出错: %v\n", err)
	}
	PrettyBug("runtime", "bug detected")
}
