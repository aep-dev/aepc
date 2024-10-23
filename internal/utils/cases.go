package utils

import (
	"strings"
)

func KebabToCamelCase(s string) string {
	parts := strings.Split(s, "-")
	for i := range parts {
		if len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(string(parts[i][0])) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}

func KebabToSnakeCase(s string) string {
	return strings.ReplaceAll(s, "-", "_")
}
