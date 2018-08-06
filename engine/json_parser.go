package engine

import (
	"fmt"
)

type JsonStructRedexParser func(parser *JsonStructParser, name string, s interface{}) (Exp, error)

type JsonStructParser struct {
	VarRedexName       string
	ApplyRedexName     string
	DataRedexName      string
	redexParsers       map[string]JsonStructRedexParser
	defaultRedexParser JsonStructRedexParser
}

func NewJsonStructParser(varRedex, applyRedex, dataRedex string) *JsonStructParser {
	parser := &JsonStructParser{
		VarRedexName:   varRedex,
		ApplyRedexName: applyRedex,
		redexParsers:   make(map[string]JsonStructRedexParser),
	}
	parser.RegisterDataRedexName(dataRedex)
	return parser
}

func (parser *JsonStructParser) Parse(s interface{}) (Exp, error) {
	if s == nil {
		return NewNull(), nil
	}

	switch v := s.(type) {
	case bool:
		return NewBoolean(v), nil
	case float64:
		return NewNumber(v), nil
	case string:
		return NewRedex(parser.VarRedexName, NewString(v)), nil
	case []interface{}:
		if len(v) == 0 {
			return nil, fmt.Errorf("invalid function application syntax: %v", v)
		}
		exps, err := parser.ParseListExp(v)
		if err != nil {
			return nil, err
		}
		return NewRedex(parser.ApplyRedexName, NewListExp(exps)), nil
	case map[string]interface{}:
		if len(v) != 1 {
			return nil, fmt.Errorf("invalid jsonp special syntax: %v", v)
		}
		for keyword, subExp := range v {
			p := parser.redexParsers[keyword]
			if p == nil {
				return parser.ParseRedexDefaulty(keyword, subExp)
			}
			return p(parser, keyword, subExp)
		}
		panic("Parse: should not get here")
	default:
		return nil, fmt.Errorf("invalid json struct: %v", s)
	}
}

func (parser *JsonStructParser) ParseListExp(v []interface{}) ([]Exp, error) {
	exps := make([]Exp, len(v))
	for i, subExp := range v {
		r, err := parser.Parse(subExp)
		if err != nil {
			return nil, err
		}
		exps[i] = r
	}
	return exps, nil
}

func (parser *JsonStructParser) ParseMapExp(m map[string]interface{}) (map[string]Exp, error) {
	exps := make(map[string]Exp, len(m))
	for name, subExp := range m {
		r, err := parser.Parse(subExp)
		if err != nil {
			return nil, err
		}
		exps[name] = r
	}
	return exps, nil
}

func (parser *JsonStructParser) ParseData(s interface{}) (Exp, error) {
	if s == nil {
		return NewNull(), nil
	}

	switch v := s.(type) {
	case bool:
		return NewBoolean(v), nil
	case float64:
		return NewNumber(v), nil
	case string:
		return NewString(v), nil
	case []interface{}:
		l := make([]Exp, len(v))
		for i, subVal := range v {
			r, err := parser.ParseData(subVal)
			if err != nil {
				return nil, err
			}
			l[i] = r
		}
		return NewList(l), nil
	case map[string]interface{}:
		m := make(map[string]Exp, len(v))
		for key, subVal := range v {
			r, err := parser.ParseData(subVal)
			if err != nil {
				return nil, err
			}
			m[key] = r
		}
		return NewMap(m), nil
	default:
		return nil, fmt.Errorf("invalid json struct: %v", s)
	}
}

func (parser *JsonStructParser) ParseRedexDefaulty(name string, s interface{}) (Exp, error) {
	if parser.defaultRedexParser == nil {
		return nil, fmt.Errorf("cannot handle redex %s", name)
	}

	return parser.defaultRedexParser(parser, name, s)
}

func (parser *JsonStructParser) RegisterRedexParser(name string, p JsonStructRedexParser) JsonStructRedexParser {
	oldParser := parser.redexParsers[name]
	if p == nil {
		delete(parser.redexParsers, name)
		return oldParser
	}
	parser.redexParsers[name] = p
	return oldParser
}

func (parser *JsonStructParser) RegisterDefaultRedexParser(p JsonStructRedexParser) JsonStructRedexParser {
	oldParser := parser.defaultRedexParser
	parser.defaultRedexParser = p
	return oldParser
}

func (parser *JsonStructParser) RegisterDataRedexName(name string) {
	if parser.DataRedexName != "" {
		delete(parser.redexParsers, parser.DataRedexName)
	}
	parser.DataRedexName = name
	parser.redexParsers[name] = parseData
}

func parseData(parser *JsonStructParser, name string, s interface{}) (Exp, error) {
	return parser.ParseData(s)
}
