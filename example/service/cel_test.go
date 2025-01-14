package service

import (
	"testing"
)

func TestCELToSQL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple expression",
			input:    "description.startsWith('tomorrow')",
			expected: "description LIKE CONCAT('tomorrow', '%')",
		},
		{
			name:     "arithemtic",
			input:    "1 + 2",
			expected: "1 + 2",
		},
		{
			name:     "empty",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := convertCELToSQL(tt.input)
			if err != nil {
				t.Errorf("convertCELToSQL() = %v, want %v", err, tt.expected)
			}
			if got != tt.expected {
				t.Errorf("convertCELToSQL() = %v, want %v", got, tt.expected)
			}
		})
	}
}
