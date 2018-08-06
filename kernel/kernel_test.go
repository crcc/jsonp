package kernel

import (
	"testing"

	"github.com/crcc/jsonp/engine"
)

func interp(exp Exp) (Exp, error) {
	ctx := engine.NewContext(nil)
	env := engine.NewEnv(preludeModule.exportValues).Protect()
	return KernelInterpreter.Interpret(ctx, exp, env)
}

func mustParse(s string) Exp {
	e, err := parse(s)
	if err != nil {
		panic(err.Error())
	}
	return e
}

func TestInterpret_FactRec(t *testing.T) {
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
	exp := mustParse(jsonStr)

	val, err := interp(exp)
	if err != nil {
		t.Fatal(err.Error())
	}
	if !engine.NewNumber(120).Equal(val) {
		t.Fatalf("expect 120")
	}
}

func TestInterpret_FactIter(t *testing.T) {
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
	exp := mustParse(jsonStr)

	val, err := interp(exp)
	if err != nil {
		t.Fatal(err.Error())
	}
	if !engine.NewNumber(120).Equal(val) {
		t.Fatalf("expect 120")
	}
}

func TestInterpret_LoopForever(t *testing.T) {
	jsonStr := `
	{"begin": [
		{"def": {
		  "loop": {"func": [[],
					 ["printString", {"data": "hello!"}],
					 ["loop"]]}
		}},
		["loop"]
	]}`

	exp := mustParse(jsonStr)

	val, err := interp(exp)

	t.Log(err)
	t.Log(val)
}
