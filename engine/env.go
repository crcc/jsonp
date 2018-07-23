package engine

import (
	"errors"
	"fmt"
	"math/rand"
)

var ErrNameNotFound = errors.New("Name Not Found")

type Env interface {
	Get(name string) (Exp, error)
	Set(name string, val Exp) error
	Define(name string, val Exp)
	Extend(kvs map[string]Exp) Env

	Protect() Env
}

type env struct {
	namespace map[string]Exp
	parent    *env
}

func (e *env) findFrame(name string) (*env, Exp, error) {
	frame := e
	val, ok := frame.namespace[name]
	for !ok && frame.parent != nil {
		frame = frame.parent
		val, ok = frame.namespace[name]
	}

	if ok {
		return frame, val, nil
	}
	return nil, nil, ErrNameNotFound
}

func (e *env) Get(name string) (Exp, error) {
	_, val, err := e.findFrame(name)
	return val, err
}

func (e *env) Set(name string, val Exp) error {
	f, _, err := e.findFrame(name)
	if err != nil {
		return err
	}
	f.namespace[name] = val
	return nil
}

func (e *env) Define(name string, val Exp) {
	e.namespace[name] = val
}

func (e *env) Extend(kvs map[string]Exp) Env {
	if kvs == nil {
		kvs = make(map[string]Exp)
	}
	return &env{
		namespace: kvs,
		parent:    e,
	}
}

func (e *env) Protect() Env {
	newNs := make(map[string]Exp, len(e.namespace))
	for name, val := range e.namespace {
		newNs[name] = val
	}

	return &env{
		namespace: newNs,
		parent:    e.parent,
	}
}

func NewEnv(kvs map[string]Exp) Env {
	if kvs == nil {
		kvs = make(map[string]Exp)
	}

	return &env{
		namespace: kvs,
		parent:    nil,
	}
}

func FreshName(e Env) string {
	key := fmt.Sprintf("_GKey%d", rand.Uint64())

	for _, err := e.Get(key); err == nil; {
		key = fmt.Sprintf("_GKey%d", rand.Uint64())
		_, err = e.Get(key)
	}

	return key
}
