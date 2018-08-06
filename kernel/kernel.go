package kernel

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/crcc/jsonp/engine"
)

var (
	ErrUninitializedValue = errors.New("Uninitialized Value")
)

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
		env.Define(name, NewUninitializedValue())
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

// module
type moduleInfo struct {
	name         string
	filename     string
	exportValues map[string]Exp
	loaded       bool
}

const (
	FindPathsKey   = "find-paths"
	EvalLevelKey   = "evaluate-level"
	ModuleTableKey = "module-table"
	ImportingKey   = "importing"
	ExportNamesKey = "export-names"
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

func GetImporting(ctx Context) bool {
	v := ctx.Get(ImportingKey)
	if v == nil {
		return true
	}
	return v.(bool)
}

func GetModuleTable(ctx Context) map[string]*moduleInfo {
	v := ctx.Get(ModuleTableKey)
	if v == nil {
		mt := map[string]*moduleInfo{
			preludeModule.name: preludeModule,
		}
		ctx.Top().Set(ModuleTableKey, mt)
		return mt
	}
	return v.(map[string]*moduleInfo)
}

func GetFindPaths(ctx Context) []string {
	v := ctx.Get(FindPathsKey)
	if v == nil {
		return nil
	}
	return v.([]string)
}

func GetPreludeEnv(ctx Context) (Env, error) {
	mt := GetModuleTable(ctx)
	preludeModule := mt["prelude"]
	if preludeModule == nil && preludeModule.loaded == false {
		return nil, fmt.Errorf("missing prelude module or invalid prelude module")
	}
	return engine.NewEnv(preludeModule.exportValues).Protect(), nil
}

func GetExportNames(ctx Context) map[string]string {
	v := ctx.Get(ExportNamesKey)
	if v == nil {
		names := map[string]string{}
		ctx.Top().Set(ExportNamesKey, names)
		return names
	}
	return v.(map[string]string)
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

	// get content
	m, err := engine.ToMapExp(exp)
	if err != nil {
		return nil, err
	}

	if len(m) != 3 {
		return nil, fmt.Errorf("invalid module redex: %v", m)
	}
	// create moduleInfo
	moduleName, err := getStringValue(m, "name")
	if err != nil {
		return nil, err
	}
	mt := GetModuleTable(ctx)
	if m, ok := mt[moduleName]; ok {
		if m.loaded {
			return engine.NewNull(), nil
		}
		return nil, fmt.Errorf("error: circular loading module %q", moduleName)
	}
	moduleFile, err := getStringValue(m, "file")
	if err != nil {
		return nil, err
	}
	module := &moduleInfo{
		name:     moduleName,
		filename: moduleFile,
		loaded:   false,
	}
	mt[moduleName] = module

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
	importing := true
	for _, subExp := range l {
		if importing && !isImport(subExp) {
			importing = false
			ctx.Set(ImportingKey, false)
		}
		_, err := interp.Interpret(ctx, subExp, env)
		if err != nil {
			return nil, err
		}
	}

	// export names
	names := GetExportNames(ctx)
	module.exportValues = make(map[string]Exp, len(names))
	for name, defName := range names {
		val, err := env.Get(defName)
		if err != nil {
			return nil, err
		}
		module.exportValues[name] = val
	}
	module.loaded = true

	return engine.NewNull(), nil
}

func LoadModule(ctx Context, findPaths []string, moduleName string, parser engine.Parser) (string, Exp, error) {
	parts := strings.Split(moduleName, "/")
	if len(parts) == 0 || parts[0] == "" || parts[len(parts)-1] == "" {
		return "", nil, fmt.Errorf("invalid module name")
	}
	modulePath := strings.Join(parts, string(os.PathSeparator))

	name := moduleName
	var filename string
	for _, path := range findPaths {
		// try find file
		fileInfo, err := os.Stat(path + modulePath + ".jsonp")
		if err != nil {
			// try find directory
			fileInfo, err = os.Stat(path + modulePath + string(os.PathSeparator) + "main.jsonp")
			if err != nil {
				continue
			}
		} else if parts[len(parts)-1] == "main" {
			name = strings.Join(parts[:len(parts)-1], "/")
		}

		filename = fileInfo.Name()

		file, err := os.Open(filename)
		if err != nil {
			return "", nil, err
		}

		var l []Exp
		for err = nil; err == nil; {
			exp, err := parser.Parse(ctx, file)
			if err != nil {
				l = append(l, exp)
			}
		}
		if err == io.EOF {
			return name, engine.NewRedex("module", engine.NewMapExp(
				map[string]Exp{
					"name": engine.NewString(name),
					"file": engine.NewString(filename),
					"body": engine.NewListExp(l),
				})), nil
		}

		return "", nil, err
	}

	return "", nil, fmt.Errorf("cannot find module %s in paths: %v", moduleName, findPaths)
}

func toNameMap(specExp Exp) (map[string]string, error) {
	l, err := engine.ToListExp(specExp)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string, len(l))
	for _, subExp := range l {
		switch subExp.Kind() {
		case engine.StringValue:
			s, err := engine.ToString(subExp)
			if err != nil {
				return nil, err
			}
			result[s] = s
		case engine.ListExp:
			l, err := engine.ToListExp(subExp)
			if err != nil {
				return nil, err
			}
			if len(l) != 2 {
				return nil, fmt.Errorf("expect [name alias]")
			}
			name, err := engine.ToString(l[0])
			if err != nil {
				return nil, err
			}
			alias, err := engine.ToString(l[1])
			if err != nil {
				return nil, err
			}
			result[name] = alias
		}
	}
	return result, nil
}

func importRedexInterpret(ctx Context, interp Interpreter, exp Exp, env Env) (Exp, error) {
	// check level
	level := GetEvalLevel(ctx)
	if level != ModuleLevel && level != TopLevel {
		return nil, fmt.Errorf("cannot evaluate module in %s", level.String())
	}

	// check importing
	if level == ModuleLevel && !GetImporting(ctx) {
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
	findPaths := GetFindPaths(ctx)
	if len(findPaths) == 0 {
		return nil, fmt.Errorf("empty find paths")
	}

	for name, specExp := range m {
		nameMap, err := toNameMap(specExp)
		if err != nil {
			return nil, err
		}

		newCtx := ctx.Protect()
		moduleName, exp, err := LoadModule(newCtx, findPaths, name, engine.ParserFunc(ParseJson))
		if err != nil {
			return nil, err
		}

		newCtx = ctx.NewChild(map[string]interface{}{
			EvalLevelKey: ModuleLevel,
		})
		newEnv, err := GetPreludeEnv(ctx)
		if err != nil {
			return nil, err
		}
		if _, err := interp.Interpret(newCtx, exp, newEnv); err != nil {
			return nil, err
		}

		mt := GetModuleTable(ctx)
		module := mt[moduleName]
		for name, alias := range nameMap {
			val := module.exportValues[name]
			if val == nil {
				return nil, fmt.Errorf("cannot import %s, no such name in module: %q", name, moduleName)
			}
			// TODO
			var _ = alias
			env.Define(name, val)
		}
	}

	return engine.NewNull(), nil
}

// init
var (
	KernelInterpreter engine.Interpreter
	preludeModule     *moduleInfo
)

func init() {
	KernelInterpreter = NewKernelInterpreter()
	preludeModule = &moduleInfo{
		name:   "prelude",
		loaded: true,
		exportValues: map[string]Exp{
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
		},
	}
}
