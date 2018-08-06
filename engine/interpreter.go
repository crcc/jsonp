package engine

import (
	"errors"
	"fmt"
)

var (
	ErrUnhandledRedex          = errors.New("Unhandled Redex")
	ErrCannotSuspendDelayedExp = errors.New("Cannot Suspend Delayed Exp")
)

type InfoExtracter interface {
	ExtractInfo(fromCtx, toCtx Context, fromEnv, toEnv Env) error
}

type InfoExtracterFunc func(ctx Context, env Env) (Context, Env, error)

func (f InfoExtracterFunc) ExtractInfo(ctx Context, env Env) (Context, Env, error) {
	return f(ctx, env)
}

type Interpreter interface {
	Interpret(ctx Context, exp Exp, env Env) (Exp, error)
	InfoExtracter
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
	RegisterInfoExtracter(extracter InfoExtracter) InfoExtracter
}

func Suspend(exp Exp, suspend bool) (Exp, error) {
	switch exp.Kind() {
	case DelayedExp:
		return nil, ErrCannotSuspendDelayedExp
	case SuspendValue:
		if !suspend {
			s, err := ToSuspendValue(exp)
			if err != nil {
				return nil, err
			}
			return UnsuspendValue(s), nil
		}
		return exp, nil
	case SuspendExp:
		s, err := ToSuspendExp(exp)
		if err != nil {
			return nil, err
		}
		r := UnsuspendExp(s)
		if suspend {
			return NewSuspendValue(r), nil
		} else {
			return r, nil
		}
	case MapExp:
		m, err := ToMap(exp)
		if err != nil {
			return nil, err
		}
		for key, subExp := range m {
			newExp, err := Suspend(subExp, suspend)
			if err != nil {
				return nil, err
			}
			m[key] = newExp
		}
		return NewMapExp(m), nil
	case ListExp:
		l, err := ToList(exp)
		if err != nil {
			return nil, err
		}
		for i, subExp := range l {
			newExp, err := Suspend(subExp, suspend)
			if err != nil {
				return nil, err
			}
			l[i] = newExp
		}
		return NewListExp(l), nil
	case ReducibleExp:
		if suspend {
			r, err := ToRedex(exp)
			if err != nil {
				return nil, err
			}
			return NewSuspendValue(r), nil
		} else {
			return exp, nil
		}
	default:
		return exp, nil
	}
}

// Identity
var IdInterpreter = IdentityInterpreter{}

type IdentityInterpreter struct{}

func (id IdentityInterpreter) Interpret(ctx Context, exp Exp, env Env) (Exp, error) {
	return exp, nil
}

func (id IdentityInterpreter) ExtractInfo(fromCtx, toCtx Context, fromEnv, toEnv Env) error {
	return nil
}

// Abstract

type InnerInterpreter func(ctx Context, exp Exp, env Env) (Exp, bool, error)

type RedexEvaluator interface {
	InterpretRedex(redexInterp RedexInterpreter, interp InnerInterpreter, ctx Context, r Redex, env Env) (Exp, bool, error)
}

type RedexEvaluatorFunc func(interp RedexInterpreter, ctx Context, r Redex, env Env) (Exp, bool, error)

func (f RedexEvaluatorFunc) InterpretRedex(interp RedexInterpreter, ctx Context, r Redex, env Env) (Exp, bool, error) {
	return f(interp, ctx, r, env)
}

func NewAbstractInterpreter(eval RedexEvaluator) AbstractInterpreter {
	return AbstractInterpreter{
		redexInterpreters: make(map[string]RedexInterpreter),
		redexEvaluator:    eval,
	}
}

type AbstractInterpreter struct {
	redexInterpreters map[string]RedexInterpreter
	extracter         InfoExtracter
	redexEvaluator    RedexEvaluator
}

func (interp *AbstractInterpreter) interpretMap(ctx Context, m Map, env Env) (Exp, bool, error) {
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

func (interp *AbstractInterpreter) interpretList(ctx Context, l List, env Env) (Exp, bool, error) {
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

func (interp *AbstractInterpreter) interpretSuspendExp(ctx Context, s SuspendEx, env Env) (Exp, bool, error) {
	newExp, expanded, err := interp.interpret(ctx, s.Exp, env)
	if err != nil {
		return nil, false, err
	}

	newS := s
	newS.Exp = newExp

	return newS, expanded, nil
}

// bool: expanded
func (interp *AbstractInterpreter) interpret(ctx Context, exp Exp, env Env) (Exp, bool, error) {
	e := exp
	expanded := true
	for expanded {
		// fmt.Println("interpret", e)
		var interpErr error
		switch e.Kind() {
		case NullValue, BooleanValue, NumberValue, StringValue,
			ListValue, MapValue, SuspendValue:
			return e, false, nil
			// return interp.interpreter.Interpret(ctx, exp, env)
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
		case SuspendExp:
			s, err := ToSuspendExp(e)
			if err != nil {
				return nil, false, err
			}
			e, expanded, interpErr = interp.interpretSuspendExp(ctx, s, env)
		case DelayedExp:
			d, err := ToDelayedExp(e)
			if err != nil {
				return nil, false, err
			}
			ctx, e, env = d.Context, d.Exp, d.Env
			expanded, interpErr = true, nil
		case ReducibleExp:
			r, err := ToRedex(e)
			if err != nil {
				return nil, false, err
			}
			redexInterp := interp.redexInterpreters[r.Name]
			e, expanded, interpErr = interp.redexEvaluator.InterpretRedex(redexInterp, interp.interpret, ctx, r, env)
		default:
			panic(fmt.Sprintf("unknown Exp Kind: %v, exp: %s", exp.Kind(), exp.String()))
		}

		if interpErr != nil {
			return nil, false, interpErr
		}
	}

	return e, false, nil
}

func (interp *AbstractInterpreter) Interpret(ctx Context, exp Exp, env Env) (Exp, error) {
	newExp, _, err := interp.interpret(ctx, exp, env)
	return newExp, err
}

func (interp *AbstractInterpreter) ExtractInfo(fromCtx, toCtx Context, fromEnv, toEnv Env) error {
	if interp.extracter == nil {
		return nil
	}
	return interp.extracter.ExtractInfo(fromCtx, toCtx, fromEnv, toEnv)
}

func (interp *AbstractInterpreter) RegisterInterpreter(name string, interpreter RedexInterpreter) RedexInterpreter {
	oldInterp := interp.redexInterpreters[name]
	if interpreter == nil {
		delete(interp.redexInterpreters, name)
		return oldInterp
	}
	interp.redexInterpreters[name] = interpreter
	return oldInterp
}

func (interp *AbstractInterpreter) RegisterInfoExtracter(extracter InfoExtracter) InfoExtracter {
	oldExtracter := interp.extracter
	interp.extracter = extracter
	return oldExtracter
}

// Application Order
func NewApplicationOrderInterpreter(fallback bool) ExtensibleInterpreter {
	result := &ApplicationOrderInterpreter{
		fallback: fallback,
	}

	result.AbstractInterpreter = NewAbstractInterpreter(result)
	return result
}

type ApplicationOrderInterpreter struct {
	AbstractInterpreter
	fallback bool
}

func (interp *ApplicationOrderInterpreter) InterpretRedex(redexInterp RedexInterpreter, interpret InnerInterpreter, ctx Context, r Redex, env Env) (Exp, bool, error) {
	if redexInterp == nil && !interp.fallback {
		return nil, false, ErrUnhandledRedex
	}

	newExp, expanded, err := interpret(ctx, r.Exp, env)
	if err != nil {
		return nil, false, err
	}

	if redexInterp == nil && interp.fallback {
		newR := r
		newR.Exp = newExp
		return NewSuspendValue(newR), expanded, nil
	}
	newExp2, err := redexInterp.RedexInterpret(ctx, interp, newExp, env)
	if err != nil {
		return nil, false, err
	}
	return newExp2, true, nil
}

// Normal Order
func NewNormalOrderInterpreter(fallback bool) ExtensibleInterpreter {
	result := &NormalOrderInterpreter{
		fallback: fallback,
	}
	result.AbstractInterpreter = NewAbstractInterpreter(result)
	return result
}

type NormalOrderInterpreter struct {
	AbstractInterpreter
	fallback bool
}

func (interp *NormalOrderInterpreter) InterpretRedex(redexInterp RedexInterpreter, interpret InnerInterpreter, ctx Context, r Redex, env Env) (Exp, bool, error) {
	if redexInterp == nil {
		if interp.fallback {
			return NewSuspendValue(r), false, nil
		}
		return nil, false, ErrUnhandledRedex
	}

	newExp, err := redexInterp.RedexInterpret(ctx, interp, r.Exp, env)
	if err != nil {
		return nil, false, err
	}
	return newExp, true, nil
}

// Layered

func NewLayeredInterpreter(interp Interpreter) ExtensibleInterpreter {
	result := &LayeredInterpreter{
		innerInterpreter: interp,
	}
	result.AbstractInterpreter = NewAbstractInterpreter(result)
	return result
}

type LayeredInterpreter struct {
	AbstractInterpreter
	innerInterpreter Interpreter
}

func (interp *LayeredInterpreter) InterpretRedex(redexInterp RedexInterpreter, interpret InnerInterpreter, ctx Context, r Redex, env Env) (Exp, bool, error) {
	if redexInterp == nil {
		newExp, err := interp.innerInterpreter.Interpret(ctx, r, env)
		if err != nil {
			return nil, false, err
		}
		return newExp, false, nil
	}

	newExp, err := redexInterp.RedexInterpret(ctx, interp, r.Exp, env)
	if err != nil {
		return nil, false, err
	}
	return newExp, true, nil
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

// context, environment: copy, use, extract into old
func (chain *ChainInterpreter) Interpret(ctx Context, exp Exp, env Env) (Exp, error) {
	// if exp.isValue() {
	// 	return exp, nil
	// }

	newCtx := ctx.Protect()
	newEnv := env.Protect()

	newVal, err := chain.first.Interpret(newCtx, exp, newEnv)
	if err != nil {
		return nil, err
	}

	if err := chain.first.ExtractInfo(newCtx, ctx, newEnv, env); err != nil {
		return nil, err
	}

	newExp, err := Suspend(newVal, false)
	if err != nil {
		return nil, err
	}

	return chain.second.Interpret(ctx, newExp, env)
}

func (chain *ChainInterpreter) ExtractInfo(fromCtx, toCtx Context, fromEnv, toEnv Env) error {
	return chain.second.ExtractInfo(fromCtx, toCtx, fromEnv, toEnv)
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
