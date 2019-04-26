package kernel

import (
	"io"

	"github.com/crcc/jsonp/engine"
)

type Repl struct {
	parser       engine.Parser
	interpreter  engine.Interpreter
	moduleLoader ModuleLoader
}

func NewRepl(parser engine.Parser, interp engine.Interpreter, moduleLoader ModuleLoader) *Repl {
	return &Repl{
		parser:       parser,
		interpreter:  interp,
		moduleLoader: moduleLoader,
	}
}

func (d *Repl) EvalBatch(filename string) error {
	ctx := engine.NewContext(map[string]interface{}{
		EvalLevelKey:    ModuleLevel,
		ModuleLoaderKey: d.moduleLoader,
	})
	_, err := d.moduleLoader.LoadModule(ctx, d.interpreter, filename)
	return err
}

func (d *Repl) Parse(r io.Reader) (Exp, error) {
	ctx := engine.NewContext(nil)
	return d.parser.Parse(ctx, r)
}

func (d *Repl) EvalInteractive(exp Exp) (Exp, error) {
	ctx := engine.NewContext(map[string]interface{}{
		EvalLevelKey:    TopLevel,
		ModuleLoaderKey: d.moduleLoader,
	})

	env := engine.NewEnv(preludeModule.ExportValues).Protect()
	return d.interpreter.Interpret(ctx, exp, env)
}

func (d *Repl) AddPaths(paths []string) {
	loader, ok := d.moduleLoader.(*FileModuleLoader)
	if ok {
		findPaths := loader.FindPaths()
		findPaths = append(paths, findPaths...)
		loader.SetFindPaths(findPaths)
	}
}
