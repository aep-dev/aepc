package cel2ansisql

import (
	"testing"

	"github.com/google/cel-go/cel"
)

func TestCELToSQL(t *testing.T) {
	env, err := cel.NewEnv(
		cel.Variable("path", cel.StringType),
		cel.Variable("description", cel.StringType),
	)
	if err != nil {
		t.Fatalf("failed to create CEL environment: %v", err)
	}

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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ast, iss := env.Compile(tt.input)
			if iss.Err() != nil {
				t.Errorf("compile() = %v, want %v", iss.Err(), tt.expected)
			}
			got, err := ConvertToSQL(ast)
			if err != nil {
				t.Errorf("ConvertToSQL() = %v, want %v", err, tt.expected)
			}
			if got != tt.expected {
				t.Errorf("ConvertToSQL() = %v, want %v", got, tt.expected)
			}
		})
	}
}
