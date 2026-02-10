package main

import (
	"VeriEQ/CodeGenerator"
	"fmt"
	"strings"
)

func diffLinesWithLabels(leftLabel string, leftData []byte, rightLabel string, rightData []byte) string {
	leftLines := strings.Split(string(leftData), "\n")
	rightLines := strings.Split(string(rightData), "\n")

	var sb strings.Builder

	maxLines := len(leftLines)
	if len(rightLines) > maxLines {
		maxLines = len(rightLines)
	}

	for i := 0; i < maxLines; i++ {
		var leftLine, rightLine string
		if i < len(leftLines) {
			leftLine = leftLines[i]
		}
		if i < len(rightLines) {
			rightLine = rightLines[i]
		}

		if leftLine != rightLine {
			sb.WriteString(fmt.Sprintf("Line %d:\n", i+1))
			sb.WriteString(fmt.Sprintf("  %s: %s\n", leftLabel, leftLine))
			sb.WriteString(fmt.Sprintf("  %s: %s\n", rightLabel, rightLine))
		}
	}

	return sb.String()
}

func diffLines(verilatorData, iverilogData []byte) string {
	return diffLinesWithLabels("Verilator", verilatorData, "Icarus", iverilogData)
}

func generateEq0Tb(generator *CodeGenerator.ExpressionGenerator) string {
	originalName := generator.Name
	generator.Name = fmt.Sprintf("%s_eq0", originalName)
	tbData := generator.GenerateTb()
	generator.Name = originalName
	return tbData
}
