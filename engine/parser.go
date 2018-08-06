package engine

import "io"

type Parser interface {
	Parse(ctx Context, r io.Reader) (Exp, error)
}

type ParserFunc func(ctx Context, r io.Reader) (Exp, error)

func (f ParserFunc) Parse(ctx Context, r io.Reader) (Exp, error) {
	return f(ctx, r)
}
