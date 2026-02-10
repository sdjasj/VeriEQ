package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

type ToolConfig struct {
	BinaryRoot      string `json:"binary_root"`
	VerilatorPath   string `json:"verilator_path"`
	IverilogPath    string `json:"iverilog_path"`
	YosysPath       string `json:"yosys_path"`
	YosysConfigPath string `json:"yosys_config_path"`
	ClangXXPath     string `json:"clangxx_path"`
	TreePath        string `json:"tree_path"`
}

func defaultToolConfig() ToolConfig {
	return ToolConfig{
		BinaryRoot:      "target/binary",
		VerilatorPath:   filepath.Join("target", "binary", "verilator", "bin", "verilator"),
		IverilogPath:    filepath.Join("target", "binary", "iverilog", "bin", "iverilog"),
		YosysPath:       filepath.Join("target", "binary", "yosys", "bin", "yosys"),
		YosysConfigPath: filepath.Join("target", "binary", "yosys", "bin", "yosys-config"),
		ClangXXPath:     "clang++",
		TreePath:        "tree",
	}
}

func (c *ToolConfig) applyDefaults() {
	if c.BinaryRoot == "" {
		c.BinaryRoot = "target/binary"
	}
	if c.VerilatorPath == "" {
		c.VerilatorPath = filepath.Join(c.BinaryRoot, "verilator", "bin", "verilator")
	}
	if c.IverilogPath == "" {
		c.IverilogPath = filepath.Join(c.BinaryRoot, "iverilog", "bin", "iverilog")
	}
	if c.YosysPath == "" {
		c.YosysPath = filepath.Join(c.BinaryRoot, "yosys", "bin", "yosys")
	}
	if c.YosysConfigPath == "" {
		c.YosysConfigPath = filepath.Join(c.BinaryRoot, "yosys", "bin", "yosys-config")
	}
	if c.ClangXXPath == "" {
		c.ClangXXPath = "clang++"
	}
	if c.TreePath == "" {
		c.TreePath = "tree"
	}
}

func resolvePath(baseDir, value string) string {
	if value == "" || filepath.IsAbs(value) {
		return value
	}
	if !strings.ContainsRune(value, os.PathSeparator) {
		return value
	}
	return filepath.Clean(filepath.Join(baseDir, value))
}

func (c *ToolConfig) resolvePaths(baseDir string) {
	c.BinaryRoot = resolvePath(baseDir, c.BinaryRoot)
	c.VerilatorPath = resolvePath(baseDir, c.VerilatorPath)
	c.IverilogPath = resolvePath(baseDir, c.IverilogPath)
	c.YosysPath = resolvePath(baseDir, c.YosysPath)
	c.YosysConfigPath = resolvePath(baseDir, c.YosysConfigPath)
	c.ClangXXPath = resolvePath(baseDir, c.ClangXXPath)
	c.TreePath = resolvePath(baseDir, c.TreePath)
}

func LoadToolConfig(path string) (ToolConfig, error) {
	cfg := ToolConfig{}
	configPath := path
	if configPath == "" {
		candidates := []string{
			"config.json",
			filepath.Join("code", "source_code", "config.json"),
		}
		for _, candidate := range candidates {
			if _, err := os.Stat(candidate); err == nil {
				configPath = candidate
				break
			}
		}
	}

	if configPath == "" {
		cfg = defaultToolConfig()
		baseDir, _ := os.Getwd()
		cfg.resolvePaths(baseDir)
		return cfg, errors.New("config file not found")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		cfg = defaultToolConfig()
		baseDir, _ := os.Getwd()
		cfg.resolvePaths(baseDir)
		return cfg, err
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		cfg = defaultToolConfig()
		baseDir, _ := os.Getwd()
		cfg.resolvePaths(baseDir)
		return cfg, err
	}
	cfg.applyDefaults()

	absPath, err := filepath.Abs(configPath)
	if err != nil {
		baseDir, _ := os.Getwd()
		cfg.resolvePaths(baseDir)
		return cfg, err
	}
	cfg.resolvePaths(filepath.Dir(absPath))
	return cfg, nil
}

var toolConfig = defaultToolConfig()
