package main

import (
	"fmt"
	"math"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	ansiReset   = "\x1b[0m"
	ansiBold    = "\x1b[1m"
	ansiDim     = "\x1b[2m"
	ansiRed     = "\x1b[31m"
	ansiGreen   = "\x1b[32m"
	ansiYellow  = "\x1b[33m"
	ansiBlue    = "\x1b[34m"
	ansiMagenta = "\x1b[35m"
	ansiCyan    = "\x1b[36m"
)

var (
	prettyEnabled = detectPrettyEnabled()
	prettyMu      sync.Mutex
)

func detectPrettyEnabled() bool {
	if os.Getenv("FUZZ_PLAIN") != "" {
		return false
	}
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	term := os.Getenv("TERM")
	if term == "" || term == "dumb" {
		return false
	}
	return true
}

func colorize(style, s string) string {
	if !prettyEnabled {
		return s
	}
	return style + s + ansiReset
}

func dim(s string) string {
	return colorize(ansiDim, s)
}

func formatScope(scope string) string {
	if scope == "" {
		return "MAIN"
	}
	return strings.ToUpper(scope)
}

func prettyLine(level, color, scope, msg string) {
	prettyMu.Lock()
	defer prettyMu.Unlock()

	levelTag := fmt.Sprintf("[%s]", level)
	scopeTag := fmt.Sprintf("(%s)", formatScope(scope))
	if prettyEnabled {
		levelTag = colorize(ansiBold+color, levelTag)
		scopeTag = colorize(color, scopeTag)
	}
	fmt.Printf("%s %s %s\n", levelTag, scopeTag, msg)
}

func prettyBlock(level, color, title, body string) {
	prettyMu.Lock()
	defer prettyMu.Unlock()

	border := strings.Repeat("=", 72)
	if prettyEnabled {
		border = colorize(ansiBold+color, border)
	}
	fmt.Println(border)
	levelTag := fmt.Sprintf("[%s]", level)
	if prettyEnabled {
		levelTag = colorize(ansiBold+color, levelTag)
	}
	fmt.Printf("%s %s\n", levelTag, title)
	if strings.TrimSpace(body) != "" {
		fmt.Println(body)
	}
	fmt.Println(border)
}

func PrettyLogo() {
	lines := strings.Split(logo, "\n")
	palette := []string{ansiCyan, ansiBlue, ansiMagenta, ansiGreen, ansiYellow}
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			fmt.Println(line)
			continue
		}
		color := palette[i%len(palette)]
		if prettyEnabled {
			fmt.Println(colorize(ansiBold+color, line))
			continue
		}
		fmt.Println(line)
	}
}

func PrettyRunHeader(name string, count, threads int) {
	prettyMu.Lock()
	defer prettyMu.Unlock()

	title := fmt.Sprintf("FUZZ MODE: %s", strings.ToUpper(name))
	border := strings.Repeat("=", len(title)+10)
	if prettyEnabled {
		border = colorize(ansiBold+ansiBlue, border)
	}
	fmt.Println(border)
	line := fmt.Sprintf("== %s ==", title)
	if prettyEnabled {
		line = colorize(ansiBold+ansiBlue, line)
	}
	fmt.Println(line)
	detail := fmt.Sprintf("threads=%d  batch=%d", threads, count)
	if prettyEnabled {
		detail = colorize(ansiDim, detail)
	}
	fmt.Println(detail)
	fmt.Println(border)
}

func PrettyOK(scope, msg string) {
	prettyLine("OK", ansiGreen, scope, msg)
}

func PrettyInfo(scope, msg string) {
	prettyLine("INFO", ansiCyan, scope, msg)
}

func PrettyWarn(scope, msg string) {
	prettyLine("WARN", ansiYellow, scope, msg)
}

func PrettyErr(scope, msg string) {
	prettyLine("ERR", ansiRed, scope, msg)
}

func PrettyBug(scope, msg string, details ...string) {
	body := strings.Join(details, "\n")
	prettyBlock("BUG", ansiRed, formatScope(scope)+": "+msg, body)
}

func StartPrettyTicker(label string, interval time.Duration) {
	if interval <= 0 {
		interval = 10 * time.Second
	}
	go func() {
		start := time.Now()
		prevTotal := int64(0)
		frames := []string{"-", "\\", "|", "/"}
		frame := 0
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			iverilog := atomic.LoadInt64(&countIverilog)
			verilator := atomic.LoadInt64(&countVerilator)
			yosys := atomic.LoadInt64(&countYosysOpt)
			cxxrtl := atomic.LoadInt64(&countCXXRTL)
			total := iverilog + verilator + yosys + cxxrtl
			delta := total - prevTotal
			prevTotal = total
			rate := float64(delta) / interval.Seconds()
			elapsed := time.Since(start).Truncate(time.Second)
			bar := rateBar(rate, 22)
			spinner := frames[frame%len(frames)]
			frame++

			msg := fmt.Sprintf("%s fuzz=%s | Icarus=%d Verilator=%d YosysOpt=%d CXXRTL=%d | +%d/%s (%.1f/s) | %s %s",
				spinner,
				strings.ToUpper(label),
				iverilog,
				verilator,
				yosys,
				cxxrtl,
				delta,
				interval,
				rate,
				elapsed,
				bar,
			)
			prettyLine("STAT", ansiBlue, "fuzz", msg)
		}
	}()
}

func rateBar(rate float64, width int) string {
	if width <= 0 {
		width = 16
	}
	scaled := int(math.Round(math.Log10(rate+1) * 6))
	if scaled < 0 {
		scaled = 0
	}
	if scaled > width {
		scaled = width
	}
	filled := strings.Repeat("#", scaled)
	empty := strings.Repeat(".", width-scaled)
	bar := fmt.Sprintf("[%s%s]", filled, empty)
	if prettyEnabled {
		return colorize(ansiGreen, bar)
	}
	return bar
}
