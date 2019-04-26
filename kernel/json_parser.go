package kernel

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/crcc/jsonp/engine"
)

func ParseJson(ctx Context, r io.Reader) (Exp, error) {
	var v interface{}
	if err := json.NewDecoder(r).Decode(&v); err != nil {
		return nil, err
	}

	return ParseJsonStruct(v)
}

func ParseJsonModule(ctx Context, r io.Reader) (Exp, error) {
	moduleName, ok := ctx.Get("module-name").(string)
	if !ok {
		return nil, fmt.Errorf("missing moduleName")
	}
	fileName, ok := ctx.Get("file-name").(string)
	if !ok {
		fileName = ""
	}

	var (
		err error
		l   []Exp
	)
	decoder := json.NewDecoder(r)

	for err == nil {
		var v interface{}
		if err = decoder.Decode(&v); err != nil {
			break
		}

		var exp Exp
		exp, err = ParseJsonStruct(v)
		if err != nil {
			return nil, err
		}
		l = append(l, exp)
	}

	if err == io.EOF {
		return engine.NewRedex("module", engine.NewMapExp(
			map[string]Exp{
				"name": engine.NewString(moduleName),
				"file": engine.NewString(fileName),
				"body": engine.NewListExp(l),
			})), nil
	}
	return nil, err
}

var jsonStructParser = engine.NewJsonStructParser("var", "apply", "data")

func init() {
	jsonStructParser.RegisterRedexParser("func", parseJsonStructFunc)
	jsonStructParser.RegisterRedexParser("def", parseJsonStructDef)
	jsonStructParser.RegisterRedexParser("set", parseJsonStructSet)
	jsonStructParser.RegisterRedexParser("begin", parseJsonStructBegin)
	jsonStructParser.RegisterRedexParser("block", parseJsonStructBlock)
	jsonStructParser.RegisterRedexParser("if", parseJsonStructIf)
	jsonStructParser.RegisterRedexParser("import", parseJsonStructImport)
	jsonStructParser.RegisterRedexParser("export", parseJsonStructExport)
}

func ParseJsonStruct(s interface{}) (Exp, error) {
	return jsonStructParser.Parse(s)
}

func parseJsonStructBody(parser *engine.JsonStructParser, l []interface{}) (Exp, error) {
	switch len(l) {
	case 0:
		return nil, fmt.Errorf("empty body")
	case 1:
		return ParseJsonStruct(l[0])
	default:
		exps, err := parser.ParseListExp(l)
		if err != nil {
			return nil, err
		}
		return engine.NewRedex("begin", engine.NewListExp(exps)), nil
	}
}

func parseJsonStructFunc(parser *engine.JsonStructParser, name string, s interface{}) (Exp, error) {
	l, ok := s.([]interface{})
	if !ok || len(l) < 2 {
		return nil, fmt.Errorf("invalid func syntax: %v", s)
	}

	args, ok := l[0].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid func syntax: %v, expect [args body ...]", s)
	}
	body := l[1:]

	argExps := make([]Exp, len(args))
	for i, arg := range args {
		s, ok := arg.(string)
		if !ok {
			return nil, fmt.Errorf("invalid func syntax: %v, not arg: %v", s, arg)
		}
		argExps[i] = engine.NewString(s)
	}

	bodyExp, err := parseJsonStructBody(parser, body)
	if err != nil {
		return nil, err
	}

	return engine.NewRedex("func", engine.NewListExp([]Exp{
		engine.NewListExp(argExps), bodyExp,
	})), nil
}

func parseJsonStructBegin(parser *engine.JsonStructParser, name string, s interface{}) (Exp, error) {
	l, ok := s.([]interface{})
	if !ok || len(l) == 0 {
		return nil, fmt.Errorf("invalid begin syntax: %v", s)
	}

	exps, err := parser.ParseListExp(l)
	if err != nil {
		return nil, err
	}

	return engine.NewRedex("begin", engine.NewListExp(exps)), nil
}

func parseJsonStructBlock(parser *engine.JsonStructParser, name string, s interface{}) (Exp, error) {
	l, ok := s.([]interface{})
	if !ok || len(l) == 0 {
		return nil, fmt.Errorf("invalid block syntax: %v", s)
	}

	bodyExp, err := parseJsonStructBody(parser, l)
	if err != nil {
		return nil, err
	}

	return engine.NewRedex("block", bodyExp), nil
}

func parseJsonStructIf(parser *engine.JsonStructParser, name string, s interface{}) (Exp, error) {
	l, ok := s.([]interface{})
	if !ok || len(l) != 3 {
		return nil, fmt.Errorf("invalid if syntax: %v", s)
	}

	exps, err := parser.ParseListExp(l)
	if err != nil {
		return nil, err
	}

	return engine.NewRedex("if", engine.NewListExp(exps)), nil
}

func parseJsonStructDef(parser *engine.JsonStructParser, name string, s interface{}) (Exp, error) {
	m, ok := s.(map[string]interface{})
	if !ok || len(m) == 0 {
		return nil, fmt.Errorf("invalid def syntax: %v", s)
	}

	exps, err := parser.ParseMapExp(m)
	if err != nil {
		return nil, err
	}

	return engine.NewRedex("def", engine.NewMapExp(exps)), nil
}

func parseJsonStructSet(parser *engine.JsonStructParser, name string, s interface{}) (Exp, error) {
	m, ok := s.(map[string]interface{})
	if !ok || len(m) == 0 {
		return nil, fmt.Errorf("invalid set syntax: %v", s)
	}

	exps, err := parser.ParseMapExp(m)
	if err != nil {
		return nil, err
	}

	return engine.NewRedex("set", engine.NewMapExp(exps)), nil
}

/*
{"import": {"path": ["name", ["name2", "alias"]]}}
*/
func parseJsonStructImport(parser *engine.JsonStructParser, name string, s interface{}) (Exp, error) {
	m, ok := s.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf(`invalid import syntax: %v, expect {"path": ["name", ["name2", "alias"]], "path2": ...}`, s)
	}

	importMap := make(map[string]Exp, len(m))
	for name, spec := range m {
		ss, ok := spec.([]interface{})
		if !ok {
			return nil, fmt.Errorf(`invalid import syntax: %v, expect ["name", ["name2", "alias"], ...]`, spec)
		}
		specExps := make([]Exp, len(ss))
		for i, item := range ss {
			switch v := item.(type) {
			case string:
				specExps[i] = engine.NewListExp([]Exp{engine.NewString(v), engine.NewBoolean(true)})
			case []interface{}:
				name, ok := v[0].(string)
				if !ok {
					return nil, fmt.Errorf(`invalid import syntax: %v, expect "name"`, v[0])
				}
				alias, ok := v[1].(string)
				if !ok {
					return nil, fmt.Errorf(`invalid import syntax: %v, expect "alias"`, v[1])
				}
				specExps[i] = engine.NewListExp([]Exp{engine.NewString(name), engine.NewString(alias), engine.NewBoolean(true)})
			default:
				return nil, fmt.Errorf(`invalid import syntax: %v, expect "name" or ["name2", "alias"]`, item)
			}
		}
		importMap[name] = engine.NewListExp(specExps)
	}

	return engine.NewRedex(name, engine.NewMapExp(importMap)), nil
}

/*
{"export": ["name", ["name2", "alias"]]}
*/
func parseJsonStructExport(parser *engine.JsonStructParser, name string, s interface{}) (Exp, error) {
	l, ok := s.([]interface{})
	if !ok {
		return nil, fmt.Errorf(`invalid export syntax: %v, expect {"path": ["name", ["name2", "alias"]], "path2": ...}`, s)
	}

	specExps := make([]Exp, len(l))
	for i, item := range l {
		switch v := item.(type) {
		case string:
			specExps[i] = engine.NewString(v)
		case []interface{}:
			name, ok := v[0].(string)
			if !ok {
				return nil, fmt.Errorf(`invalid export syntax: %v, expect "name"`, v[0])
			}
			alias, ok := v[1].(string)
			if !ok {
				return nil, fmt.Errorf(`invalid export syntax: %v, expect "alias"`, v[1])
			}
			specExps[i] = engine.NewListExp([]Exp{engine.NewString(name), engine.NewString(alias)})
		default:
			return nil, fmt.Errorf(`invalid export syntax: %v, expect "name" or ["name2", "alias"]`, item)
		}
	}

	return engine.NewRedex(name, engine.NewListExp(specExps)), nil
}
