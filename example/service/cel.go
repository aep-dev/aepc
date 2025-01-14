package service

// required for cel2db.

// extern char *cel_to_sql(const char *cel_expr, const char *sql_dialect);
// #cgo LDFLAGS: -L ${SRCDIR} -lcel2db
import "C"

import (
	"errors"

	"github.com/aep-dev/aepc/pkg/cel2ansisql"
	"github.com/google/cel-go/cel"
)

func convertCELToSQL(expr string) (string, error) {
	condition := C.cel_to_sql(C.CString(expr), C.CString("sqlite"))
	if condition == nil {
		return "", errors.New("failed to convert CEL to SQL")
	}
	return C.GoString(condition), nil
}

func convertCELToSQL2(expr string) (string, error) {
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
