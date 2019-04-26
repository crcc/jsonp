package kernel

import (
	"errors"
	"fmt"

	"github.com/crcc/jsonp/engine"
)

//// Custom Value
const (
	ClosureValue       engine.Kind = engine.CustomValue
	UninitializedValue engine.Kind = engine.CustomValue + 1
	PrimitiveFuncValue engine.Kind = engine.CustomValue + 2
	AmbiguousValue     engine.Kind = engine.CustomValue + 3
)

// Closure
type Closure struct {
	Args []string
	Body Exp
	Env  Env
}

func (clo Closure) Kind() engine.Kind {
	return ClosureValue
}

func (clo Closure) Equal(v Exp) bool {
	if v.Kind() != ClosureValue {
		return false
	}
	clo2 := v.(Closure)
	return &clo == &clo2
}

func (clo Closure) String() string {
	return fmt.Sprintf(`{"closure": %p}`, &clo.Body)
}

func NewClosure(args []string, body Exp, env Env) Closure {
	return Closure{
		Args: args,
		Body: body,
		Env:  env,
	}
}

var ErrNotClosureValue = errors.New("Not Closure Value")

func ToClosure(exp Exp) (Closure, error) {
	if exp.Kind() != ClosureValue {
		return Closure{}, ErrNotClosureValue
	}

	return exp.(Closure), nil
}

// Uninitialized Value
type UninitializedVal struct{}

func (u UninitializedVal) Kind() engine.Kind {
	return UninitializedValue
}

func (u UninitializedVal) Equal(v Exp) bool {
	return v.Kind() == UninitializedValue
}

func (u UninitializedVal) String() string {
	return `{"uninitializedValue": null}`
}

var uninitializedValue = UninitializedVal{}

func NewUninitializedValue() UninitializedVal {
	return uninitializedValue
}

func IsUninitializedValue(exp Exp) bool {
	return exp.Kind() == UninitializedValue
}

// Primitive Function
type PrimitiveFunc struct {
	Arity int
	Func  func(vals []Exp) (Exp, error)
}

func (p PrimitiveFunc) Kind() engine.Kind {
	return PrimitiveFuncValue
}

func (p PrimitiveFunc) Equal(exp Exp) bool {
	if exp.Kind() != PrimitiveFuncValue {
		return false
	}

	p2 := exp.(PrimitiveFunc)
	return &p == &p2
}

func (p PrimitiveFunc) String() string {
	return fmt.Sprintf(`{"primitiveFunc": %p}`, &p)
}

func NewPrimitive(arity int, f func(vals []Exp) (Exp, error)) Exp {
	return PrimitiveFunc{
		Arity: arity,
		Func:  f,
	}
}

var ErrNotPrimitiveFuncValue = errors.New("Not Primitive Function Value")

func ToPrimitive(exp Exp) (PrimitiveFunc, error) {
	if exp.Kind() != PrimitiveFuncValue {
		return PrimitiveFunc{}, ErrNotPrimitiveFuncValue
	}

	return exp.(PrimitiveFunc), nil
}

// Ambiguous Value
type AmbiguousVal struct{}

func (u AmbiguousVal) Kind() engine.Kind {
	return AmbiguousValue
}

func (u AmbiguousVal) Equal(v Exp) bool {
	return v.Kind() == AmbiguousValue
}

func (u AmbiguousVal) String() string {
	return `{"ambiguousValue": null}`
}

var ambiguousValue = AmbiguousVal{}

func NewAmbiguousValue() AmbiguousVal {
	return ambiguousValue
}

func IsAmbiguousValue(exp Exp) bool {
	return exp.Kind() == AmbiguousValue
}
