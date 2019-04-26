package main

import (
	"flag"
	"fmt"

	"github.com/crcc/jsonp/engine"
	"github.com/crcc/jsonp/kernel"
	"github.com/crcc/jsonp/repl"
)

func main() {
	flag.Parse()

	interp := kernel.NewKernelInterpreter()
	loader := kernel.NewFileModuleLoader([]string{}, engine.ParserFunc(kernel.ParseJsonModule))
	eval := kernel.NewRepl(engine.ParserFunc(kernel.ParseJson), interp, loader)

	err := repl.Do(eval)
	if err != nil {
		fmt.Println(err.Error())
	}
}
