package kernel

import (
	"strings"
	"testing"

	"github.com/crcc/jsonp/engine"
)

func parse(s string) (Exp, error) {
	ctx := engine.NewContext(nil)
	return ParseJson(ctx, strings.NewReader(s))
}

func TestParse_FactRec(t *testing.T) {
	jsonStr := `
	{"begin": [
		{"def": {
		  "fact": {"func": [["n"],
					 {"if": [["<=", "n", 0],
							 1,
							 ["*", "n", ["fact", ["-", "n", 1]]]]}
				  ]}
		}},
		["fact", 5]
	]}`
	e, err := parse(jsonStr)
	if err != nil {
		t.Fatal(err.Error())
	}

	exp := engine.NewRedex("begin",
		engine.NewListExp([]Exp{
			engine.NewRedex("def",
				engine.NewMapExp(map[string]Exp{
					"fact": engine.NewRedex("func",
						engine.NewListExp([]Exp{
							engine.NewListExp([]Exp{engine.NewString("n")}),
							engine.NewRedex("if", engine.NewListExp([]Exp{
								engine.NewRedex("apply", engine.NewListExp([]Exp{
									engine.NewRedex("var", engine.NewString("<=")),
									engine.NewRedex("var", engine.NewString("n")),
									engine.NewNumber(0),
								})),
								engine.NewNumber(1),
								engine.NewRedex("apply", engine.NewListExp([]Exp{
									engine.NewRedex("var", engine.NewString("*")),
									engine.NewRedex("var", engine.NewString("n")),
									engine.NewRedex("apply", engine.NewListExp([]Exp{
										engine.NewRedex("var", engine.NewString("fact")),
										engine.NewRedex("apply", engine.NewListExp([]Exp{
											engine.NewRedex("var", engine.NewString("-")),
											engine.NewRedex("var", engine.NewString("n")),
											engine.NewNumber(1),
										})),
									})),
								})),
							})),
						})),
				})),
			engine.NewRedex("apply",
				engine.NewListExp([]Exp{
					engine.NewRedex("var", engine.NewString("fact")),
					engine.NewNumber(5),
				})),
		}))

	if !exp.Equal(e) {
		t.Fatalf("expect %s, but found %s", exp.String(), e.String())
	}
}

func TestParse_FactIter(t *testing.T) {
	jsonStr := `
	{"begin": [
		{"def": {
		  "fact": {"func": [["n", "a"],
					 {"if": [["<=", "n", 0],
							 "a",
							 ["fact", ["-", "n", 1], ["*", "n", "a"]]]}
				  ]}
		}},
		["fact", 5, 1]
	]}`
	e, err := parse(jsonStr)
	if err != nil {
		t.Fatal(err.Error())
	}

	exp := engine.NewRedex("begin",
		engine.NewListExp([]Exp{
			engine.NewRedex("def",
				engine.NewMapExp(map[string]Exp{
					"fact": engine.NewRedex("func",
						engine.NewListExp([]Exp{
							engine.NewListExp([]Exp{engine.NewString("n"), engine.NewString("a")}),
							engine.NewRedex("if", engine.NewListExp([]Exp{
								engine.NewRedex("apply", engine.NewListExp([]Exp{
									engine.NewRedex("var", engine.NewString("<=")),
									engine.NewRedex("var", engine.NewString("n")),
									engine.NewNumber(0),
								})),
								engine.NewRedex("var", engine.NewString("a")),
								engine.NewRedex("apply", engine.NewListExp([]Exp{
									engine.NewRedex("var", engine.NewString("fact")),
									engine.NewRedex("apply", engine.NewListExp([]Exp{
										engine.NewRedex("var", engine.NewString("-")),
										engine.NewRedex("var", engine.NewString("n")),
										engine.NewNumber(1),
									})),
									engine.NewRedex("apply", engine.NewListExp([]Exp{
										engine.NewRedex("var", engine.NewString("*")),
										engine.NewRedex("var", engine.NewString("n")),
										engine.NewRedex("var", engine.NewString("a")),
									})),
								})),
							})),
						})),
				})),
			engine.NewRedex("apply",
				engine.NewListExp([]Exp{
					engine.NewRedex("var", engine.NewString("fact")),
					engine.NewNumber(5),
					engine.NewNumber(1),
				})),
		}))

	if !exp.Equal(e) {
		t.Fatalf("expect %s, but found %s", exp.String(), e.String())
	}
}

func TestParse_LoopForever(t *testing.T) {
	jsonStr := `
	{"begin": [
		{"def": {
		  "loop": {"func": [[],
					 ["printString", {"data": "hello!"}],
					 ["loop"]]}
		}},
		["loop"]
	]}`

	e, err := parse(jsonStr)
	if err != nil {
		t.Fatal(err.Error())
	}

	exp := engine.NewRedex("begin",
		engine.NewListExp([]Exp{
			engine.NewRedex("def",
				engine.NewMapExp(map[string]Exp{
					"loop": engine.NewRedex("func",
						engine.NewListExp([]Exp{
							engine.NewListExp([]Exp{}),
							engine.NewRedex("begin", engine.NewListExp([]Exp{
								engine.NewRedex("apply", engine.NewListExp([]Exp{
									engine.NewRedex("var", engine.NewString("printString")),
									engine.NewString("hello!"),
								})),
								engine.NewRedex("apply", engine.NewListExp([]Exp{
									engine.NewRedex("var", engine.NewString("loop")),
								})),
							})),
						})),
				})),
			engine.NewRedex("apply",
				engine.NewListExp([]Exp{
					engine.NewRedex("var", engine.NewString("loop")),
				})),
		}))

	if !exp.Equal(e) {
		t.Fatalf("expect %s, but found %s", exp.String(), e.String())
	}
}
