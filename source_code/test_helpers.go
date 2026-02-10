package main

import (
	"VeriEQ/CodeGenerator"
	"fmt"
	"math/rand"
	"os"
	"time"
)

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
	g.Name = "top"

	equalNumber := 3
	modules := g.GenerateEquivalentModulesWithOneTop(equalNumber)

	if err := os.WriteFile("test.v", []byte(modules), 0644); err != nil {
		panic(err)
	}

	tb := g.GenerateEquivalenceCheckTb(equalNumber)
	if err := os.WriteFile("tb.v", []byte(tb), 0644); err != nil {
		panic(err)
	}

	input := g.GenerateInputFile()
	if err := os.WriteFile("input.txt", []byte(input), 0644); err != nil {
		panic(err)
	}

}

func TestAllEquivalence() {
	StartCounterLogger("cxxrtl_verilator_task_counter.txt")

	//go EqualFuzzIverilog(50, 10)
	go EqualFuzzVerilator(50, 10, true)
	//go EqualFuzzYosysOpt(30, 10)
	go EqualFuzzCXXRTL(50, 10, true)

	select {}
}
