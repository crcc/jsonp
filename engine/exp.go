package engine

import (
	"errors"
	"fmt"
	"strings"
)

// Value
type Kind uint8

const (
	NullValue Kind = iota
	BooleanValue
	NumberValue
	StringValue
	ListValue
	MapValue

	SuspendValue

	MapExp
	ListExp
	ReducibleExp

	SuspendExp
	DelayedExp
)

type Exp interface {
	Kind() Kind
	Equal(v Exp) bool
	String() string
}

// Null
type Null struct{}

func (n Null) Kind() Kind {
	return NullValue
}

func (n Null) Equal(v Exp) bool {
	return v.Kind() == NullValue
}

func (n Null) String() string {
	return "null"
}

var nullVal = Null{}

func NewNull() Null {
	return nullVal
}

func IsNull(v Exp) bool {
	return v.Kind() == NullValue
}

// Boolean
type Boolean bool

func (b Boolean) Kind() Kind {
	return BooleanValue
}

func (b Boolean) Equal(v Exp) bool {
	return v.Kind() == BooleanValue && v.(Boolean) == b
}

func (b Boolean) String() string {
	return fmt.Sprint(bool(b))
}

func NewBoolean(b bool) Boolean {
	return Boolean(b)
}

var ErrNotBooleanValue = errors.New("Not Boolean Value")

func ToBoolean(v Exp) (bool, error) {
	if v.Kind() != BooleanValue {
		return false, ErrNotBooleanValue
	}

	return bool(v.(Boolean)), nil
}

// Number
type Number float64

func (n Number) Kind() Kind {
	return NumberValue
}

func (n Number) Equal(v Exp) bool {
	return v.Kind() == NumberValue && v.(Number) == n
}

func (n Number) String() string {
	return fmt.Sprint(float64(n))
}

func NewNumber(n float64) Number {
	return Number(n)
}

var ErrNotNumberValue = errors.New("Not Number Value")

func ToNumber(v Exp) (float64, error) {
	if v.Kind() != NumberValue {
		return 0.0, ErrNotNumberValue
	}

	return float64(v.(Number)), nil
}

// String
type String string

func (s String) Kind() Kind {
	return StringValue
}

func (s String) Equal(v Exp) bool {
	return v.Kind() == StringValue && v.(String) == s
}

func (s String) String() string {
	return string(s)
}

func NewString(s string) String {
	return String(s)
}

var ErrNotStringValue = errors.New("Not String Value")

func ToString(v Exp) (string, error) {
	if v.Kind() != StringValue {
		return "", ErrNotStringValue
	}
	return string(v.(String)), nil
}

// MapValue
type Map map[string]Exp

func (m Map) Kind() Kind {
	return MapValue
}

func (m Map) Equal(v Exp) bool {
	return v.Kind() == MapValue && EqualMap(m, v.(Map))
}

func (m Map) String() string {
	strs := make([]string, 0, len(m))
	for name, val := range m {
		strs = append(strs, fmt.Sprintf("%q: %s", name, val.String()))
	}
	return fmt.Sprintf("{%s}", strings.Join(strs, ", "))
}

func NewMap(m map[string]Exp) Map {
	return Map(m)
}

var ErrNotMapValue = errors.New("Not Map Value")

func ToMap(v Exp) (map[string]Exp, error) {
	if v.Kind() != MapValue {
		return nil, ErrNotMapValue
	}

	return v.(Map), nil
}

// MapExp
type MapEx map[string]Exp

func (m MapEx) Kind() Kind {
	return MapExp
}

func EqualMap(m, m2 map[string]Exp) bool {
	if len(m) != len(m2) {
		return false
	}

	for name, val := range m {
		val2, ok := m2[name]
		if !ok {
			return false
		}
		if !val.Equal(val2) {
			return false
		}
	}

	return true
}

func (m MapEx) Equal(v Exp) bool {
	return v.Kind() == MapExp && EqualMap(m, v.(MapEx))
}

func (m MapEx) String() string {
	strs := make([]string, 0, len(m))
	for name, val := range m {
		strs = append(strs, fmt.Sprintf("%q: %s", name, val.String()))
	}
	return fmt.Sprintf("{%s}", strings.Join(strs, ", "))
}

func NewMapExp(m map[string]Exp) MapEx {
	return MapEx(m)
}

var ErrNotMapExp = errors.New("Not Map Exp")

func ToMapExp(v Exp) (map[string]Exp, error) {
	if v.Kind() != MapExp {
		return nil, ErrNotMapExp
	}

	return v.(MapEx), nil
}

// ListValue
type List []Exp

func (l List) Kind() Kind {
	return ListValue
}

func (l List) Equal(v Exp) bool {
	return v.Kind() == ListValue && EqualList(l, v.(List))
}

func (l List) String() string {
	strs := make([]string, len(l))
	for i, val := range l {
		strs[i] = val.String()
	}
	return fmt.Sprintf("[%s]", strings.Join(strs, ", "))
}

func NewList(list []Exp) List {
	return List(list)
}

var ErrNotListValue = errors.New("Not List Value")

func ToList(v Exp) ([]Exp, error) {
	if v.Kind() != ListValue {
		return nil, ErrNotListValue
	}

	return v.(List), nil
}

// ListExp
type ListEx []Exp

func (l ListEx) Kind() Kind {
	return ListExp
}

func EqualList(l, l2 []Exp) bool {
	if len(l) != len(l2) {
		return false
	}

	for i, val := range l {
		if !val.Equal(l2[i]) {
			return false
		}
	}

	return true
}

func (l ListEx) Equal(v Exp) bool {
	return v.Kind() == ListExp && EqualList(l, v.(ListEx))
}

func (l ListEx) String() string {
	strs := make([]string, len(l))
	for i, val := range l {
		strs[i] = val.String()
	}
	return fmt.Sprintf("[%s]", strings.Join(strs, ", "))
}

func NewListExp(list []Exp) ListEx {
	return ListEx(list)
}

var ErrNotListExp = errors.New("Not List Exp")

func ToListExp(v Exp) ([]Exp, error) {
	if v.Kind() != ListExp {
		return nil, ErrNotListExp
	}

	return v.(ListEx), nil
}

// SuspendValue
type SuspendVal Redex

func (s SuspendVal) Kind() Kind {
	return SuspendValue
}

func (s SuspendVal) Equal(v Exp) bool {
	return v.Kind() == SuspendValue && EqualRedex(Redex(s), Redex(v.(SuspendVal)))
}

func (s SuspendVal) String() string {
	return fmt.Sprintf(`{"data": %s}`, Redex(s).String())
}

func NewSuspendValue(r Redex) SuspendVal {
	return SuspendVal(r)
}

var ErrNotSuspendValue = errors.New("Not Suspend Value")

func ToSuspendValue(v Exp) (SuspendVal, error) {
	if v.Kind() != SuspendValue {
		return SuspendVal{}, ErrNotSuspendValue
	}

	return v.(SuspendVal), nil
}

func UnsuspendValue(s SuspendVal) Redex {
	return Redex(s)
}

// Redex
type Redex struct {
	Name string
	Exp  Exp
}

func (r Redex) Kind() Kind {
	return ReducibleExp
}

func EqualRedex(r, r2 Redex) bool {
	return r.Name == r2.Name && r.Exp.Equal(r2.Exp)
}

func (r Redex) Equal(v Exp) bool {
	return v.Kind() == ReducibleExp && EqualRedex(r, v.(Redex))
}

func (r Redex) String() string {
	return fmt.Sprintf(`{%q: %s}`, r.Name, r.Exp.String())
}

func NewRedex(name string, exp Exp) Redex {
	return Redex{
		Name: name,
		Exp:  exp,
	}
}

var ErrNotReducibleExp = errors.New("Not Reducible Exp")

func ToRedex(v Exp) (Redex, error) {
	if v.Kind() != ReducibleExp {
		return Redex{}, ErrNotReducibleExp
	}

	return v.(Redex), nil
}

// DelayedExp
type DelayedEx struct {
	Context Context
	Exp     Exp
	Env     Env
}

func (d DelayedEx) Kind() Kind {
	return DelayedExp
}

func (d DelayedEx) Equal(v Exp) bool {
	return false
}

func (d DelayedEx) String() string {
	return fmt.Sprintf(`{"delayed": %s}`, d.Exp.String())
}

func NewDelayedExp(ctx Context, exp Exp, env Env) DelayedEx {
	return DelayedEx{
		Context: ctx,
		Exp:     exp,
		Env:     env,
	}
}

var ErrNotDelayedExp = errors.New("Not Delayed Exp")

func ToDelayedExp(v Exp) (DelayedEx, error) {
	if v.Kind() != DelayedExp {
		return DelayedEx{}, ErrNotDelayedExp
	}

	return v.(DelayedEx), nil
}

// SuspendExp
type SuspendEx Redex

func (s SuspendEx) Kind() Kind {
	return SuspendExp
}

func (s SuspendEx) Equal(v Exp) bool {
	return v.Kind() == SuspendExp && EqualRedex(Redex(s), Redex(v.(SuspendEx)))
}

func (s SuspendEx) String() string {
	return fmt.Sprintf(`{"suspend": %s}`, Redex(s))
}

func NewSuspendExp(r Redex) SuspendEx {
	return SuspendEx(r)
}

var ErrNotSuspendExp = errors.New("Not Suspend Exp")

func ToSuspendExp(v Exp) (SuspendEx, error) {
	if v.Kind() != SuspendExp {
		return SuspendEx{}, ErrNotSuspendExp
	}

	return v.(SuspendEx), nil
}

func UnsuspendExp(s SuspendEx) Redex {
	return Redex(s)
}

func IsValue(exp Exp) bool {
	switch exp.Kind() {
	case DelayedExp:
		return false
	case MapExp:
		m, err := ToMapExp(exp)
		if err != nil {
			panic(err)
		}
		for _, subExp := range m {
			if !IsValue(subExp) {
				return false
			}
		}

		return true
	case ListExp:
		l, err := ToListExp(exp)
		if err != nil {
			panic(err)
		}
		for _, subExp := range l {
			if !IsValue(subExp) {
				return false
			}
		}

		return true
	case SuspendExp:
		s, err := ToSuspendExp(exp)
		if err != nil {
			panic(err)
		}
		return IsValue(s.Exp)
	case ReducibleExp:
		return false
	default:
		return true
	}
}

func IsSimpleValue(exp Exp) bool {
	switch exp.Kind() {
	case MapExp, ListExp, DelayedExp, ReducibleExp, SuspendExp:
		return false
	default:
		return true
	}
}
