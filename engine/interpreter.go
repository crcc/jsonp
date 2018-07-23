package engine

import (
	"errors"
	"fmt"
)

var (
	ErrUnhandledRedex          = errors.New("Unhandled Redex")
	ErrExposedCustomExp        = errors.New("Exposed Custom Exp")
	ErrCannotSuspendDelayedExp = errors.New("Cannot Suspend Delayed Exp")
)

type Interpreter interface {
	Interpret(ctx Context, exp Exp, env Env) (Exp, error)
	ExtractInfo(fromCtx, toCtx Context) error
}

type RedexInterpreter interface {
	RedexInterpret(ctx Context, interp Interpreter, exp Exp, env Env) (Exp, error)
}

type RedexInterpreterFunc func(ctx Context, interp Interpreter, exp Exp, env Env) (Exp, error)

func (f RedexInterpreterFunc) RedexInterpret(ctx Context, interp Interpreter, exp Exp, env Env) (Exp, error) {
	return f(ctx, interp, exp, env)
}

type ExtensibleInterpreter interface {
	Interpreter
	RegisterInterpreter(name string, interpreter RedexInterpreter) RedexInterpreter
	RegisterInfoExtracter(f func(fromCtx, toCtx Context) error) func(fromCtx, toCtx Context) error
}

func SuspendExp(exp Exp, suspend bool) error {
	switch exp.Kind() {
	case NullValue, BooleanValue, NumberValue, StringValue,
		ListValue, MapValue, CustomValue:
	case CustomExp:
		return ErrExposedCustomExp
	case DelayedExp:
		return ErrCannotSuspendDelayedExp
	case MapExp:
		m, _ := ToMap(exp)
		for _, subExp := range m {
			if err := SuspendExp(subExp, suspend); err != nil {
				return err
			}
		}
	case ListExp:
		l, _ := ToList(exp)
		for _, subExp := range l {
			if err := SuspendExp(subExp, suspend); err != nil {
				return err
			}
		}
	case ReducibleExp:
		r, _ := ToRedex(exp)
		if suspend {
			r.Suspend = SuspendAll
		} else {
			r.Suspend = NotSuspend
			if err := SuspendExp(r.Exp, suspend); err != nil {
				return err
			}
		}
	}

	return nil
}

// Identity
var IdInterpreter = IdentityInterpreter{}

type IdentityInterpreter struct{}

func (id IdentityInterpreter) Interpret(ctx Context, exp Exp, env Env) (Exp, error) {
	return exp, nil
}

func (id IdentityInterpreter) ExtractInfo(fromCtx, toCtx Context) error {
	return nil
}

// Application Order
func NewApplicationOrderInterpreter(fallback bool) ExtensibleInterpreter {
	return &ApplicationOrderInterpreter{
		redexInterpreters: make(map[string]RedexInterpreter),
		fallback:          fallback,
	}
}

type ApplicationOrderInterpreter struct {
	redexInterpreters map[string]RedexInterpreter
	extractInfo       func(fromCtx, toCtx Context) error
	fallback          bool
}

func (interp *ApplicationOrderInterpreter) interpretRedex(ctx Context, r Redex, env Env) (Exp, bool, error) {
	switch r.Suspend {
	case SuspendAll:
		return r, false, nil
	case Suspend:
		newExp, expanded, err := interp.interpret(ctx, r.Exp, env)
		if err != nil {
			return nil, false, err
		}

		newR := r
		newR.Exp = newExp

		return newR, expanded, nil
	case NotSuspend:
		redexInterp, ok := interp.redexInterpreters[r.Name]
		if !ok && !interp.fallback {
			return nil, false, ErrUnhandledRedex
		}

		newExp, expanded, err := interp.interpret(ctx, r.Exp, env)
		if err != nil {
			return nil, false, err
		}

		if !ok && interp.fallback {
			newR := r
			newR.Exp = newExp
			newR.Suspend = Suspend
			return newR, expanded, nil
		}
		newExp2, err := redexInterp.RedexInterpret(ctx, interp, newExp, env)
		if err != nil {
			return nil, false, err
		}
		return newExp2, true, nil
	default:
		panic(fmt.Sprintf("unknwon suspend type: %d", r.Suspend))
	}
}

func (interp *ApplicationOrderInterpreter) interpretMap(ctx Context, m Map, env Env) (Exp, bool, error) {
	newM := make(map[string]Exp, len(m))
	hasExpanded := false
	for key, subExp := range m {
		newExp, expanded, err := interp.interpret(ctx, subExp, env)
		if err != nil {
			return nil, false, err
		}
		newM[key] = newExp
		hasExpanded = hasExpanded || expanded
	}
	return NewMap(newM), hasExpanded, nil
}

func (interp *ApplicationOrderInterpreter) interpretList(ctx Context, l List, env Env) (Exp, bool, error) {
	newL := make([]Exp, len(l))
	hasExpanded := false
	for i, subExp := range l {
		newExp, expanded, err := interp.interpret(ctx, subExp, env)
		if err != nil {
			return nil, false, err
		}
		newL[i] = newExp
		hasExpanded = hasExpanded || expanded
	}
	return NewList(newL), hasExpanded, nil
}

// bool: expanded
func (interp *ApplicationOrderInterpreter) interpret(ctx Context, exp Exp, env Env) (Exp, bool, error) {
	e := exp
	expanded := true
	for expanded {
		var interpErr error
		switch e.Kind() {
		case NullValue, BooleanValue, NumberValue, StringValue,
			ListValue, MapValue, CustomValue:
			return e, false, nil
			// return interp.interpreter.Interpret(ctx, exp, env)
		case CustomExp:
			return nil, false, ErrExposedCustomExp
		case MapExp:
			m, err := ToMap(e)
			if err != nil {
				return nil, false, err
			}
			e, expanded, interpErr = interp.interpretMap(ctx, m, env)
		case ListExp:
			l, err := ToList(e)
			if err != nil {
				return nil, false, err
			}
			e, expanded, interpErr = interp.interpretList(ctx, l, env)
		case DelayedExp:
			d, err := ToDelayedExp(e)
			if err != nil {
				return nil, false, err
			}
			e, expanded, interpErr = interp.interpret(d.Context, d.Exp, d.Env)
		case ReducibleExp:
			r, err := ToRedex(e)
			if err != nil {
				return nil, false, err
			}
			e, expanded, interpErr = interp.interpretRedex(ctx, r, env)
		default:
			panic(fmt.Sprintf("unknown Exp Kind: %v, exp: %s", exp.Kind(), exp.String()))
		}

		if interpErr != nil {
			return nil, false, interpErr
		}
	}

	return e, false, nil
}

func (interp *ApplicationOrderInterpreter) Interpret(ctx Context, exp Exp, env Env) (Exp, error) {
	newExp, _, err := interp.interpret(ctx, exp, env)
	return newExp, err
}

func (interp *ApplicationOrderInterpreter) ExtractInfo(fromCtx Context, toCtx Context) error {
	if interp.extractInfo == nil {
		return nil
	}
	return interp.extractInfo(fromCtx, toCtx)
}

func (interp *ApplicationOrderInterpreter) RegisterInterpreter(name string, interpreter RedexInterpreter) RedexInterpreter {
	oldInterp := interp.redexInterpreters[name]
	if interpreter == nil {
		delete(interp.redexInterpreters, name)
		return oldInterp
	}
	interp.redexInterpreters[name] = interpreter
	return oldInterp
}

func (interp *ApplicationOrderInterpreter) RegisterInfoExtracter(f func(fromCtx, toCtx Context) error) func(fromCtx, toCtx Context) error {
	oldExtractInfo := interp.extractInfo
	interp.extractInfo = f
	return oldExtractInfo
}

// Normal Order
func NewNormalOrderInterpreter(fallback bool) ExtensibleInterpreter {
	return &NormalOrderInterpreter{
		redexInterpreters: make(map[string]RedexInterpreter),
		fallback:          fallback,
	}
}

type NormalOrderInterpreter struct {
	redexInterpreters map[string]RedexInterpreter
	extractInfo       func(fromCtx, toCtx Context) error
	fallback          bool
}

func (interp *NormalOrderInterpreter) interpretRedex(ctx Context, r Redex, env Env) (Exp, bool, error) {
	switch r.Suspend {
	case SuspendAll:
		return r, false, nil
	case Suspend:
		newExp, expanded, err := interp.interpret(ctx, r.Exp, env)
		if err != nil {
			return nil, false, err
		}

		newR := r
		newR.Exp = newExp

		return newR, expanded, nil
	case NotSuspend:
		redexInterp, ok := interp.redexInterpreters[r.Name]
		if !ok {
			if interp.fallback {
				newR := r
				newR.Suspend = SuspendAll
				return newR, false, nil
			}
			return nil, false, ErrUnhandledRedex
		}

		newExp, err := redexInterp.RedexInterpret(ctx, interp, r.Exp, env)
		if err != nil {
			return nil, false, err
		}
		return newExp, true, nil
	default:
		panic(fmt.Sprintf("unknwon suspend type: %d", r.Suspend))
	}
}

func (interp *NormalOrderInterpreter) interpretMap(ctx Context, m Map, env Env) (Exp, bool, error) {
	newM := make(map[string]Exp, len(m))
	hasExpanded := false
	for key, subExp := range m {
		newExp, expanded, err := interp.interpret(ctx, subExp, env)
		if err != nil {
			return nil, false, err
		}
		newM[key] = newExp
		hasExpanded = hasExpanded || expanded
	}
	return NewMap(newM), hasExpanded, nil
}

func (interp *NormalOrderInterpreter) interpretList(ctx Context, l List, env Env) (Exp, bool, error) {
	newL := make([]Exp, len(l))
	hasExpanded := false
	for i, subExp := range l {
		newExp, expanded, err := interp.interpret(ctx, subExp, env)
		if err != nil {
			return nil, false, err
		}
		newL[i] = newExp
		hasExpanded = hasExpanded || expanded
	}
	return NewList(newL), hasExpanded, nil
}

// bool: expanded
func (interp *NormalOrderInterpreter) interpret(ctx Context, exp Exp, env Env) (Exp, bool, error) {
	e := exp
	expanded := true
	for expanded {
		fmt.Println("normal", e)
		var interpErr error
		switch e.Kind() {
		case NullValue, BooleanValue, NumberValue, StringValue,
			ListValue, MapValue, CustomValue:
			return e, false, nil
			// return interp.interpreter.Interpret(ctx, exp, env)
		case CustomExp:
			return nil, false, ErrExposedCustomExp
		case MapExp:
			m, err := ToMap(e)
			if err != nil {
				return nil, false, err
			}
			e, expanded, interpErr = interp.interpretMap(ctx, m, env)
		case ListExp:
			l, err := ToList(e)
			if err != nil {
				return nil, false, err
			}
			e, expanded, interpErr = interp.interpretList(ctx, l, env)
		case DelayedExp:
			d, err := ToDelayedExp(e)
			if err != nil {
				return nil, false, err
			}
			e, expanded, interpErr = interp.interpret(d.Context, d.Exp, d.Env)
		case ReducibleExp:
			r, err := ToRedex(e)
			if err != nil {
				return nil, false, err
			}
			e, expanded, interpErr = interp.interpretRedex(ctx, r, env)
		default:
			panic(fmt.Sprintf("unknown Exp Kind: %v, exp: %s", exp.Kind(), exp.String()))
		}

		if interpErr != nil {
			return nil, false, interpErr
		}
	}

	return e, false, nil
}

func (interp *NormalOrderInterpreter) Interpret(ctx Context, exp Exp, env Env) (Exp, error) {
	newExp, _, err := interp.interpret(ctx, exp, env)
	return newExp, err
}

func (interp *NormalOrderInterpreter) ExtractInfo(fromCtx Context, toCtx Context) error {
	if interp.extractInfo == nil {
		return nil
	}
	return interp.extractInfo(fromCtx, toCtx)
}

func (interp *NormalOrderInterpreter) RegisterInterpreter(name string, interpreter RedexInterpreter) RedexInterpreter {
	oldInterp := interp.redexInterpreters[name]
	if interpreter == nil {
		delete(interp.redexInterpreters, name)
		return oldInterp
	}
	interp.redexInterpreters[name] = interpreter
	return oldInterp
}

func (interp *NormalOrderInterpreter) RegisterInfoExtracter(f func(fromCtx, toCtx Context) error) func(fromCtx, toCtx Context) error {
	oldExtractInfo := interp.extractInfo
	interp.extractInfo = f
	return oldExtractInfo
}

// Chain
func Chain(interps ...Interpreter) Interpreter {
	switch len(interps) {
	case 0:
		return IdInterpreter
	case 1:
		return interps[0]
	default:
		result := &ChainInterpreter{
			first:  interps[len(interps)-2],
			second: interps[len(interps)-1],
		}
		for i := len(interps) - 3; i >= 0; i-- {
			result = &ChainInterpreter{
				first:  interps[i],
				second: result,
			}
		}
		return result
	}
}

type ChainInterpreter struct {
	first  Interpreter
	second Interpreter
}

func (chain *ChainInterpreter) Interpret(ctx Context, exp Exp, env Env) (Exp, error) {
	// if exp.isValue() {
	// 	return exp, nil
	// }

	newCtx := ctx.Protect()
	newEnv := env.Protect()

	newExp, err := chain.first.Interpret(newCtx, exp, newEnv)
	if err != nil {
		return nil, err
	}

	if err := chain.first.ExtractInfo(newCtx, ctx); err != nil {
		return nil, err
	}

	if err := SuspendExp(newExp, false); err != nil {
		return nil, err
	}

	return chain.second.Interpret(ctx, newExp, env)
}

func (chain *ChainInterpreter) ExtractInfo(fromCtx, toCtx Context) error {
	return chain.second.ExtractInfo(fromCtx, toCtx)
}

/*
func WalkExpPost(exp Exp, f func(exp Exp)) {
	switch exp.Kind() {
	case NullValue, BooleanValue, NumberValue, StringValue, CustomValue:
		f(exp)
	case MapExp:
		m, _ := ToMap(exp)
		for _, subExp := range m {
			WalkExpPost(subExp, f)
		}
	case ListExp:
		l, _ := ToList(exp)
		for _, subExp := range l {
			WalkExpPost(subExp, f)
		}
	case ReducibleExp:
		r, _ := ToRedex(exp)
		WalkExpPost(r.Exp, f)
	}

	f(exp)
}

func WalkExpPre(exp Exp, f func(exp Exp)) {
	f(exp)

	switch exp.Kind() {
	case NullValue, BooleanValue, NumberValue, StringValue, CustomValue:
		// do nothing
	case MapExp:
		m, _ := ToMap(exp)
		for _, subExp := range m {
			WalkExpPre(subExp, f)
		}
	case ListExp:
		l, _ := ToList(exp)
		for _, subExp := range l {
			WalkExpPre(subExp, f)
		}
	case ReducibleExp:
		r, _ := ToRedex(exp)
		WalkExpPre(r.Exp, f)
	}
}

func WalkExpPrePost(exp Exp, pre, post func(Exp)) {
	pre(exp)

	switch exp.Kind() {
	case NullValue, BooleanValue, NumberValue, StringValue, CustomValue:
		// do nothing
	case MapExp:
		m, _ := ToMap(exp)
		for _, subExp := range m {
			WalkExpPrePost(subExp, pre, post)
		}
	case ListExp:
		l, _ := ToList(exp)
		for _, subExp := range l {
			WalkExpPrePost(subExp, pre, post)
		}
	case ReducibleExp:
		r, _ := ToRedex(exp)
		WalkExpPrePost(r.Exp, pre, post)
	}

	post(exp)
}

func WalkExp(exp Exp, f func(exp Exp, walk func(exp Exp))) {
	walk := func(exp Exp) {
		WalkExp(exp, f)
	}
	f(exp, walk)
}
*/
