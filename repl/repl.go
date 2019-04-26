package repl

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/crcc/jsonp/engine"
)

// Repl

type Exp = engine.Exp

type Repl interface {
	EvalBatch(filename string) error
	Parse(r io.Reader) (Exp, error)
	EvalInteractive(exp Exp) (Exp, error)
	AddPaths(paths []string)
}

var ErrStopRepl = errors.New("Stop Repl")

var (
	interactive bool
	batchModule string
	findPaths   string
)

func init() {
	flag.BoolVar(&interactive, "i", false, "run in interative mode")
	flag.StringVar(&batchModule, "b", "", "evaluate file in batch mode")
	flag.StringVar(&findPaths, "p", "", "'path1,path2,...', tell me where to find modules")
}

func getFindPaths() ([]string, error) {
	if findPaths == "" {
		dir, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		return []string{dir}, nil
	}

	paths := strings.Split(findPaths, ",")

	for i, path := range paths {
		p, err := filepath.Abs(path)
		if err != nil {
			return nil, err
		}
		paths[i] = p
	}

	return paths, nil
}

// func isInteractive() bool {
// 	return interactive
// }

func isBatch() bool {
	return !interactive && batchModule != ""
}

func Do(e Repl) error {
	// process arguments
	if len(flag.Args()) > 1 {
		return fmt.Errorf("expect at most one filename")
	} else if len(flag.Args()) == 1 && batchModule != "" {
		batchModule = flag.Arg(0)
	}

	// add paths
	paths, err := getFindPaths()
	if err != nil {
		return err
	}
	e.AddPaths(paths)

	// eval in batch mode
	if isBatch() {
		return e.EvalBatch(batchModule)
	}

	// eval in interactive mode
	var (
		exp, val Exp
	)
	for {
		fmt.Print("> ")
		exp, err = e.Parse(os.Stdin)
		if err != nil {
			fmt.Printf("Error: %s\n", err.Error())
			continue
		}

		val, err = e.EvalInteractive(exp)
		if err != nil {
			if err == ErrStopRepl {
				return nil
			}
			fmt.Printf("Error: %s\n", err.Error())
			continue
		}
		fmt.Printf("Value: %s\n", val.String())
	}

	return nil
}
