package lang

import (
	"errors"
	"fmt"

	"github.com/crcc/jsonp/engine"
)

var KernelInterpreter engine.Interpreter = NewKernelInterpreter()

func NewKernelInterpreter() engine.Interpreter {
	interp := engine.NewNormalOrderInterpreter(false)
	interp.RegisterInterpreter("var", engine.RedexInterpreterFunc(varRedexInterpret))
	interp.RegisterInterpreter("func", engine.RedexInterpreterFunc(funcRedexInterpret))
	interp.RegisterInterpreter("apply", engine.RedexInterpreterFunc(applyRedexInterpret))
	interp.RegisterInterpreter("def", engine.RedexInterpreterFunc(defRedexInterpret))
	interp.RegisterInterpreter("set", engine.RedexInterpreterFunc(setRedexInterpret))
	interp.RegisterInterpreter("begin", engine.RedexInterpreterFunc(beginRedexIntepret))
	interp.RegisterInterpreter("if", engine.RedexInterpreterFunc(ifRedexIntepret))
	interp.RegisterInterpreter("block", engine.RedexInterpreterFunc(blockRedexInterpret))
	return interp
}

//// Custom Value
const (
	ClosureTag engine.Tag = iota
	UninitializedTag
	PrimitiveTag
)

// Closure
type Closure struct {
	Args []string
	Body Exp
	Env  Env
}

func NewClosure(args []string, body Exp, env Env) engine.CustomVal {
	return engine.CustomVal{
		Tag: ClosureTag,
		Value: &Closure{
			Args: args,
			Body: body,
			Env:  env,
		},
	}
}

var ErrNotClosureValue = errors.New("Not Closure Value")

func ToClosure(exp Exp) (*Closure, error) {
	v, err := engine.ToCustomValue(exp)
	if err != nil {
		return nil, err
	}

	if v.Tag != ClosureTag {
		return nil, ErrNotClosureValue
	}

	return v.Value.(*Closure), nil
}

// Uninitialized Value
var UninitializedValue = engine.CustomVal{
	Tag:   UninitializedTag,
	Value: nil,
}

func IsUninitializedValue(exp Exp) bool {
	v, err := engine.ToCustomValue(exp)
	if err != nil {
		return false
	}
	return v.Tag == UninitializedTag
}

var ErrUninitializedValue = errors.New("Uninitialized Value")

// Primitive Function
type Primitive struct {
	Arity int
	Func  func(vals []Exp) (Exp, error)
}

func NewPrimitive(arity int, f func(vals []Exp) (Exp, error)) engine.CustomVal {
	return engine.CustomVal{
		Tag: PrimitiveTag,
		Value: &Primitive{
			Arity: arity,
			Func:  f,
		},
	}
}

var ErrNotPrimitiveValue = errors.New("Not Primitive Value")

func ToPrimitive(exp Exp) (*Primitive, error) {
	v, err := engine.ToCustomValue(exp)
	if err != nil {
		return nil, err
	}

	if v.Tag != PrimitiveTag {
		return nil, ErrNotPrimitiveValue
	}

	return v.Value.(*Primitive), nil
}

// redex interpreter

func varRedexInterpret(ctx Context, intrep Interpreter, exp Exp, env Env) (Exp, error) {
	varName, err := engine.ToString(exp)
	if err != nil {
		return nil, err
	}

	val, err := env.Get(varName)
	if err != nil {
		return nil, err
	}
	if IsUninitializedValue(val) {
		return nil, ErrUninitializedValue
	}

	return val, nil
}

func validVarName(name string) error {
	if name == "..." || name == "" {
		return errors.New("illegal variable name: " + name)
	}
	return nil
}

func funcRedexInterpret(ctx Context, interp Interpreter, exp Exp, env Env) (Exp, error) {
	// get args and body
	l, err := engine.ToListExp(exp)
	if err != nil {
		return nil, err
	}

	if len(l) != 2 {
		return nil, errors.New("expect [args, body]")
	}

	argExps, err := engine.ToListExp(l[0])
	if err != nil {
		return nil, err
	}
	body := l[1]

	// 0 args
	if len(argExps) == 0 {
		return NewClosure(nil, body, env), nil
	}

	// convert args
	args := make([]string, len(argExps))
	dupM := make(map[string]struct{}, len(argExps))
	for i, subExp := range argExps {
		arg, err := engine.ToString(subExp)
		if err != nil {
			return nil, err
		}
		if err := validVarName(arg); err != nil {
			return nil, err
		}
		_, ok := dupM[arg]
		if ok {
			return nil, errors.New("duplicated argument: " + arg)
		}
		args[i] = arg
		dupM[arg] = struct{}{}
	}

	return NewClosure(args, body, env), nil
}

func applyRedexInterpret(ctx Context, interp Interpreter, exp Exp, env Env) (Exp, error) {
	l, err := engine.ToListExp(exp)
	if err != nil {
		return nil, err
	}

	if len(l) == 0 {
		return nil, errors.New("expect [func args ...]")
	}

	funcExp, err := interp.Interpret(ctx, l[0], env)
	if err != nil {
		return nil, err
	}
	argExps := l[1:]

	// primitive
	pri, err := ToPrimitive(funcExp)
	if err == nil {
		if len(argExps) != pri.Arity {
			return nil, errors.New(fmt.Sprintf("invalid arity. expect %d args", pri.Arity))
		}

		args := make([]Exp, len(argExps))
		for i, argExp := range argExps {
			arg, err := interp.Interpret(ctx, argExp, env)
			if err != nil {
				return nil, err
			}
			args[i] = arg
		}

		return pri.Func(args)
	}

	// closure
	clo, err := ToClosure(funcExp)
	if err != nil {
		return nil, err
	}

	if len(argExps) != len(clo.Args) {
		return nil, errors.New(fmt.Sprintf("invalid arity. expect %d args", pri.Arity))
	}

	kvs := make(map[string]Exp, len(clo.Args))
	for i, argExp := range argExps {
		arg, err := interp.Interpret(ctx, argExp, env)
		if err != nil {
			return nil, err
		}

		kvs[clo.Args[i]] = arg
	}

	newEnv := clo.Env.Extend(kvs)
	return engine.NewDelayedExp(ctx, clo.Body, newEnv), nil
}

func defRedexInterpret(ctx Context, interp Interpreter, exp Exp, env Env) (Exp, error) {
	m, err := engine.ToMapExp(exp)
	if err != nil {
		return nil, err
	}

	for name, _ := range m {
		env.Define(name, UninitializedValue)
	}

	vals := make(map[string]Exp, len(m))
	for name, subExp := range m {
		val, err := interp.Interpret(ctx, subExp, env)
		if err != nil {
			return nil, err
		}

		vals[name] = val
	}

	for name, val := range vals {
		env.Define(name, val)
	}

	return engine.NewNull(), nil
}

func setRedexInterpret(ctx Context, interp Interpreter, exp Exp, env Env) (Exp, error) {
	m, err := engine.ToMapExp(exp)
	if err != nil {
		return nil, err
	}

	vals := make(map[string]Exp, len(m))
	for name, subExp := range m {
		val, err := interp.Interpret(ctx, subExp, env)
		if err != nil {
			return nil, err
		}

		vals[name] = val
	}

	for name, val := range vals {
		env.Set(name, val)
	}

	return engine.NewNull(), nil
}

func beginRedexIntepret(ctx Context, interp Interpreter, exp Exp, env Env) (Exp, error) {
	l, err := engine.ToListExp(exp)
	if err != nil {
		return nil, err
	}

	for _, subExp := range l[:len(l)-1] {
		_, err = interp.Interpret(ctx, subExp, env)
		if err != nil {
			return nil, err
		}
	}

	lastExp := l[len(l)-1]
	return engine.NewDelayedExp(ctx, lastExp, env), nil
}

func ifRedexIntepret(ctx Context, interp Interpreter, exp Exp, env Env) (Exp, error) {
	l, err := engine.ToListExp(exp)
	if err != nil {
		return nil, err
	}

	if len(l) != 3 {
		return nil, errors.New("expect [test then else]")
	}

	testResult, err := interp.Interpret(ctx, l[0], env)
	if err != nil {
		return nil, err
	}

	res, err := engine.ToBoolean(testResult)
	if err != nil {
		return nil, err
	}

	if res {
		return engine.NewDelayedExp(ctx, l[1], env), nil
	}
	return engine.NewDelayedExp(ctx, l[2], env), nil
}

func blockRedexInterpret(ctx Context, interp Interpreter, exp Exp, env Env) (Exp, error) {
	newEnv := env.Extend(nil)
	return engine.NewDelayedExp(ctx, exp, newEnv), nil
}

func InitEnv() Env {
	kvs := map[string]Exp{
		"+": NewPrimitive(2, func(vals []Exp) (Exp, error) {
			n1, err := engine.ToNumber(vals[0])
			if err != nil {
				return nil, err
			}

			n2, err := engine.ToNumber(vals[1])
			if err != nil {
				return nil, err
			}

			return engine.NewNumber(n1 + n2), nil
		}),
		"-": NewPrimitive(2, func(vals []Exp) (Exp, error) {
			n1, err := engine.ToNumber(vals[0])
			if err != nil {
				return nil, err
			}

			n2, err := engine.ToNumber(vals[1])
			if err != nil {
				return nil, err
			}

			return engine.NewNumber(n1 - n2), nil
		}),
		"*": NewPrimitive(2, func(vals []Exp) (Exp, error) {
			n1, err := engine.ToNumber(vals[0])
			if err != nil {
				return nil, err
			}

			n2, err := engine.ToNumber(vals[1])
			if err != nil {
				return nil, err
			}

			return engine.NewNumber(n1 * n2), nil
		}),
		"/": NewPrimitive(2, func(vals []Exp) (Exp, error) {
			n1, err := engine.ToNumber(vals[0])
			if err != nil {
				return nil, err
			}

			n2, err := engine.ToNumber(vals[1])
			if err != nil {
				return nil, err
			}

			return engine.NewNumber(n1 / n2), nil
		}),
		"<": NewPrimitive(2, func(vals []Exp) (Exp, error) {
			n1, err := engine.ToNumber(vals[0])
			if err != nil {
				return nil, err
			}

			n2, err := engine.ToNumber(vals[1])
			if err != nil {
				return nil, err
			}

			return engine.NewBoolean(n1 < n2), nil
		}),
		">": NewPrimitive(2, func(vals []Exp) (Exp, error) {
			n1, err := engine.ToNumber(vals[0])
			if err != nil {
				return nil, err
			}

			n2, err := engine.ToNumber(vals[1])
			if err != nil {
				return nil, err
			}

			return engine.NewBoolean(n1 > n2), nil
		}),
		"<=": NewPrimitive(2, func(vals []Exp) (Exp, error) {
			n1, err := engine.ToNumber(vals[0])
			if err != nil {
				return nil, err
			}

			n2, err := engine.ToNumber(vals[1])
			if err != nil {
				return nil, err
			}

			return engine.NewBoolean(n1 <= n2), nil
		}),
		">=": NewPrimitive(2, func(vals []Exp) (Exp, error) {
			n1, err := engine.ToNumber(vals[0])
			if err != nil {
				return nil, err
			}

			n2, err := engine.ToNumber(vals[1])
			if err != nil {
				return nil, err
			}

			return engine.NewBoolean(n1 >= n2), nil
		}),
		"=": NewPrimitive(2, func(vals []Exp) (Exp, error) {
			n1, err := engine.ToNumber(vals[0])
			if err != nil {
				return nil, err
			}

			n2, err := engine.ToNumber(vals[1])
			if err != nil {
				return nil, err
			}

			return engine.NewBoolean(n1 == n2), nil
		}),
		"printString": NewPrimitive(1, func(vals []Exp) (Exp, error) {
			s, err := engine.ToString(vals[0])
			if err != nil {
				return nil, err
			}

			println(s)

			return engine.NewNull(), nil
		}),
	}

	return engine.NewEnv(kvs)
}
