package service

import (
	"github.com/aep-dev/aepc/pkg/cel2ansisql"
	"github.com/google/cel-go/cel"
)

func convertCELToSQL(expr string) (string, error) {
	if expr == "" {
		return "", nil
	}
	env, err := cel.NewEnv(
		cel.Variable("path", cel.StringType),
		cel.Variable("description", cel.StringType),
	)
	if err != nil {
		return "", err
	}
	ast, iss := env.Compile(expr)
	if iss.Err() != nil {
		return "", iss.Err()
	}

	sql, err := cel2ansisql.ConvertToSQL(ast)
	if err != nil {
		return "", err
	}
	return sql, nil
}
