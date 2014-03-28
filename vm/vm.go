package vm

import (
	"errors"
	"fmt"
	"strconv"
	"github.com/mattn/ungo/ast"
	"reflect"
	"strings"
)

type Env map[string]reflect.Value

var NilValue = reflect.ValueOf(nil)

type Func func(args ...reflect.Value) (reflect.Value, error)

func ToFunc(f Func) reflect.Value {
	return reflect.ValueOf(f)
}

func Run(stmt ast.Stmt, env Env) (reflect.Value, error) {
	switch stmt := stmt.(type) {
	case *ast.ExprStmt:
		v, err := invokeExpr(stmt.Expr, env)
		if err != nil {
			return NilValue, err
		}
		return v, nil
	case *ast.VarStmt:
		v, err := invokeExpr(stmt.Expr, env)
		if err != nil {
			return NilValue, err
		}
		env[stmt.Name] = v
		return v, nil
	case *ast.ReturnStmt:
		v, err := invokeExpr(stmt.Expr, env)
		if err != nil {
			return NilValue, err
		}
		return v, nil
	case *ast.FuncStmt:
		f := reflect.ValueOf(func(stmt *ast.FuncStmt, env Env) Func {
			newenv := make(Env)
			for k, v := range env {
				newenv[k] = v
			}
			return func(args ...reflect.Value) (reflect.Value, error) {
				v := NilValue
				var err error
				if len(args) != len(stmt.Args) {
					return NilValue, errors.New("Arguments Number of mismatch")
				}
				for i, arg := range stmt.Args {
					newenv[arg] = args[i]
				}
				for _, stmt := range stmt.Stmts {
					v, err = Run(stmt, newenv)
					if err != nil {
						return NilValue, err
					}
					if _, ok := stmt.(*ast.ReturnStmt); ok {
						return v, nil
					}
				}
				return v, nil
			}
		}(stmt, env))
		env[stmt.Name] = f
		return f, nil
	default:
		return NilValue, errors.New("Unknown Statement type")
	}
}

func toFloat64(v reflect.Value) float64 {
	switch v.Kind() {
	case reflect.Float32, reflect.Float64:
		return v.Float()
	case reflect.Int, reflect.Int32, reflect.Int64:
		return float64(v.Int())
	}
	return 0.0
}

func toInt64(v reflect.Value) int64 {
	switch v.Kind() {
	case reflect.Float32, reflect.Float64:
		return int64(v.Float())
	case reflect.Int, reflect.Int32, reflect.Int64:
		return v.Int()
	}
	return 0
}

func invokeExpr(expr ast.Expr, env Env) (reflect.Value, error) {
	switch e := expr.(type) {
	case *ast.NumberExpr:
		if strings.Contains(e.Lit, ".") {
			v, err := strconv.ParseFloat(e.Lit, 64)
			if err != nil {
				return NilValue, err
			}
			return reflect.ValueOf(float64(v)), nil
		}
		v, err := strconv.Atoi(e.Lit)
		if err != nil {
			return NilValue, err
		}
		return reflect.ValueOf(int64(v)), nil
	case *ast.IdentExpr:
		if v, ok := env[e.Lit]; ok {
			return v, nil
		} else {
			return NilValue, fmt.Errorf("Undefined variable: %s", e.Lit)
		}
	case *ast.StringExpr:
		return reflect.ValueOf(e.Lit), nil
	case *ast.UnaryMinusExpr:
		v, err := invokeExpr(e.SubExpr, env)
		if err != nil {
			return NilValue, err
		}
		if v.Kind() == reflect.Float64 {
			return reflect.ValueOf(-v.Float()), nil
		}
		return reflect.ValueOf(-v.Int()), nil
	case *ast.ParenExpr:
		v, err := invokeExpr(e.SubExpr, env)
		if err != nil {
			return NilValue, err
		}
		return v, nil
	case *ast.BinOpExpr:
		lhsV, err := invokeExpr(e.Lhs, env)
		if err != nil {
			return NilValue, err
		}
		rhsV, err := invokeExpr(e.Rhs, env)
		if err != nil {
			return NilValue, err
		}
		switch e.Operator {
		case '+':
			if lhsV.Kind() == reflect.Float64 || rhsV.Kind() == reflect.Float64 {
				return reflect.ValueOf(toFloat64(lhsV) + toFloat64(rhsV)), nil
			}
			return reflect.ValueOf(toInt64(lhsV) + toInt64(rhsV)), nil
		case '-':
			if lhsV.Kind() == reflect.Float64 || rhsV.Kind() == reflect.Float64 {
				return reflect.ValueOf(toFloat64(lhsV) - toFloat64(rhsV)), nil
			}
			return reflect.ValueOf(toInt64(lhsV) - toInt64(rhsV)), nil
		case '*':
			if lhsV.Kind() == reflect.Float64 || rhsV.Kind() == reflect.Float64 {
				return reflect.ValueOf(toFloat64(lhsV) * toFloat64(rhsV)), nil
			}
			return reflect.ValueOf(toInt64(lhsV) * toInt64(rhsV)), nil
		case '/':
			return reflect.ValueOf(toFloat64(lhsV) / toFloat64(rhsV)), nil
		case '%':
			return reflect.ValueOf(toInt64(lhsV) % toInt64(rhsV)), nil
		default:
			return NilValue, errors.New("Unknown operator")
		}
	case *ast.CallExpr:
		f := env[e.Name]
		args := make([]reflect.Value, f.Type().NumIn())
		for i, expr := range e.SubExprs {
			arg, err := invokeExpr(expr, env)
			if err != nil {
				return NilValue, err
			}
			args[i] = reflect.ValueOf(arg)
		}
		ret := f.Call(args)
		err := ret[1].Interface()
		if err != nil {
			return NilValue, err.(error)
		}
		return ret[0].Interface().(reflect.Value), nil
	default:
		return NilValue, errors.New("Unknown Expression type")
	}
}
