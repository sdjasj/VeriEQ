package main

import (
	"fmt"
	"os"
	"sync/atomic"
	"time"
)

func RunSelectedFuzzer(name string, count, threads int) {
	PrettyRunHeader(name, count, threads)
	switch name {
	case "iverilog":
		go EqualFuzzIverilog(threads, count, diffSimEnabled)
	case "verilator":
		go EqualFuzzVerilator(threads, count, diffSimEnabled)
	case "yosys":
		go EqualFuzzYosysOpt(threads, count, diffSimEnabled)
	case "cxxrtl":
		go EqualFuzzCXXRTL(threads, count, diffSimEnabled)
	default:
		fmt.Fprintf(os.Stdout, "Unknown fuzzer: %s\n", name)
		os.Exit(1)
	}
	for {

	}
}

func EqualFuzzIverilog(workersPerType, equalNumber int, diffSim bool) {
	fuzzerIcarus := &Fuzzer{
		StartTime:     time.Now().UnixMilli(),
		EnableDiffSim: diffSim,
	}
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

func EqualFuzzVerilator(workersPerType, equalNumber int, diffSim bool) {
	fuzzerVerilator := &Fuzzer{
		StartTime:     time.Now().UnixMilli(),
		EnableDiffSim: diffSim,
	}
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

func EqualFuzzYosysOpt(workersPerType, equalNumber int, diffSim bool) {

	fuzzerYosysOpt := &Fuzzer{
		StartTime:     time.Now().UnixMilli(),
		EnableDiffSim: diffSim,
	}
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

func EqualFuzzCXXRTL(workersPerType, equalNumber int, diffSim bool) {
	fuzzerCXXRTL := &Fuzzer{
		StartTime:     time.Now().UnixMilli(),
		EnableDiffSim: diffSim,
	}
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
