package engine

import "io"

type Parser interface {
	Parse(ctx Context, r io.Reader) (Exp, error)
}
