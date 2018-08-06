package kernel

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/crcc/jsonp/engine"
)

func ParseJson(ctx Context, r io.Reader) (Exp, error) {
	var m map[string]interface{}
	if err := json.NewDecoder(r).Decode(&m); err != nil {
		return nil, err
	}

	return ParseJsonStruct(m)
}

var jsonStructParser = engine.NewJsonStructParser("var", "apply", "data")

func init() {
	jsonStructParser.RegisterRedexParser("func", parseJsonStructFunc)
	jsonStructParser.RegisterRedexParser("def", parseJsonStructDef)
	jsonStructParser.RegisterRedexParser("set", parseJsonStructSet)
	jsonStructParser.RegisterRedexParser("begin", parseJsonStructBegin)
	jsonStructParser.RegisterRedexParser("block", parseJsonStructBlock)
	jsonStructParser.RegisterRedexParser("if", parseJsonStructIf)
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
