package kernel

import (
	"os"
	"strings"
	"testing"

	"github.com/crcc/jsonp/engine"
)

var (
	evalS *Repl
	evalF *Repl
)

func init() {
	loaderS := &SimpleModuleLoader{
		Modules: map[string]Exp{
			"fact": mustNewModule("fact", `
			{"def": {
				"factRec": {"func": [["n"],
						   {"if": [["<=", "n", 0],
								   1,
								   ["*", "n", ["factRec", ["-", "n", 1]]]]}
					]},
				"factIter": {"func": [["n", "a"],
					{"if": [["<=", "n", 0],
							"a",
							["factIter", ["-", "n", 1], ["*", "n", "a"]]]}
					]}
			}}
			
			{"export": ["factRec", "factIter"]}`),
			"fact2": mustNewModule("fact2", `
			{"import": {"fact": ["factIter"]}}

			{"def": {"fact": {"func": [["n"], ["factIter", "n", 1]]}}}

			{"export": ["fact"]}`),
			"main": mustNewModule("main", `
				{"import": {"fact2": ["fact"]}}
				["print", ["fact", 6]]
			`),
			"main2": mustNewModule("main2", `
				{"import": {"fact2": ["fact"], "fact": ["factRec"]}}
				["print", ["+", ["fact", 6], ["factRec", 5]]]
			`),
		},
	}
	interp := NewKernelInterpreter()
	evalS = NewRepl(engine.ParserFunc(ParseJson), interp, loaderS)

	dir, err := os.Getwd()
	if err != nil {
		panic(err.Error())
	}
	loaderF := NewFileModuleLoader([]string{dir + string(os.PathSeparator) + "test"},
		engine.ParserFunc(ParseJsonModule))
	evalF = NewRepl(engine.ParserFunc(ParseJson), interp, loaderF)
}

func mustNewModule(name string, jsonStr string) Exp {
	ctx := engine.NewContext(map[string]interface{}{
		"module-name": name,
	})
	m, err := ParseJsonModule(ctx, strings.NewReader(jsonStr))
	if err != nil {
		panic(err.Error())
	}

	return m
}

func interp(exp Exp) (Exp, error) {
	return evalS.EvalInteractive(exp)
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
	t.Skip("loop forever, ignore it")
	jsonStr := `
	{"begin": [
		{"def": {
		  "loop": {"func": [[],
					 ["print", {"data": "hello!"}],
					 ["loop"]]}
		}},
		["loop"]
	]}`

	exp := mustParse(jsonStr)

	val, err := interp(exp)

	t.Log(err)
	t.Log(val)
}

func TestModule_Simple_TopLevel(t *testing.T) {
	e, err := parse(`{"begin": [
		{"import": {"fact2": ["fact"]}},
		["fact", 6]
	]}`)
	if err != nil {
		t.Fatal(err)
	}

	val, err := evalS.EvalInteractive(e)
	if err != nil {
		t.Fatal(err.Error())
	}
	if !engine.NewNumber(720).Equal(val) {
		t.Fatalf("expect 720")
	}

	e2, err := parse(`{"begin": [
		{"import": {"fact2": ["fact"], "fact": ["factRec"]}},
		["+", ["fact", 6], ["factRec", 5]]
	]}`)
	if err != nil {
		t.Fatal(err)
	}

	val, err = evalS.EvalInteractive(e2)
	if err != nil {
		t.Fatal(err.Error())
	}
	if !engine.NewNumber(840).Equal(val) {
		t.Fatalf("expect 840")
	}
}

func TestModule_Simple_ModuleLevel(t *testing.T) {
	err := evalS.EvalBatch("main")
	if err != nil {
		t.Fatal(err.Error())
	}

	err = evalS.EvalBatch("main2")
	if err != nil {
		t.Fatal(err.Error())
	}
}

func TestModule_File_TopLevel(t *testing.T) {
	e, err := parse(`{"begin": [
		{"import": {"fact2": ["fact"]}},
		["fact", 6]
	]}`)
	if err != nil {
		t.Fatal(err)
	}

	val, err := evalF.EvalInteractive(e)
	if err != nil {
		t.Fatal(err.Error())
	}
	if !engine.NewNumber(720).Equal(val) {
		t.Fatalf("expect 720")
	}

	e2, err := parse(`{"begin": [
		{"import": {"fact2": ["fact"], "fact": ["factRec"]}},
		["+", ["fact", 6], ["factRec", 5]]
	]}`)
	if err != nil {
		t.Fatal(err)
	}

	val, err = evalS.EvalInteractive(e2)
	if err != nil {
		t.Fatal(err.Error())
	}
	if !engine.NewNumber(840).Equal(val) {
		t.Fatalf("expect 840")
	}
}

func TestModule_File_ModuleLevel(t *testing.T) {
	err := evalF.EvalBatch("test")
	if err != nil {
		t.Fatal(err.Error())
	}

	err = evalF.EvalBatch("test2")
	if err != nil {
		t.Fatal(err.Error())
	}
}
