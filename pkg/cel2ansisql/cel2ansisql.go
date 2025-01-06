package cel2ansisql

import (
	"fmt"
	"strings"

	"github.com/google/cel-go/cel"
	exprpb "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

// ConvertToSQL converts a CEL AST to ANSI SQL
func ConvertToSQL(a *cel.Ast) (string, error) {
	checkedExpr, err := cel.AstToCheckedExpr(a)
	if err != nil {
		return "", err
	}
	expr := checkedExpr.Expr
	return convertExpr(expr)
}

func convertExpr(expr *exprpb.Expr) (string, error) {
	switch expr.ExprKind.(type) {
	case *exprpb.Expr_CallExpr:
		return convertCall(expr.GetCallExpr())
	case *exprpb.Expr_IdentExpr:
		return handleIdentExpr(expr.GetIdentExpr())
	case *exprpb.Expr_ConstExpr:
		return handleConstExpr(expr.GetConstExpr())
	default:
		return "", fmt.Errorf("unsupported expression type: %T", expr)
	}
}

func convertCall(call *exprpb.Expr_Call) (string, error) {
	if call.Target != nil {
		return convertCallWithTarget(call.Function, call.Target, call.Args)
	}
	// Handle unary operators
	if len(call.Args) == 1 {
		return handleUnaryOp(call.Function, call.Args[0])
	}

	if len(call.Args) == 2 {
		return handleBinaryOp(call.Function, call.Args[0], call.Args[1])
	}
	return "", fmt.Errorf("unsupported call expression: %v", call)
}

func convertCallWithTarget(function string, target *exprpb.Expr, args []*exprpb.Expr) (string, error) {
	targetSQL, err := convertExpr(target)
	if err != nil {
		return "", err
	}
	if len(args) == 1 {
		argSQL, err := convertExpr(args[0])
		if err != nil {
			return "", err
		}
		switch function {
		case "startsWith":
			return fmt.Sprintf("%s LIKE CONCAT(%s, '%%')", targetSQL, argSQL), nil
		case "contains":
			return fmt.Sprintf("%s LIKE CONCAT('%%', %s, '%%')", targetSQL, argSQL), nil
		case "endsWith":
			return fmt.Sprintf("%s LIKE CONCAT('%%', %s)", targetSQL, argSQL), nil
		}
	}
	return "", fmt.Errorf("unsupported call expression with target: (%v, %v, %v)", function, target, args)
}

func handleIdentExpr(ident *exprpb.Expr_Ident) (string, error) {
	return ident.Name, nil
}

func handleUnaryOp(function string, argument *exprpb.Expr) (string, error) {
	arg, err := convertExpr(argument)
	if err != nil {
		return "", err
	}

	switch function {
	case "!":
		return fmt.Sprintf("NOT (%s)", arg), nil
	case "-":
		return fmt.Sprintf("-%s", arg), nil
	case "size":
		return fmt.Sprintf("LENGTH(%s)", arg), nil
	case "type":
		return "", fmt.Errorf("type checking not supported in SQL conversion")
	default:
		return "", fmt.Errorf("unsupported unary operator: %s", function)
	}
}

func handleBinaryOp(function string, left *exprpb.Expr, right *exprpb.Expr) (string, error) {
	leftSQL, err := convertExpr(left)
	if err != nil {
		return "", err
	}
	rightSQL, err := convertExpr(right)
	if err != nil {
		return "", err
	}
	switch function {
	case "==":
		return fmt.Sprintf("%s = %s", leftSQL, rightSQL), nil
	case "!=":
		return fmt.Sprintf("%s <> %s", leftSQL, rightSQL), nil
	case "<":
		return fmt.Sprintf("%s < %s", leftSQL, rightSQL), nil
	case "<=":
		return fmt.Sprintf("%s <= %s", leftSQL, rightSQL), nil
	case ">":
		return fmt.Sprintf("%s > %s", leftSQL, rightSQL), nil
	case ">=":
		return fmt.Sprintf("%s >= %s", leftSQL, rightSQL), nil
	case "&&":
		return fmt.Sprintf("%s AND %s", leftSQL, rightSQL), nil
	case "||":
		return fmt.Sprintf("%s OR %s", leftSQL, rightSQL), nil
	case "+":
		// TODO: handle string concatenation
		return fmt.Sprintf("%s + %s", leftSQL, rightSQL), nil
	case "-":
		return fmt.Sprintf("%s - %s", leftSQL, rightSQL), nil
	case "*":
		return fmt.Sprintf("%s * %s", leftSQL, rightSQL), nil
	case "/":
		return fmt.Sprintf("%s / %s", leftSQL, rightSQL), nil
	case "%":
		return fmt.Sprintf("%s %s %s", leftSQL, function, rightSQL), nil
	// alphabetical
	case "in":
	case "has":
		return fmt.Sprintf("%s IN %s", leftSQL, rightSQL), nil
	case "matches":
		return fmt.Sprintf("%s REGEXP %s", leftSQL, rightSQL), nil
	}
	return "", fmt.Errorf("unsupported binary operator: %s", function)
}

func handleConstExpr(c *exprpb.Constant) (string, error) {
	switch c.ConstantKind.(type) {
	case *exprpb.Constant_NullValue:
		return "NULL", nil
	case *exprpb.Constant_StringValue:
		return fmt.Sprintf("'%s'", strings.ReplaceAll(c.GetStringValue(), "'", "''")), nil
	case *exprpb.Constant_BoolValue:
		return fmt.Sprintf("%t", c.GetBoolValue()), nil
	case *exprpb.Constant_Int64Value:
		return fmt.Sprintf("%d", c.GetInt64Value()), nil
	case *exprpb.Constant_DoubleValue:
		return fmt.Sprintf("%f", c.GetDoubleValue()), nil
	default:
		return "", fmt.Errorf("unsupported constant type: %T", c.ConstantKind)
	}
}
