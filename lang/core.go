package lang

// import (
// 	"errors"
// 	"fmt"

// 	"github.com/crcc/jsonp/engine"
// )

// var coreInterpreter engine.Interpreter

// //// Custom Value
// const (
// 	ClosureTag engine.Tag = iota
// 	UninitializedTag
// 	PrimitiveTag
// )

// // Closure
// type Closure struct {
// 	Args    []string
// 	RestArg string
// 	Body    Exp
// 	Env     Env
// }

// func NewClosure(args []string, restArg string, body Exp, env Env) engine.CustomVal {
// 	return engine.CustomVal{
// 		Tag: ClosureTag,
// 		Value: &Closure{
// 			Args:    args,
// 			RestArg: restArg,
// 			Body:    body,
// 			Env:     env,
// 		},
// 	}
// }

// var ErrNotClosureValue = errors.New("Not Closure Value")

// func ToClosure(exp Exp) (*Closure, error) {
// 	v, err := engine.ToCustomValue(exp)
// 	if err != nil {
// 		return nil, err
// 	}

// 	if v.Tag != ClosureTag {
// 		return nil, ErrNotClosureValue
// 	}

// 	return v.Value.(*Closure), nil
// }

// // Uninitialized Value
// var UninitializedValue = engine.CustomVal{
// 	Tag:   UninitializedTag,
// 	Value: nil,
// }

// func IsUninitializedValue(exp Exp) bool {
// 	v, err := engine.ToCustomValue(exp)
// 	if err != nil {
// 		return false
// 	}
// 	return v.Tag == UninitializedTag
// }

// var ErrUninitializedValue = errors.New("Uninitialized Value")

// // Primitive Function
// type Primitive struct {
// 	Arity   int
// 	HasRest bool
// 	Func    func(vals []Exp) (Exp, error)
// }

// func NewPrimitive(arity int, hasRest bool, f func(vals []Exp) (Exp, error)) engine.CustomVal {
// 	return engine.CustomVal{
// 		Tag: PrimitiveTag,
// 		Value: &Primitive{
// 			Arity:   arity,
// 			HasRest: hasRest,
// 			Func:    f,
// 		},
// 	}
// }

// var ErrNotPrimitiveValue = errors.New("Not Primitive Value")

// func ToPrimitive(exp Exp) (*Primitive, error) {
// 	v, err := engine.ToCustomValue(exp)
// 	if err != nil {
// 		return nil, err
// 	}

// 	if v.Tag != PrimitiveTag {
// 		return nil, ErrNotPrimitiveValue
// 	}

// 	return v.Value.(*Primitive), nil
// }

// func init() {
// 	interp := engine.NewNormalOrderInterpreter(false)
// 	interp.RegisterInterpreter("var", engine.RedexInterpreterFunc(varRedexInterpret))
// 	interp.RegisterInterpreter("func", engine.RedexInterpreterFunc(funcRedexInterpret))
// 	interp.RegisterInterpreter("apply", engine.RedexInterpreterFunc(applyRedexInterpret))
// 	interp.RegisterInterpreter("def", engine.RedexInterpreterFunc(defRedexInterpret))
// 	interp.RegisterInterpreter("set", engine.RedexInterpreterFunc(setRedexInterpret))
// 	interp.RegisterInterpreter("begin", engine.RedexInterpreterFunc(beginRedexIntepret))
// 	interp.RegisterInterpreter("cond", engine.RedexInterpreterFunc(condRedexIntepret))
// 	interp.RegisterInterpreter("block", engine.RedexInterpreterFunc(blockRedexInterpret))
// 	coreInterpreter = interp
// }

// func varRedexInterpret(ctx Context, intrep Interpreter, exp Exp, env Env) (Exp, error) {
// 	varName, err := engine.ToString(exp)
// 	if err != nil {
// 		return nil, err
// 	}

// 	val, err := env.Get(varName)
// 	if err != nil {
// 		return nil, err
// 	}
// 	if IsUninitializedValue(val) {
// 		return nil, ErrUninitializedValue
// 	}

// 	return val, nil
// }

// func validVarName(name string) error {
// 	if name == "..." || name == "" {
// 		return errors.New("illegal variable name: " + name)
// 	}
// 	return nil
// }

// func funcRedexInterpret(ctx Context, interp Interpreter, exp Exp, env Env) (Exp, error) {
// 	// get args and body
// 	l, err := engine.ToListExp(exp)
// 	if err != nil {
// 		return nil, err
// 	}

// 	if len(l) != 2 {
// 		return nil, errors.New("expect [args, body]")
// 	}

// 	argExps, err := engine.ToListExp(l[0])
// 	if err != nil {
// 		return nil, err
// 	}
// 	body := l[1]

// 	// 0 args
// 	if len(argExps) == 0 {
// 		return NewClosure(nil, "", body, env), nil
// 	}

// 	// get last arg
// 	lastArgExp := argExps[len(argExps)-1]
// 	lastArg, err := engine.ToString(lastArgExp)
// 	if err != nil {
// 		return nil, err
// 	}
// 	restTail := (lastArg == "...")
// 	if !restTail {
// 		if err := validVarName(lastArg); err != nil {
// 			return nil, err
// 		}
// 	}

// 	// 1 arg
// 	if len(argExps) == 1 {
// 		if restTail {
// 			return nil, errors.New("illegal variable name: " + lastArg)
// 		}

// 		return NewClosure([]string{lastArg}, "", body, env), nil
// 	}

// 	// >= 2 args
// 	argExps = argExps[:len(argExps)-1]

// 	// convert other args
// 	args := make([]string, len(argExps), len(argExps)+1)
// 	dupM := make(map[string]struct{}, len(argExps)+1)
// 	if !restTail {
// 		dupM[lastArg] = struct{}{}
// 	}

// 	for i, subExp := range argExps {
// 		arg, err := engine.ToString(subExp)
// 		if err != nil {
// 			return nil, err
// 		}
// 		if err := validVarName(arg); err != nil {
// 			return nil, err
// 		}
// 		_, ok := dupM[arg]
// 		if ok {
// 			return nil, errors.New("duplicated argument: " + arg)
// 		}
// 		args[i] = arg
// 		dupM[arg] = struct{}{}
// 	}

// 	if restTail {
// 		restArg := args[len(args)-1]
// 		args = args[:len(args)-1]
// 		return NewClosure(args, restArg, body, env), nil
// 	}

// 	args = append(args, lastArg)
// 	return NewClosure(args, "", body, env), nil
// }

// func validArity(expectArity int, hasRest bool, actualArity int) error {
// 	if hasRest {
// 		if actualArity < expectArity {
// 			return errors.New(fmt.Sprintf("invalid arity. expect at least %d args", expectArity))
// 		}
// 		return nil
// 	}

// 	if actualArity != expectArity {
// 		return errors.New(fmt.Sprintf("invalid arity. expect %d args", expectArity))
// 	}
// 	return nil
// }

// func applyRedexInterpret(ctx Context, interp Interpreter, exp Exp, env Env) (Exp, error) {
// 	l, err := engine.ToListExp(exp)
// 	if err != nil {
// 		return nil, err
// 	}

// 	if len(l) == 0 {
// 		return nil, errors.New("expect [func args ...]")
// 	}

// 	funcExp, err := interp.Interpret(ctx, l[0], env)
// 	if err != nil {
// 		return nil, err
// 	}
// 	argExps := l[1:]

// 	// primitive
// 	pri, err := ToPrimitive(funcExp)
// 	if err == nil {
// 		if err := validArity(pri.Arity, pri.HasRest, len(argExps)); err != nil {
// 			return nil, err
// 		}

// 		args := make([]Exp, len(argExps))
// 		for i, argExp := range argExps {
// 			arg, err := interp.Interpret(ctx, argExp, env)
// 			if err != nil {
// 				return nil, err
// 			}
// 			args[i] = arg
// 		}

// 		return pri.Func(args)
// 	}

// 	// closure
// 	clo, err := ToClosure(funcExp)
// 	if err != nil {
// 		return nil, err
// 	}

// 	hasRest := clo.RestArg != ""
// 	if err := validArity(len(clo.Args), hasRest, len(argExps)); err != nil {
// 		return nil, err
// 	}

// 	kvs := make(map[string]Exp, len(clo.Args)+1)
// 	restArgs := make([]Exp, 0, len(argExps)-len(clo.Args))
// 	for i, argExp := range argExps {
// 		arg, err := interp.Interpret(ctx, argExp, env)
// 		if err != nil {
// 			return nil, err
// 		}

// 		if i < len(clo.Args) {
// 			key := clo.Args[i]
// 			kvs[key] = arg
// 		} else {
// 			restArgs = append(restArgs, arg)
// 		}
// 	}
// 	if hasRest {
// 		kvs[clo.RestArg] = engine.NewList(restArgs)
// 	}

// 	newEnv := clo.Env.Extend(kvs)
// 	return engine.NewDelayedExp(ctx, clo.Body, newEnv), nil
// }

// func defRedexInterpret(ctx Context, interp Interpreter, exp Exp, env Env) (Exp, error) {
// 	m, err := engine.ToMapExp(exp)
// 	if err != nil {
// 		return nil, err
// 	}

// 	for name, _ := range m {
// 		env.Define(name, UninitializedValue)
// 	}

// 	vals := make(map[string]Exp, len(m))
// 	for name, subExp := range m {
// 		val, err := interp.Interpret(ctx, subExp, env)
// 		if err != nil {
// 			return nil, err
// 		}

// 		vals[name] = val
// 	}

// 	for name, val := range vals {
// 		env.Define(name, val)
// 	}

// 	return engine.NewNull(), nil
// }

// func setRedexInterpret(ctx Context, interp Interpreter, exp Exp, env Env) (Exp, error) {
// 	m, err := engine.ToMapExp(exp)
// 	if err != nil {
// 		return nil, err
// 	}

// 	vals := make(map[string]Exp, len(m))
// 	for name, subExp := range m {
// 		val, err := interp.Interpret(ctx, subExp, env)
// 		if err != nil {
// 			return nil, err
// 		}

// 		vals[name] = val
// 	}

// 	for name, val := range vals {
// 		env.Set(name, val)
// 	}

// 	return engine.NewNull(), nil
// }

// func beginRedexIntepret(ctx Context, interp Interpreter, exp Exp, env Env) (Exp, error) {
// 	l, err := engine.ToListExp(exp)
// 	if err != nil {
// 		return nil, err
// 	}

// 	for _, subExp := range l[:len(l)-1] {
// 		_, err = interp.Interpret(ctx, subExp, env)
// 		if err != nil {
// 			return nil, err
// 		}
// 	}

// 	lastExp := l[len(l)-1]
// 	return engine.NewDelayedExp(ctx, lastExp, env), nil
// }

// func condRedexIntepret(ctx Context, interp Interpreter, exp Exp, env Env) (Exp, error) {
// 	l, err := engine.ToListExp(exp)
// 	if err != nil {
// 		return nil, err
// 	}

// 	if len(l) == 0 {
// 		return nil, errors.New("expect [[test exp]...]")
// 	}

// 	clauses := make([][]Exp, len(l))
// 	for i, subExp := range l {
// 		newExp, err := engine.ToList(subExp)
// 		if err != nil {
// 			return nil, err
// 		}
// 		if len(newExp) != 2 {
// 			return nil, errors.New("expect [test exp]")
// 		}
// 		clauses[i] = newExp
// 	}

// 	var caluseExp Exp
// 	for _, clause := range clauses {
// 		testResult, err := interp.Interpret(ctx, clause[0], env)
// 		if err != nil {
// 			return nil, err
// 		}
// 		res, err := engine.ToBoolean(testResult)
// 		if err != nil {
// 			return nil, err
// 		}
// 		if res {
// 			caluseExp = clause[1]
// 			break
// 		}
// 	}

// 	return engine.NewDelayedExp(ctx, caluseExp, env), nil
// }

// func blockRedexInterpret(ctx Context, interp Interpreter, exp Exp, env Env) (Exp, error) {
// 	newEnv := env.Extend(nil)
// 	return engine.NewDelayedExp(ctx, exp, newEnv), nil
// }

// func InitEnv() Env {
// 	kvs := map[string]Exp{
// 		"+": NewPrimitive(0, true, func(vals []Exp) (Exp, error) {
// 			result := 0.0
// 			for _, val := range vals {
// 				n, err := engine.ToNumber(val)
// 				if err != nil {
// 					return nil, err
// 				}
// 				result += n
// 			}
// 			return engine.NewNumber(result), nil
// 		}),
// 	}

// 	return engine.NewEnv(kvs)
// }
