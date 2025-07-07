# VeriEQ

Verilog compilers, including simulators and synthesizers, play a critical role in chip design and verification. However, due to the complexity of simulation and synthesis processes, they easily intro- duce various types of bugs, such as crashes, hangs, and incorrect results. Among them, Mis-Compilation Bugs (MCBs) are particu- larly severe, as they directly cause incorrect results by introducing subtle semantic deviations that make the chip behave differently from its intended design, potentially enabling hardware backdoors.

In this work, we propose VeriEQ, an automated framework based on the idea of metamorphic testing, which detects MCBs by generating semantically equivalent Verilog programs. First, to increase the likelihood of triggering MCB, we analyze the structural patterns of historical MCB and design a Verilog code template. Second, we generate semantically equivalent variants by applying circuit transformation rules. These rules include constraints on bit-width and signedness to ensure logical consistency before and after the transformation. Finally, we design an inlined differential checking mechanism that embeds multiple equivalent modules within a single testbench to improve testing efficiency. We implement and evaluate VeriEQ on four mainstream Verilog compilers. Experimental results show that VeriEQ achieves a 138.1% to 4161.9% speedup over state-of-the-art tools. In total, VeriEQ successfully detects 33 previously unknown bugs, including 21 MCBs, along with 4 hang and 8 crash bugs as additional findings. All discovered bugs have been confirmed, with 27 already fixed. In contrast, TransFuzz and Verismith only respectively detect 7 and 0 bugs.

# Repo Structure

To help users understand the repository structure of VeriEQ, we provide the following explanation:

`source_code`: the source code of VeriEQ in 4 Verilog Compilers

- **Yosys**
- **Verilator**
- **CXXRTL**
- **IVerilog**

`experiment_data`: All experimental data is located in the `experiment_data` directory.

- **evaluation1**
  - The Verilog programs that triggered bugs in the four Verilog compilers.
- **evaluation2**
  - Experimental results comparing the test case generation speed of VeriEQ, TransFuzz, and VeriSmith across four Verilog compilers.
- **evaluation3**
  - The `Efficiency` directory contains experimental results showing the test case generation speed of the four Verilog compilers with and without VeriEQ’s inlined differential checking.
  - The `Accuracy` directory contains two false positives previously found by VeriEQ.

# Quickstart



## VeriEQ for Yosys

### prerequisites

Setup Yosys environment, can be found in https://github.com/YosysHQ/yosys

### start testing

```
cd source_code
go build
./VeriEQ --fuzzer yosys --count 10 --threads 100
```



## VeriEQ for Verilator

### prerequisites

Setup Verilator environment, can be found in https://github.com/verilator/verilator

### start testing

```
cd source_code
go build
./VeriEQ --fuzzer verilator --count 10 --threads 100
```



## VeriEQ for CXXRTL

### prerequisites

Setup CXXRTL environment, can be found in https://github.com/YosysHQ/yosys

### start testing

```
cd source_code
go build
./VeriEQ --fuzzer cxxrtl --count 10 --threads 100
```



## VeriEQ for IVerilog

### prerequisites

Setup IVerilog environment, can be found in https://github.com/steveicarus/iverilog

### start testing

```
cd source_code
go build
./VeriEQ --fuzzer iverilog --count 10 --threads 100
```