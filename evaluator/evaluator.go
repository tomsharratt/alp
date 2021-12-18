package evaluator

import (
	"context"
	"fmt"

	"github.com/tomsharratt/alp/ast"
	"github.com/tomsharratt/alp/object"
)

var (
	NULL  = &object.Null{}
	TRUE  = &object.Boolean{Value: true}
	FALSE = &object.Boolean{Value: false}
)

func Eval(
	ctx context.Context,
	node ast.Node,
	env *object.Environment,
) (object.Object, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	switch node := node.(type) {
	case *ast.Program:
		return evalProgram(ctx, node.Statements, env)
	case *ast.BlockStatement:
		return evalBlockStatement(ctx, node, env)
	case *ast.ReturnStatement:
		val, err := Eval(ctx, node.ReturnValue, env)
		if err != nil {
			return nil, err
		}
		if isError(val) {
			return val, nil
		}
		return &object.ReturnValue{Value: val}, nil
	case *ast.LetStatement:
		val, err := Eval(ctx, node.Value, env)
		if err != nil {
			return nil, err
		}
		if isError(val) {
			return val, nil
		}
		env.Set(node.Name.Value, val)
	case *ast.Identifier:
		return evalIdentifier(node, env), nil
	case *ast.ExpressionStatement:
		return Eval(ctx, node.Expression, env)
	case *ast.FunctionLiteral:
		params := node.Parameters
		body := node.Body
		return &object.Function{Parameters: params, Env: env, Body: body}, nil
	case *ast.IntegerLiteral:
		return &object.Integer{Value: node.Value}, nil
	case *ast.StringLiteral:
		return &object.String{Value: node.Value}, nil
	case *ast.Boolean:
		return nativeBoolToBooleanObject(node.Value), nil
	case *ast.ArrayLiteral:
		elements, err := evalExpressions(ctx, node.Elements, env)
		if err != nil {
			return nil, err
		}
		if len(elements) == 1 && isError(elements[0]) {
			return elements[0], nil
		}
		return &object.Array{Elements: elements}, nil
	case *ast.HashLiteral:
		return evalHashLiteral(ctx, node, env)
	case *ast.IndexExpression:
		left, err := Eval(ctx, node.Left, env)
		if err != nil {
			return nil, err
		}
		if isError(left) {
			return left, nil
		}
		index, err := Eval(ctx, node.Index, env)
		if err != nil {
			return nil, err
		}
		if isError(index) {
			return index, nil
		}
		return evalIndexExpression(left, index), nil
	case *ast.PrefixExpression:
		right, err := Eval(ctx, node.Right, env)
		if err != nil {
			return nil, err
		}
		if isError(right) {
			return right, nil
		}
		return evalPrefixExpression(node.Operator, right), nil
	case *ast.InfixExpression:
		left, err := Eval(ctx, node.Left, env)
		if err != nil {
			return nil, err
		}
		if isError(left) {
			return left, nil
		}

		right, err := Eval(ctx, node.Right, env)
		if err != nil {
			return nil, err
		}
		if isError(right) {
			return right, nil
		}
		return evalInfixExpression(node.Operator, left, right), nil
	case *ast.IfExpression:
		return evalIfExpression(ctx, node, env)
	case *ast.CallExpression:
		function, err := Eval(ctx, node.Function, env)
		if err != nil {
			return nil, err
		}
		if isError(function) {
			return function, nil
		}
		args, err := evalExpressions(ctx, node.Arguments, env)
		if err != nil {
			return nil, err
		}
		if len(args) == 1 && isError(args[0]) {
			return args[0], nil
		}
		return applyFunction(ctx, function, args)
	}

	return nil, nil
}

func evalProgram(
	ctx context.Context,
	statements []ast.Statement,
	env *object.Environment,
) (object.Object, error) {
	var result object.Object
	var err error

	for _, statement := range statements {
		result, err = Eval(ctx, statement, env)
		if err != nil {
			return nil, err
		}

		switch result := result.(type) {
		case *object.ReturnValue:
			return result.Value, nil
		case *object.Error:
			return result, nil
		}
	}

	return result, nil
}

func evalBlockStatement(
	ctx context.Context,
	block *ast.BlockStatement,
	env *object.Environment,
) (object.Object, error) {
	var result object.Object
	var err error

	for _, statement := range block.Statements {
		result, err = Eval(ctx, statement, env)
		if err != nil {
			return nil, err
		}

		if result != nil {
			rt := result.Type()
			if rt == object.RETURN_VALUE_OBJ || rt == object.ERROR_OBJ {
				return result, nil
			}
		}
	}

	return result, nil
}

func evalIdentifier(
	node *ast.Identifier,
	env *object.Environment,
) object.Object {
	if val, ok := env.Get(node.Value); ok {
		return val
	}

	if builtin, ok := builtins[node.Value]; ok {
		return builtin
	}

	return newError("identifier not found: " + node.Value)
}

func evalPrefixExpression(operator string, right object.Object) object.Object {
	switch operator {
	case "!":
		return evalBangOperatorExpression(right)
	case "-":
		return evalMinusPrefixOperatorExpression(right)
	default:
		return newError("unknown operator: %s%s", operator, right.Type())
	}
}

func evalBangOperatorExpression(right object.Object) object.Object {
	switch right {
	case TRUE:
		return FALSE
	case FALSE:
		return TRUE
	case NULL:
		return TRUE
	default:
		return FALSE
	}
}

func evalMinusPrefixOperatorExpression(right object.Object) object.Object {
	if right.Type() != object.INTEGER_OBJ {
		return newError("unknown operator: -%s", right.Type())
	}

	value := right.(*object.Integer).Value
	return &object.Integer{Value: -value}
}

func evalInfixExpression(
	operator string,
	left, right object.Object,
) object.Object {
	switch {
	case left.Type() == object.INTEGER_OBJ && right.Type() == object.INTEGER_OBJ:
		return evalIntegerInfixExpression(operator, left, right)
	case operator == "==":
		return nativeBoolToBooleanObject(left == right)
	case operator == "!=":
		return nativeBoolToBooleanObject(left != right)
	case left.Type() == object.STRING_OBJ && right.Type() == object.STRING_OBJ:
		return evalStringInfixExpression(operator, left, right)
	case left.Type() != right.Type():
		return newError("type mismatch: %s %s %s",
			left.Type(), operator, right.Type())
	default:
		return newError("unknown operator: %s %s %s",
			left.Type(), operator, right.Type())
	}
}

func evalIntegerInfixExpression(
	operator string,
	left, right object.Object,
) object.Object {
	leftVal := left.(*object.Integer).Value
	rightVal := right.(*object.Integer).Value

	switch operator {
	case "+":
		return &object.Integer{Value: leftVal + rightVal}
	case "-":
		return &object.Integer{Value: leftVal - rightVal}
	case "*":
		return &object.Integer{Value: leftVal * rightVal}
	case "/":
		return &object.Integer{Value: leftVal / rightVal}
	case "<":
		return nativeBoolToBooleanObject(leftVal < rightVal)
	case ">":
		return nativeBoolToBooleanObject(leftVal > rightVal)
	case "==":
		return nativeBoolToBooleanObject(leftVal == rightVal)
	case "!=":
		return nativeBoolToBooleanObject(leftVal != rightVal)
	default:
		return newError("unknown operator: %s %s %s",
			left.Type(), operator, right.Type())
	}
}

func evalStringInfixExpression(
	operator string,
	left, right object.Object,
) object.Object {
	if operator != "+" {
		return newError("unknown operator: %s %s %s",
			left.Type(), operator, right.Type())
	}

	leftVal := left.(*object.String).Value
	rightVal := right.(*object.String).Value
	return &object.String{Value: leftVal + rightVal}
}

func evalIfExpression(
	ctx context.Context,
	ie *ast.IfExpression,
	env *object.Environment,
) (object.Object, error) {
	condition, err := Eval(ctx, ie.Condition, env)
	if err != nil {
		return nil, err
	}
	if isError(condition) {
		return condition, nil
	}

	if isTruthy(condition) {
		return Eval(ctx, ie.Consequence, env)
	} else if ie.Alternative != nil {
		return Eval(ctx, ie.Alternative, env)
	} else {
		return NULL, nil
	}
}

func evalIndexExpression(left, index object.Object) object.Object {
	switch {
	case left.Type() == object.ARRAY_OBJ && index.Type() == object.INTEGER_OBJ:
		return evalArrayIndexExpression(left, index)
	case left.Type() == object.HASH_OBJ:
		return evalHashIndexExpression(left, index)
	default:
		return newError("index operator not supported: %s", left.Type())
	}
}

func evalArrayIndexExpression(array, index object.Object) object.Object {
	arrayObject := array.(*object.Array)
	idx := index.(*object.Integer).Value
	max := int64(len(arrayObject.Elements) - 1)

	if idx < 0 || idx > max {
		return NULL
	}

	return arrayObject.Elements[idx]
}

func evalHashIndexExpression(hash, index object.Object) object.Object {
	hashObject := hash.(*object.Hash)

	key, ok := index.(object.Hashable)
	if !ok {
		return newError("unusable as hash key: %s", index.Type())
	}

	pair, ok := hashObject.Pairs[key.HashKey()]
	if !ok {
		return NULL
	}

	return pair.Value
}

func evalExpressions(
	ctx context.Context,
	exps []ast.Expression,
	env *object.Environment,
) ([]object.Object, error) {
	var result []object.Object

	for _, e := range exps {
		evaluated, err := Eval(ctx, e, env)
		if err != nil {
			return nil, err
		}
		if isError(evaluated) {
			return []object.Object{evaluated}, nil
		}
		result = append(result, evaluated)
	}

	return result, nil
}

func evalHashLiteral(
	ctx context.Context,
	node *ast.HashLiteral,
	env *object.Environment,
) (object.Object, error) {
	pairs := make(map[object.HashKey]object.HashPair)

	for keyNode, valueNode := range node.Pairs {
		key, err := Eval(ctx, keyNode, env)
		if err != nil {
			return nil, err
		}
		if isError(key) {
			return key, nil
		}

		hashKey, ok := key.(object.Hashable)
		if !ok {
			return newError("unusable as hash key: %s", key.Type()), nil
		}

		value, err := Eval(ctx, valueNode, env)
		if err != nil {
			return nil, err
		}
		if isError(value) {
			return value, nil
		}

		hashed := hashKey.HashKey()
		pairs[hashed] = object.HashPair{Key: key, Value: value}
	}

	return &object.Hash{Pairs: pairs}, nil
}

func applyFunction(
	ctx context.Context,
	fn object.Object,
	args []object.Object,
) (object.Object, error) {
	switch fn := fn.(type) {

	case *object.Function:
		extendedEnv := extendFunctionEnv(fn, args)
		evaluated, err := Eval(ctx, fn.Body, extendedEnv)
		if err != nil {
			return nil, err
		}
		return unwrapReturnValue(evaluated), nil

	case *object.Builtin:
		return fn.Fn(args...), nil

	default:
		return newError("not a function: %s", fn.Type()), nil
	}
}

func extendFunctionEnv(
	fn *object.Function,
	args []object.Object,
) *object.Environment {
	env := object.NewEnclosedEnvironment(fn.Env)

	for paramIdx, param := range fn.Parameters {
		env.Set(param.Value, args[paramIdx])
	}

	return env
}

func unwrapReturnValue(obj object.Object) object.Object {
	if returnValue, ok := obj.(*object.ReturnValue); ok {
		return returnValue.Value
	}

	return obj
}

func isTruthy(obj object.Object) bool {
	switch obj {
	case NULL:
		return false
	case TRUE:
		return true
	case FALSE:
		return false
	default:
		return true
	}
}

func isError(obj object.Object) bool {
	if obj != nil {
		return obj.Type() == object.ERROR_OBJ
	}

	return false
}

func nativeBoolToBooleanObject(input bool) *object.Boolean {
	if input {
		return TRUE
	}
	return FALSE
}

func newError(format string, a ...interface{}) *object.Error {
	return &object.Error{Message: fmt.Sprintf(format, a...)}
}
