package report

import (
	"reflect"
	"testing"
)

func TestEscapeCSVCell(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Safe values - should not be escaped
		{"empty", "", ""},
		{"normal_text", "Charizard", "Charizard"},
		{"number", "123.45", "123.45"},
		{"safe_special", "#001", "#001"},
		{"internal_equal", "A=B", "A=B"},

		// Formula injections - must be escaped
		{"formula_equal", "=SUM(A1:A10)", "'=SUM(A1:A10)"},
		{"formula_plus", "+123", "'+123"},
		{"formula_minus", "-123", "'-123"},
		{"formula_at", "@SUM(A:A)", "'@SUM(A:A)"},
		{"formula_pipe", "|echo test", "'|echo test"},
		{"formula_percent", "%PATH%", "'%PATH%"},

		// Whitespace injections
		{"tab_start", "\t=EXEC()", "'\t=EXEC()"},
		{"newline_start", "\n=FORMULA()", "'\n=FORMULA()"},
		{"carriage_return", "\r=DATA()", "'\r=DATA()"},

		// Real card names that might trigger false positives
		{"card_negative", "-2 Pikachu", "'-2 Pikachu"},
		{"card_at_symbol", "@card_name", "'@card_name"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EscapeCSVCell(tt.input)
			if result != tt.expected {
				t.Errorf("EscapeCSVCell(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestEscapeCSVRow(t *testing.T) {
	input := []string{
		"Charizard",
		"=SUM(A1:A10)",
		"100.50",
		"-50",
		"@malicious",
		"Normal Text",
	}

	expected := []string{
		"Charizard",
		"'=SUM(A1:A10)",
		"100.50",
		"'-50",
		"'@malicious",
		"Normal Text",
	}

	result := EscapeCSVRow(input)

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("EscapeCSVRow() failed")
		for i := range result {
			if result[i] != expected[i] {
				t.Errorf("  Index %d: got %q, want %q", i, result[i], expected[i])
			}
		}
	}
}

func TestEscapeCSVRows(t *testing.T) {
	input := [][]string{
		{"Header1", "=FORMULA", "Header3"},
		{"Data1", "+123", "-456"},
		{"Normal", "Text", "Here"},
	}

	expected := [][]string{
		{"Header1", "'=FORMULA", "Header3"},
		{"Data1", "'+123", "'-456"},
		{"Normal", "Text", "Here"},
	}

	result := EscapeCSVRows(input)

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("EscapeCSVRows() failed")
		for i := range result {
			for j := range result[i] {
				if result[i][j] != expected[i][j] {
					t.Errorf("  [%d][%d]: got %q, want %q", i, j, result[i][j], expected[i][j])
				}
			}
		}
	}
}

func TestSafeCSVHeaders(t *testing.T) {
	input := []string{
		"Card",
		"=IMPORTXML()",
		"Price",
		"@Rarity",
	}

	expected := []string{
		"Card",
		"'=IMPORTXML()",
		"Price",
		"'@Rarity",
	}

	result := SafeCSVHeaders(input)

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("SafeCSVHeaders() = %v, want %v", result, expected)
	}
}

func BenchmarkEscapeCSVCell(b *testing.B) {
	testCases := []string{
		"Normal text",
		"=SUM(A1:A10)",
		"123.45",
		"-negative",
		"@formula",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = EscapeCSVCell(testCases[i%len(testCases)])
	}
}

func BenchmarkEscapeCSVRow(b *testing.B) {
	row := []string{
		"Charizard",
		"001",
		"100.50",
		"=FORMULA()",
		"Rare",
		"tcgplayer.market",
		"+50.00",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = EscapeCSVRow(row)
	}
}
