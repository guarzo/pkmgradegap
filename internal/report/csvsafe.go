package report

import (
	"strings"
)

// EscapeCSVCell protects against CSV formula injection attacks
// by escaping cells that start with dangerous characters
func EscapeCSVCell(value string) string {
	if value == "" {
		return value
	}

	// Check if the first character is a formula indicator
	firstChar := value[0]
	if firstChar == '=' || firstChar == '+' || firstChar == '-' || firstChar == '@' {
		// Prefix with single quote to escape formula
		return "'" + value
	}

	// Also check for other potential formula patterns
	// Some spreadsheets may interpret these as formulas
	if strings.HasPrefix(value, "|") || strings.HasPrefix(value, "%") {
		return "'" + value
	}

	// Check for tab character at the start (can be used for injection)
	if strings.HasPrefix(value, "\t") {
		return "'" + value
	}

	// Check for carriage return or newline at start
	if strings.HasPrefix(value, "\r") || strings.HasPrefix(value, "\n") {
		return "'" + value
	}

	return value
}

// EscapeCSVRow escapes all cells in a row
func EscapeCSVRow(row []string) []string {
	escaped := make([]string, len(row))
	for i, cell := range row {
		escaped[i] = EscapeCSVCell(cell)
	}
	return escaped
}

// EscapeCSVRows escapes all cells in multiple rows
func EscapeCSVRows(rows [][]string) [][]string {
	escaped := make([][]string, len(rows))
	for i, row := range rows {
		escaped[i] = EscapeCSVRow(row)
	}
	return escaped
}

// SafeCSVHeaders ensures header row is consistent and safe
func SafeCSVHeaders(headers []string) []string {
	return EscapeCSVRow(headers)
}
