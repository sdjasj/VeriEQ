package main

import (
	"fmt"
	"os"
	"sync/atomic"
	"time"
)

var countIverilog int64 = 0
var countVerilator int64 = 0
var countYosysOpt int64 = 0
var countCXXRTL int64 = 0

var outputFile = "task_counter.txt"

func StartCounterLogger(outputFile string) {
	startTimestamp := time.Now().Unix() // 起始时间戳（秒）

	go func() {
		for {
			time.Sleep(30 * time.Second)

			currentTime := time.Now().Format("2006-01-02 15:04:05")
			currentTimestamp := time.Now().Unix()
			elapsed := currentTimestamp - startTimestamp

			iverilog := atomic.LoadInt64(&countIverilog)
			verilator := atomic.LoadInt64(&countVerilator)
			yosys := atomic.LoadInt64(&countYosysOpt)
			cxxrtl := atomic.LoadInt64(&countCXXRTL)

			line := fmt.Sprintf("[%s] Icarus=%d Verilator=%d YosysOpt=%d CXXRTL=%d Elapsed=%ds\n",
				currentTime, iverilog, verilator, yosys, cxxrtl, elapsed)

			f, err := os.OpenFile(outputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err == nil {
				defer f.Close()
				f.WriteString(line)
			}
		}
	}()
}
