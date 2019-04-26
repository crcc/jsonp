package kernel

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/crcc/jsonp/engine"
)

// module

type ImportVal struct {
	Value    Exp
	Explicit bool
}

type LoadingState struct {
	ImportingStage bool
	// exporting name -> definition name
	ExportNames map[string]string
}

type Module struct {
	Name         string
	Filename     string
	ImportValues map[string]*ImportVal
	ExportValues map[string]Exp
	loadingState *LoadingState
}

func NewModule(name, filename string, importValues map[string]*ImportVal) *Module {
	return &Module{
		Name:         name,
		Filename:     filename,
		ImportValues: importValues,
		loadingState: &LoadingState{
			ImportingStage: true,
			ExportNames:    make(map[string]string),
		},
	}
}

func (m *Module) LoadingState() *LoadingState {
	return m.loadingState
}

func (m *Module) IsLoaded() bool {
	return m.loadingState == nil
}

func (m *Module) FinishLoading() {
	m.loadingState = nil
}

// module table

const (
	ModuleTableKey    = "module-table"
	PreludeModuleName = "prelude"
)

func GetModuleTable(ctx Context) map[string]*Module {
	v := ctx.Get(ModuleTableKey)
	if v == nil {
		mt := map[string]*Module{
			preludeModule.Name: preludeModule,
		}
		ctx.Top().Set(ModuleTableKey, mt)
		return mt
	}
	return v.(map[string]*Module)
}

func NewInitImportValues(module *Module) map[string]*ImportVal {
	result := make(map[string]*ImportVal, len(module.ExportValues))
	for name, val := range module.ExportValues {
		result[name] = &ImportVal{
			Value:    val,
			Explicit: false,
		}
	}
	return result
}

func GetInitImportValues(ctx Context) (map[string]*ImportVal, error) {
	mt := GetModuleTable(ctx)
	preludeModule := mt[PreludeModuleName]
	if preludeModule == nil && !preludeModule.IsLoaded() {
		return nil, fmt.Errorf("missing prelude module or invalid prelude module")
	}
	return NewInitImportValues(preludeModule), nil
}

// module loader
type ModuleLoader interface {
	LoadModule(ctx Context, interp Interpreter, name string) (*Module, error)
}

type FileModuleLoader struct {
	findPaths []string
	parser    engine.Parser
}

func NewFileModuleLoader(findPaths []string, parser engine.Parser) *FileModuleLoader {
	return &FileModuleLoader{
		findPaths: findPaths,
		parser:    parser,
	}
}

func (loader *FileModuleLoader) NormalizeModuleName(ctx Context, moduleName string) (string, string, error) {
	parts := strings.Split(moduleName, "/")
	if len(parts) == 0 || parts[0] == "" || parts[len(parts)-1] == "" {
		return "", "", fmt.Errorf("invalid module name")
	}
	modulePath := path.Join(parts...)

	name := moduleName
	var filename string
	for _, rootPath := range loader.findPaths {
		// try find file
		filename = path.Join(rootPath, modulePath+".jsonp")
		fileInfo, err := os.Stat(filename)
		if err != nil {
			// try find directory
			filename = path.Join(rootPath, modulePath, "main.jsonp")
			fileInfo, err = os.Stat(filename)
			if err != nil {
				continue
			}
		} else if parts[len(parts)-1] == "main" {
			name = strings.Join(parts[:len(parts)-1], "/")
		}

		if fileInfo.IsDir() {
			return "", "", fmt.Errorf("expect module %s is a file, but found a directory: %s", moduleName, filename)
		}

		return name, filename, nil
	}

	return "", "", fmt.Errorf("cannot find module %s in paths: %v", moduleName, loader.findPaths)
}

func (loader *FileModuleLoader) ParseModule(ctx Context, moduleName, fileName string) (Exp, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}

	newCtx := ctx.NewChild(map[string]interface{}{
		"module-name": moduleName,
		"file-name":   fileName,
	})
	return loader.parser.Parse(newCtx, file)
}

func (loader *FileModuleLoader) LoadModule(ctx Context, interp Interpreter, name string) (*Module, error) {
	// normalize module name
	newCtx := ctx.Protect()
	moduleName, fileName, err := loader.NormalizeModuleName(newCtx, name)
	if err != nil {
		return nil, err
	}
	// if module is loaded, return it
	mt := GetModuleTable(newCtx)
	if m, ok := mt[moduleName]; ok {
		if m.IsLoaded() {
			return m, nil
		}
		return nil, fmt.Errorf("error: circular loading module %q", moduleName)
	}

	// parse module
	exp, err := loader.ParseModule(newCtx, moduleName, fileName)
	if err != nil {
		return nil, err
	}

	// evaluate module
	newCtx = EnsureEvalLevel(newCtx, ModuleLevel)
	newEnv := engine.NewEnv(nil)
	if _, err := interp.Interpret(newCtx, exp, newEnv); err != nil {
		return nil, err
	}

	module := mt[moduleName]
	return module, nil
}

func (loader *FileModuleLoader) FindPaths() []string {
	return loader.findPaths
}

func (loader *FileModuleLoader) SetFindPaths(paths []string) {
	loader.findPaths = paths
}

type SimpleModuleLoader struct {
	Modules map[string]Exp
}

func (loader *SimpleModuleLoader) LoadModule(ctx Context, interp Interpreter, name string) (*Module, error) {
	// if module is loaded, return it
	mt := GetModuleTable(ctx)
	if m, ok := mt[name]; ok {
		if m.IsLoaded() {
			return m, nil
		}
		return nil, fmt.Errorf("error: circular loading module %q", name)
	}

	exp := loader.Modules[name]
	newCtx := EnsureEvalLevel(ctx.Protect(), ModuleLevel)
	newEnv := engine.NewEnv(nil)
	if _, err := interp.Interpret(newCtx, exp, newEnv); err != nil {
		return nil, err
	}

	module := mt[name]
	return module, nil
}
