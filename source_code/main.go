package main

import (
	"VeriEQ/CodeGenerator"
	"flag"
	"fmt"
	"os"
)

func main() {
	//TestAllEquivalence()
	PrettyLogo()
	fuzzer := flag.String("fuzzer", "verilator", "Which fuzzer to run: iverilog | verilator | yosys | cxxrtl")
	threads := flag.Int("threads", 30, "Number of threads")
	count := flag.Int("count", 5, "Number of equivalent test cases")
	configPath := flag.String("config", "", "Path to config file")
	controlFlowEquiv := flag.Bool("control-flow-equiv", controlFlowEquivEnabled, "Enable control-flow equivalence transformations")
	xInputs := flag.Bool("x-input", xInputEnabled, "Enable X-valued inputs in testbench")
	flag.Parse()
	cfg, err := LoadToolConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Config load warning: %v\n", err)
	}
	toolConfig = cfg
	CodeGenerator.DefaultUsePaperInitGen = paperInitGenEnabled
	CodeGenerator.DefaultEnableControlFlowEquiv = *controlFlowEquiv
	CodeGenerator.DefaultEnableXInputs = *xInputs
	if *xInputs {
		diffSimEnabled = false
	}
	RunSelectedFuzzer(*fuzzer, *count, *threads)
}
