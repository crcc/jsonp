package lang

import (
	"testing"

	"github.com/crcc/jsonp/engine"
)

func interp(exp Exp) (Exp, error) {
	ctx := engine.NewContext(nil)
	env := InitEnv()
	return KernelInterpreter.Interpret(ctx, exp, env)
}

func Test1(t *testing.T) {
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

	t.Log(exp)
	val, err := interp(exp)

	t.Log(err)
	t.Log(val)
}

func Test2(t *testing.T) {

}

func Test3(t *testing.T) {
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

	t.Log(exp)
	val, err := interp(exp)

	t.Log(err)
	t.Log(val)
}
