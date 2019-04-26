package kernel

import (
	"errors"
	"fmt"

	"github.com/crcc/jsonp/engine"
)

var (
	ErrUninitializedValue = errors.New("Uninitialized Value")
)

// type Evaluator interface {
// 	Eval(exp Exp, level EvalLevel) (Exp, error)
// 	LoadModule(name string) (*Module, error)
// }

// func NewEvaluator(loader ModuleLoader) Evaluator {
// 	return &evaluator{
// 		interp:       KernelInterpreter,
// 		moduleLoader: loader,
// 	}
// }

// type evaluator struct {
// 	interp       engine.Interpreter
// 	moduleLoader ModuleLoader
// }

// func (e *evaluator) Eval(exp Exp, level EvalLevel) (Exp, error) {
// 	ctx := engine.NewContext(map[string]interface{}{
// 		EvalLevelKey:    level,
// 		ModuleLoaderKey: e.moduleLoader,
// 	})

// 	var env Env
// 	if level == ModuleLevel {
// 		env = engine.NewEnv(nil)
// 	} else {
// 		env = engine.NewEnv(preludeModule.ExportValues).Protect()
// 	}
// 	return e.interp.Interpret(ctx, exp, env)
// }

// func (e *evaluator) LoadModule(name string) (*Module, error) {
// 	ctx := engine.NewContext(map[string]interface{}{
// 		EvalLevelKey:    ModuleLevel,
// 		ModuleLoaderKey: e.moduleLoader,
// 	})
// 	return e.moduleLoader.LoadModule(ctx, e.interp, name)
// }

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
	interp.RegisterInterpreter("module", engine.RedexInterpreterFunc(moduleRedexInterpret))
	interp.RegisterInterpreter("import", engine.RedexInterpreterFunc(importRedexInterpret))
	interp.RegisterInterpreter("export", engine.RedexInterpreterFunc(exportRedexInterpret))
	return interp
}

// redex interpreter

func varRedexInterpret(ctx Context, intrep Interpreter, exp Exp, env Env) (Exp, error) {
	// check level: any level
	// leaf exp
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
	// check level: any level
	// leaf exp
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
	// check level: any level
	// function and arguments evaluated in ExprLevel
	// closure body evaluated in BlockLevel
	l, err := engine.ToListExp(exp)
	if err != nil {
		return nil, err
	}

	if len(l) == 0 {
		return nil, errors.New("expect [func args ...]")
	}

	newCtx := EnsureEvalLevel(ctx, ExprLevel)
	funcExp, err := interp.Interpret(newCtx, l[0], env)
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
			arg, err := interp.Interpret(newCtx, argExp, env)
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
		return nil, errors.New(fmt.Sprintf("invalid arity. expect %d args", len(clo.Args)))
	}

	kvs := make(map[string]Exp, len(clo.Args))
	for i, argExp := range argExps {
		arg, err := interp.Interpret(newCtx, argExp, env)
		if err != nil {
			return nil, err
		}

		kvs[clo.Args[i]] = arg
	}

	newCtx = EnsureEvalLevel(ctx, BlockLevel)
	newEnv := clo.Env.Extend(kvs)
	return engine.NewDelayedExp(newCtx, clo.Body, newEnv), nil
}

func defRedexInterpret(ctx Context, interp Interpreter, exp Exp, env Env) (Exp, error) {
	// check level
	level := GetEvalLevel(ctx)
	if level == ExprLevel {
		return nil, fmt.Errorf("cannot evaluate def in %s", level.String())
	}
	// exps evaluated in ExprLevel

	// get body
	m, err := engine.ToMapExp(exp)
	if err != nil {
		return nil, err
	}

	for name, _ := range m {
		env.Define(name, NewUninitializedValue())
	}

	newCtx := EnsureEvalLevel(ctx, ExprLevel)
	vals := make(map[string]Exp, len(m))
	for name, subExp := range m {
		val, err := interp.Interpret(newCtx, subExp, env)
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
	// check level: any level
	// exps evaluated in ExprLevel
	// get body
	m, err := engine.ToMapExp(exp)
	if err != nil {
		return nil, err
	}

	newCtx := EnsureEvalLevel(ctx, ExprLevel)
	vals := make(map[string]Exp, len(m))
	for name, subExp := range m {
		val, err := interp.Interpret(newCtx, subExp, env)
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
	// check level: any level
	// passing level
	// get body
	l, err := engine.ToListExp(exp)
	if err != nil {
		return nil, err
	}

	if len(l) == 0 {
		return nil, errors.New("empty begin sequence")
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
	// check level: any level
	// test, then, else evaluated in ExprLevel
	// get body
	l, err := engine.ToListExp(exp)
	if err != nil {
		return nil, err
	}

	if len(l) != 3 {
		return nil, errors.New("expect [test then else]")
	}

	newCtx := EnsureEvalLevel(ctx, ExprLevel)
	testResult, err := interp.Interpret(newCtx, l[0], env)
	if err != nil {
		return nil, err
	}

	res, err := engine.ToBoolean(testResult)
	if err != nil {
		return nil, err
	}

	if res {
		return engine.NewDelayedExp(newCtx, l[1], env), nil
	}
	return engine.NewDelayedExp(newCtx, l[2], env), nil
}

func blockRedexInterpret(ctx Context, interp Interpreter, exp Exp, env Env) (Exp, error) {
	// check level: any level
	// body evaluated in BlockLevel
	newCtx := EnsureEvalLevel(ctx, BlockLevel)
	newEnv := env.Extend(nil)
	return engine.NewDelayedExp(newCtx, exp, newEnv), nil
}

// module

const (
	ModuleLoaderKey  = "module-loader"
	EvalLevelKey     = "evaluate-level"
	CurrentModuleKey = "current-module"
)

type EvalLevel uint8

const (
	TopLevel EvalLevel = iota
	ModuleLevel
	BlockLevel
	ExprLevel
)

func (l EvalLevel) String() string {
	switch l {
	case TopLevel:
		return "Top Level"
	case ModuleLevel:
		return "Module Level"
	case BlockLevel:
		return "Block Level"
	case ExprLevel:
		return "Expr Level"
	default:
		panic("unknown EvalLevel")
	}
}

func GetEvalLevel(ctx Context) EvalLevel {
	level := ctx.Get(EvalLevelKey)
	if level == nil {
		return TopLevel
	}
	return level.(EvalLevel)
}

func EnsureEvalLevel(ctx Context, level EvalLevel) Context {
	l := GetEvalLevel(ctx)
	if level != l {
		return ctx.NewChild(map[string]interface{}{
			EvalLevelKey: level,
		})
	}
	return ctx
}

func GetModuleLoader(ctx Context) ModuleLoader {
	v := ctx.Get(ModuleLoaderKey)
	if v == nil {
		return nil
	}
	return v.(ModuleLoader)
}

func GetCurrentModule(ctx Context) *Module {
	v := ctx.Get(CurrentModuleKey)
	if v == nil {
		return nil
	}
	return v.(*Module)
}

func isImport(exp Exp) bool {
	r, err := engine.ToRedex(exp)
	if err != nil {
		return false
	}
	return r.Name == "import"
}

func getStringValue(m map[string]Exp, key string) (string, error) {
	exp, ok := m[key]
	if !ok {
		return "", fmt.Errorf("invalid module redex: empty %s", key)
	}
	return engine.ToString(exp)
}

// 先不做
// 增加一个pass做name checking，类似type checking（先不做）
// export在执行阶段，必须放在模块的最后面（先不做）
// import时，隐式导入的名字相同，则不能使用该名字，使用则报错。
// import时，隐式导入的名字和显式导入的名字相同，则使用显式导入的名字
// import时，显式导入的名字相同，则报错

// 一个模块只加载一次
// 模块依赖不能成环
// import必须在模块的最开头，必须在顶层
// import时，如果导入的名字相同，则报错

// export可以在顶层，import之后的任意位置
// export时，导出的名字必须不相同，否则报错

// 需要的信息
// Find Paths
// Current Module Name
// 是否在模块顶层
// 模块表
func moduleRedexInterpret(ctx Context, interp Interpreter, exp Exp, env Env) (Exp, error) {
	// check level
	level := GetEvalLevel(ctx)
	if level != ModuleLevel {
		return nil, fmt.Errorf("cannot evaluate module in %s", level.String())
	}
	// body evaluated in ModuleLevel

	// get content
	m, err := engine.ToMapExp(exp)
	if err != nil {
		return nil, err
	}

	if len(m) != 3 {
		return nil, fmt.Errorf("invalid module redex: %v", m)
	}
	// create module
	moduleName, err := getStringValue(m, "name")
	if err != nil {
		return nil, err
	}
	mt := GetModuleTable(ctx)
	if _, ok := mt[moduleName]; ok {
		return nil, fmt.Errorf("module %s is loading or loaded, should not evaluate module again", moduleName)
	}

	moduleFile, err := getStringValue(m, "file")
	if err != nil {
		return nil, err
	}
	importValues, err := GetInitImportValues(ctx)
	if err != nil {
		return nil, err
	}

	// init context
	module := NewModule(moduleName, moduleFile, importValues)
	state := module.LoadingState()
	mt[moduleName] = module
	ctx.Set(CurrentModuleKey, module)

	// get module body
	body, ok := m["body"]
	if !ok {
		return nil, fmt.Errorf("invalid module redex, missing body")
	}
	l, err := engine.ToListExp(body)
	if err != nil {
		return nil, err
	}

	// evaluate module
	for _, subExp := range l {
		if state.ImportingStage && !isImport(subExp) {
			state.ImportingStage = false
			for name, importVal := range module.ImportValues {
				env.Define(name, importVal.Value)
			}
		}
		_, err := interp.Interpret(ctx, subExp, env)
		if err != nil {
			return nil, err
		}
	}

	// export names
	names := state.ExportNames
	module.ExportValues = make(map[string]Exp, len(names))
	for name, defName := range names {
		val, err := env.Get(defName)
		if err != nil {
			return nil, err
		}
		module.ExportValues[name] = val
	}
	module.FinishLoading()

	return engine.NewNull(), nil
}

// import

type importName struct {
	name     string
	explicit bool
}

func importSpecToNameMap(importSpecExp Exp) (map[string]*importName, error) {
	l, err := engine.ToListExp(importSpecExp)
	if err != nil {
		return nil, err
	}

	result := make(map[string]*importName, len(l))
	for _, subExp := range l {
		l, err := engine.ToListExp(subExp)
		if err != nil {
			return nil, err
		}
		var (
			name     string
			alias    string
			explicit bool
		)
		switch len(l) {
		case 2:
			name, err = engine.ToString(l[0])
			if err != nil {
				return nil, err
			}
			alias = name
			explicit, err = engine.ToBoolean(l[1])
			if err != nil {
				return nil, err
			}
		case 3:
			name, err = engine.ToString(l[0])
			if err != nil {
				return nil, err
			}
			alias, err = engine.ToString(l[1])
			if err != nil {
				return nil, err
			}
			explicit, err = engine.ToBoolean(l[2])
			if err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("expect [name explicit] or [name alias exlicit]")
		}
		result[name] = &importName{
			name:     alias,
			explicit: explicit,
		}
	}
	return result, nil
}

func importValue(importValues map[string]*ImportVal, moduleName string, importName *importName, val Exp) error {
	ival, ok := importValues[importName.name]
	if ok {
		if ival.Explicit {
			if importName.explicit {
				return fmt.Errorf("conflict import name: %q, when loading %s", importName.name, moduleName)
			}
		} else {
			if importName.explicit {
				ival.Value = val
				ival.Explicit = true
			} else {
				ival.Value = NewAmbiguousValue()
			}
		}
	} else {
		importValues[importName.name] = &ImportVal{
			Value:    val,
			Explicit: importName.explicit,
		}
	}
	return nil
}

func importRedexInterpret(ctx Context, interp Interpreter, exp Exp, env Env) (Exp, error) {
	// check level
	level := GetEvalLevel(ctx)
	if level != ModuleLevel && level != TopLevel {
		return nil, fmt.Errorf("cannot evaluate import in %s", level.String())
	}

	// check importing
	curModule := GetCurrentModule(ctx)
	if level == ModuleLevel && !curModule.LoadingState().ImportingStage {
		return nil, fmt.Errorf("import statment must at the start of the module")
	}

	// get body
	m, err := engine.ToMapExp(exp)
	if err != nil {
		return nil, err
	}
	if len(m) == 0 {
		return nil, fmt.Errorf("empty import body")
	}

	// find module
	loader := GetModuleLoader(ctx)
	if loader == nil {
		return nil, fmt.Errorf("missing module loader")
	}

	for name, importSpecExp := range m {
		nameMap, err := importSpecToNameMap(importSpecExp)
		if err != nil {
			return nil, err
		}

		module, err := loader.LoadModule(ctx, interp, name)
		if err != nil {
			return nil, err
		}

		// install values
		for name, importName := range nameMap {
			val := module.ExportValues[name]
			if val == nil {
				return nil, fmt.Errorf("cannot import %s, no such name in module: %q", name, module.Name)
			}

			if level == TopLevel {
				env.Define(importName.name, val)
			} else {
				err := importValue(curModule.ImportValues, module.Name, importName, val)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	return engine.NewNull(), nil
}

// export

func exportRedexInterpret(ctx Context, interp Interpreter, exp Exp, env Env) (Exp, error) {
	// check level
	level := GetEvalLevel(ctx)
	if level != ModuleLevel {
		return nil, fmt.Errorf("cannot evaluate export in %s", level.String())
	}

	// get body
	l, err := engine.ToListExp(exp)
	if err != nil {
		return nil, err
	}
	if len(l) == 0 {
		return nil, fmt.Errorf("empty export body")
	}

	// collecting exportNames
	curModule := GetCurrentModule(ctx)
	names := curModule.LoadingState().ExportNames
	for _, subExp := range l {
		err := addExportNameToNameMap(subExp, names)
		if err != nil {
			return nil, err
		}
	}

	return engine.NewNull(), nil
}

func addExportNameToNameMap(subExp Exp, names map[string]string) error {
	switch subExp.Kind() {
	case engine.StringValue:
		s, err := engine.ToString(subExp)
		if err != nil {
			return err
		}
		if _, ok := names[s]; ok {
			return fmt.Errorf("conflict export name: %s", s)
		}
		names[s] = s
	case engine.ListExp:
		l, err := engine.ToListExp(subExp)
		if err != nil {
			return err
		}
		if len(l) != 2 {
			return fmt.Errorf("expect [name alias]")
		}
		name, err := engine.ToString(l[0])
		if err != nil {
			return err
		}
		alias, err := engine.ToString(l[1])
		if err != nil {
			return err
		}
		if _, ok := names[alias]; ok {
			return fmt.Errorf("conflict export name: %s", alias)
		}
		names[alias] = name
	default:
		return fmt.Errorf("invalid export spec: %s", subExp.String())
	}
	return nil
}

// init
var (
	preludeModule *Module
)

func init() {
	preludeModule = &Module{
		Name: "prelude",
		ExportValues: map[string]Exp{
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
			"equal": NewPrimitive(2, func(vals []Exp) (Exp, error) {
				return engine.NewBoolean(vals[0].Equal(vals[1])), nil
			}),
			"append-string": NewPrimitive(2, func(vals []Exp) (Exp, error) {
				s1, err := engine.ToString(vals[0])
				if err != nil {
					return nil, err
				}

				s2, err := engine.ToString(vals[1])
				if err != nil {
					return nil, err
				}

				return engine.NewString(s1 + s2), nil
			}),
			"print": NewPrimitive(1, func(vals []Exp) (Exp, error) {
				fmt.Println(vals[0].String())

				return engine.NewNull(), nil
			}),
		},
	}
}
