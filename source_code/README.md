# VeriEQ Run Guide

## Prerequisites
- Go 1.21+ (see `go.mod`).
- The four platform binaries under `../../target/binary`:
  - `iverilog/bin/iverilog`
  - `verilator/bin/verilator`
  - `yosys/bin/yosys`
  - `cxxrtl/bin/yosys-config`
- `clang++` available in `PATH` for the CXXRTL flow.

## Configuration
The tool reads `config.json` by default.
- `binary_root` is already set to `../../target/binary`.
- Leave per-tool paths empty to auto-fill from `binary_root`.
- If your binaries are elsewhere, edit `config.json` or pass `-config`.

## Run (manual)
From the repo root (`/root/artifact-evaluation/code/source_code`):

```bash
# Icarus Verilog
GOCACHE=.gocache go run . -fuzzer iverilog -threads 64 -count 5

# Verilator
GOCACHE=.gocache go run . -fuzzer verilator -threads 64 -count 5

# Yosys (opt flow)
GOCACHE=.gocache go run . -fuzzer yosys -threads 64 -count 5

# CXXRTL
GOCACHE=.gocache go run . -fuzzer cxxrtl -threads 64 -count 5
```

## Run (scripts)
Scripts live in `scripts/` and accept optional arguments:

```bash
scripts/run_iverilog.sh [threads] [count] [config_path]
scripts/run_verilator.sh [threads] [count] [config_path]
scripts/run_yosys.sh [threads] [count] [config_path]
scripts/run_cxxrtl.sh [threads] [count] [config_path]
```

Example:

```bash
scripts/run_iverilog.sh 16 20
```

## Output directories
Each run creates subdirectories under:
- `tmp/` for working files
- `log/` for logs
- `crash/` for saved failing cases

They are timestamped by run start time.

## Notes
- If Go build cache permission errors occur, keep using `GOCACHE=.gocache` as shown above.
- The CXXRTL flow uses `yosys-config` to locate the runtime headers; ensure it matches the Yosys binary.
