package main

type Fuzzer struct {
	StartTime        int64
	LogDir           string
	TmpDir           string
	CrashDir         string
	TestBenchName    string
	TestFileName     string
	VerilatorOptions []string
	EnableDiffSim    bool
}
