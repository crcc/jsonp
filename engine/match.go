package engine

import (
	"fmt"
	"regexp"
	"strings"
)

// Pattern ::= Capture(name, Pattern)
// | ExpOfKind([]Kind)
// | Value(Exp)
// | RedexPattern
// | List([]ListItemPattern)
// | Map([]MapItemPattern)
// | SuspendExp(RedexPattern)
// | SuspendValue(RedexPattern)
// | SeqOr([]Pattern)
// RedexPattern ::= Redex(Name, Pattern)
// ListItemPattern ::= ListItem(Pattern) | RepeatListItem(ListItem, from, to)
// MapItemPattern ::= MapItem(Regexp, Pattern) | RepeatMapItem(MapItem, from, to)

type PatternBuilder interface {
	As(name string) PatternBuilder
	SeqOr(pat Pattern) PatternBuilder
	Redex(name string) PatternBuilder
	SuspendExp(name string) PatternBuilder
	SuspendValue(name string) PatternBuilder

	Build() Pattern
	BuildListItem() ListItemPattern
	BuildListRepeat(from, to Times) ListItemPattern
	BuildMapItem(keyPat string) MapItemPattern
	BuildMapRepeat(keyPat string, from, to Times) MapItemPattern
}

var Pat = struct {
	Any       PatternBuilder
	OfKind    func(kinds ...Kind) PatternBuilder
	OfPattern func(pat Pattern) PatternBuilder
	Equal     func(exp Exp) PatternBuilder
	List      func(pats ...ListItemPattern) PatternBuilder
	Map       func(pats ...MapItemPattern) PatternBuilder
}{
	Any: patBuilder{
		pat: NewExpOfKindPattern(nil),
	},
	OfKind: func(kinds ...Kind) PatternBuilder {
		return patBuilder{
			pat: NewExpOfKindPattern(kinds),
		}
	},
	OfPattern: func(pat Pattern) PatternBuilder {
		return patBuilder{
			pat: pat,
		}
	},
	Equal: func(exp Exp) PatternBuilder {
		return patBuilder{
			pat: NewEqualPattern(exp),
		}
	},
	List: func(pats ...ListItemPattern) PatternBuilder {
		return patBuilder{
			pat: NewListPattern(pats),
		}
	},
	Map: func(pats ...MapItemPattern) PatternBuilder {
		return patBuilder{
			pat: NewMapPattern(pats),
		}
	},
}

type patBuilder struct {
	pat Pattern
}

func (pb patBuilder) As(name string) PatternBuilder {
	return patBuilder{
		pat: NewCapturePattern(name, pb.pat),
	}
}

func (pb patBuilder) SeqOr(pat Pattern) PatternBuilder {
	var pats []Pattern
	if sop, ok := pb.pat.(*SeqOrPat); ok {
		pats = sop.pats
		if sop2, ok := pat.(*SeqOrPat); ok {
			pats = append(pats, sop2.pats...)
		} else {
			pats = append(pats, pat)
		}
	} else {
		if sop2, ok := pat.(*SeqOrPat); ok {
			pats = make([]Pattern, 0, len(sop2.pats)+1)
			pats = append(pats, pb.pat)
			pats = append(pats, sop2.pats...)
		} else {
			pats = []Pattern{pb.pat, pat}
		}
	}

	return patBuilder{
		pat: NewSeqOrPattern(pats),
	}
}

func (pb patBuilder) Redex(name string) PatternBuilder {
	return patBuilder{
		pat: NewRedexPattern(name, pb.pat),
	}
}

func (pb patBuilder) SuspendExp(name string) PatternBuilder {
	return patBuilder{
		pat: NewSuspendExpPattern(name, pb.pat),
	}
}

func (pb patBuilder) SuspendValue(name string) PatternBuilder {
	return patBuilder{
		pat: NewSuspendValuePattern(name, pb.pat),
	}
}

func (pb patBuilder) Build() Pattern {
	return pb.pat
}

func (pb patBuilder) BuildListItem() ListItemPattern {
	return NewListItemPat(pb.pat)
}

func (pb patBuilder) BuildListRepeat(from, to Times) ListItemPattern {
	return NewRepeatListItemPat(pb.pat, from, to)
}

func (pb patBuilder) BuildMapItem(keyPat string) MapItemPattern {
	return NewMapItemPat(regexp.MustCompile(keyPat), pb.pat)
}

func (pb patBuilder) BuildMapRepeat(keyPat string, from, to Times) MapItemPattern {
	mapItemPat := NewMapItemPat(regexp.MustCompile(keyPat), pb.pat)
	return NewRepeatMapItemPat(mapItemPat, from, to)
}

type Pattern interface {
	Match(ctx Context, mapping map[string]Exp, exp Exp) error
	String() string
}

type ListItemPattern interface {
	Match(ctx Context, mapping map[string]Exp, exp []Exp) ([]Exp, error)
	String() string
}

type MapItemPattern interface {
	Match(ctx Context, mapping map[string]Exp, exp map[string]Exp) (map[string]Exp, error)
	String() string
}

func Match(exp Exp, pat Pattern) (map[string]Exp, error) {
	result := make(map[string]Exp)
	if err := pat.Match(NewContext(nil), result, exp); err != nil {
		return nil, err
	}
	return result, nil
}

// helper

func copyMapExp(m map[string]Exp) map[string]Exp {
	res := make(map[string]Exp, len(m))
	for k, v := range m {
		res[k] = v
	}
	return res
}

const (
	patternSuspendKey = "pattern-suspend"
	expSuspendKey     = "exp-suspend"
)

func isPatternSuspend(ctx Context) bool {
	val := ctx.Get(patternSuspendKey)
	if val == nil {
		return false
	}
	v, ok := val.(bool)
	return ok && v
}

func isExpSuspend(ctx Context) bool {
	val := ctx.Get(expSuspendKey)
	if val == nil {
		return false
	}
	v, ok := val.(bool)
	return ok && v
}

// capture
type CapturePat struct {
	name string
	pat  Pattern
}

func NewCapturePattern(name string, pat Pattern) *CapturePat {
	return &CapturePat{
		name: name,
		pat:  pat,
	}
}

func (p *CapturePat) Match(ctx Context, mapping map[string]Exp, exp Exp) error {
	err := p.pat.Match(ctx, mapping, exp)
	if err != nil {
		return err
	}

	if _, ok := mapping[p.name]; ok {
		return fmt.Errorf("duplicated variable name %s", p.name)
	}

	mapping[p.name] = exp
	return nil
}

func (p *CapturePat) String() string {
	return fmt.Sprintf(`{"as": [%s, %q]}`, p.pat.String(), p.name)
}

// exp of kind
type ExpOfKindPat struct {
	kinds []Kind
}

func NewExpOfKindPattern(kinds []Kind) *ExpOfKindPat {
	return &ExpOfKindPat{
		kinds: kinds,
	}
}

func (p *ExpOfKindPat) Match(ctx Context, mapping map[string]Exp, exp Exp) error {
	if len(p.kinds) == 0 {
		return nil
	}

	for _, kind := range p.kinds {
		if exp.Kind() == kind {
			return nil
		}
	}

	return fmt.Errorf("expect %s, but found %s", p.String(), exp.String())
}

func (p *ExpOfKindPat) String() string {
	if len(p.kinds) == 0 {
		return `{"any": null}`
	}
	kindStrs := make([]string, len(p.kinds))
	for i, k := range p.kinds {
		kindStrs[i] = fmt.Sprintf("%d", k)
	}
	return fmt.Sprintf(`{"ofKind": [%s]`, strings.Join(kindStrs, ","))
}

// equal
type EqualPat struct {
	exp Exp
}

func NewEqualPattern(exp Exp) *EqualPat {
	return &EqualPat{
		exp: exp,
	}
}

func (p *EqualPat) Match(ctx Context, mapping map[string]Exp, exp Exp) error {
	if p.exp.Equal(exp) {
		return nil
	}
	return fmt.Errorf("expect %s, but found %s", p.String(), exp.String())
}

func (p *EqualPat) String() string {
	return fmt.Sprintf(`{"equal": %s}`, p.exp.String())
}

// redex
type RedexPat struct {
	name string
	pat  Pattern
}

func NewRedexPattern(name string, pat Pattern) *RedexPat {
	return &RedexPat{
		name: name,
		pat:  pat,
	}
}

// 获取suspend的exp/value中的redex
func unwrapSuspend(exp Exp, expSuspend bool) (Redex, bool, error) {
	switch exp.Kind() {
	case SuspendExp:
		s, err := ToSuspendExp(exp)
		if err != nil {
			return Redex{}, false, err
		}
		return UnsuspendExp(s), expSuspend, nil
	case SuspendValue:
		s, err := ToSuspendValue(exp)
		if err != nil {
			return Redex{}, false, err
		}
		return UnsuspendValue(s), true, nil
	case ReducibleExp:
		if !expSuspend {
			return Redex{}, false, fmt.Errorf("not match, expect suspend exp/value, but found redex %s", exp.String())
		}
		r, err := ToRedex(exp)
		if err != nil {
			return Redex{}, false, err
		}
		return r, expSuspend, nil
	default:
		return Redex{}, false, fmt.Errorf("not match. expect redex (suspend or not suspend)")
	}
}

func (p *RedexPat) Match(ctx Context, mapping map[string]Exp, exp Exp) error {
	patSuspend := isPatternSuspend(ctx)
	expSuspend := isExpSuspend(ctx)
	// expSuspend, patSuspend
	// false, false: 都是redex，相同name，exp匹配
	// false, true: exp是SuspendExp或SuspendVal，相同name，exp匹配
	// true, false: 不匹配
	// true, true: exp是redex，SuspendExp，SuspendVal，相同name，exp匹配
	var redex Redex
	newCtx := ctx
	if patSuspend {
		r, suspend, err := unwrapSuspend(exp, expSuspend)
		if err != nil {
			return err
		}
		redex = r
		if suspend != expSuspend {
			newCtx = ctx.NewChild(map[string]interface{}{
				expSuspendKey: suspend,
			})
		}
	} else {
		if expSuspend {
			return fmt.Errorf("not match, expect redex %s, but found suspend exp/value %s", p.String(), exp.String())
		}

		r, err := ToRedex(exp)
		if err != nil {
			return fmt.Errorf("expect %s, but found %s", p.String(), exp.String())
		}
		redex = r
	}

	if redex.Name != p.name {
		return fmt.Errorf("expect %s, but found %s", p.String(), exp.String())
	}
	return p.pat.Match(newCtx, mapping, redex.Exp)
}

func (p *RedexPat) String() string {
	return fmt.Sprintf(`{"redex": [%q, %s]`, p.name, p.pat.String())
}

// suspendExp

type SuspendExpPat struct {
	name string
	pat  Pattern
}

func NewSuspendExpPattern(name string, pat Pattern) *SuspendExpPat {
	return &SuspendExpPat{
		name: name,
		pat:  pat,
	}
}

func (p *SuspendExpPat) Match(ctx Context, mapping map[string]Exp, exp Exp) error {
	// patSuspend := isPatternSuspend(ctx)
	expSuspend := isExpSuspend(ctx)
	// expSuspend, patSuspend
	// false, false: exp是SuspendExp或SuspendVal，相同name，exp匹配
	// false, true: exp是SuspendExp或SuspendVal，相同name，exp匹配
	// true, false: exp是redex，SuspendExp，SuspendVal，相同name，exp匹配
	// true, true: exp是redex，SuspendExp，SuspendVal，相同name，exp匹配
	r, suspend, err := unwrapSuspend(exp, expSuspend)
	if err != nil {
		return err
	}
	newCtx := ctx
	if suspend != expSuspend {
		newCtx = ctx.NewChild(map[string]interface{}{
			expSuspendKey: suspend,
		})
	}

	if r.Name != p.name {
		return fmt.Errorf("expect %s, but found %s", p.String(), exp.String())
	}
	if err := p.pat.Match(newCtx, mapping, r.Exp); err != nil {
		return err
	}
	return nil
}

func (p *SuspendExpPat) String() string {
	return fmt.Sprintf(`{"suspendExp": [%q, %s]`, p.name, p.pat.String())
}

// suspendValue
type SuspendValuePat struct {
	name string
	pat  Pattern
}

func NewSuspendValuePattern(name string, pat Pattern) *SuspendValuePat {
	return &SuspendValuePat{
		name: name,
		pat:  pat,
	}
}

func (p *SuspendValuePat) Match(ctx Context, mapping map[string]Exp, exp Exp) error {
	patSuspend := isPatternSuspend(ctx)
	expSuspend := isExpSuspend(ctx)

	newCtx := ctx
	if !patSuspend {
		newCtx = ctx.NewChild(map[string]interface{}{
			patternSuspendKey: true,
		})
	}

	r, suspend, err := unwrapSuspend(exp, expSuspend)
	if err != nil {
		return err
	}
	if suspend != expSuspend {
		if !patSuspend {
			newCtx.Set(expSuspendKey, suspend)
		} else {
			newCtx = ctx.NewChild(map[string]interface{}{
				expSuspendKey: suspend,
			})
		}
	}

	if r.Name != p.name {
		return fmt.Errorf("expect %s, but found %s", p.String(), exp.String())
	}
	return p.pat.Match(newCtx, mapping, r.Exp)
}

func (p *SuspendValuePat) String() string {
	return fmt.Sprintf(`{"suspendValue": [%q, %s]`, p.name, p.pat.String())
}

// seq or

type SeqOrPat struct {
	pats []Pattern
}

func NewSeqOrPattern(pats []Pattern) *SeqOrPat {
	return &SeqOrPat{
		pats: pats,
	}
}

func (p *SeqOrPat) Match(ctx Context, mapping map[string]Exp, exp Exp) error {
	for _, pat := range p.pats {
		err := pat.Match(ctx, mapping, exp)
		if err == nil {
			return nil
		}
	}
	return fmt.Errorf("expect %s, buf found %s", p.String(), exp.String())
}

func (p *SeqOrPat) String() string {
	pStrs := make([]string, len(p.pats))
	for i, p := range p.pats {
		pStrs[i] = p.String()
	}
	return fmt.Sprintf(`{"seqOr": [%s]`, strings.Join(pStrs, ","))
}

// list

type ListPat struct {
	pats []ListItemPattern
}

func NewListPattern(pats []ListItemPattern) *ListPat {
	return &ListPat{
		pats: pats,
	}
}

func (p *ListPat) Match(ctx Context, mapping map[string]Exp, exp Exp) error {
	l, err := ToListExp(exp)
	if err != nil {
		return fmt.Errorf("expect %s, buf found %s", p.String(), exp.String())
	}

	for _, pat := range p.pats {
		l, err = pat.Match(ctx, mapping, l)
		if err != nil {
			return err
		}
	}

	if len(l) != 0 {
		return fmt.Errorf("expect %s, buf found %s", p.String(), exp.String())
	}

	return nil
}

func (p *ListPat) String() string {
	pStrs := make([]string, len(p.pats))
	for i, p := range p.pats {
		pStrs[i] = p.String()
	}
	return fmt.Sprintf(`{"list": [%s]}`, strings.Join(pStrs, ","))
}

// list item pattern

type ListItemPat struct {
	pat Pattern
}

func NewListItemPat(pat Pattern) *ListItemPat {
	return &ListItemPat{
		pat: pat,
	}
}

func (p *ListItemPat) Match(ctx Context, mapping map[string]Exp, exps []Exp) ([]Exp, error) {
	if len(exps) == 0 {
		return nil, fmt.Errorf("expect %s, buf found %s", p.String(), ListEx(exps).String())
	}

	if err := p.pat.Match(ctx, mapping, exps[0]); err != nil {
		return nil, err
	}

	return exps[1:], nil
}

func (p *ListItemPat) String() string {
	return p.pat.String()
}

// repeat list item

type Times int32

func minLen(len int, times Times) int {
	if times < 0 {
		return len
	}

	t := int(times)
	if len > t {
		return t
	} else {
		return len
	}
}

const (
	InfiniteTimes Times = -1
)

type RepeatListItemPat struct {
	pat  Pattern
	from Times
	to   Times
}

func NewRepeatListItemPat(pat Pattern, from, to Times) *RepeatListItemPat {
	return &RepeatListItemPat{
		pat:  pat,
		from: from,
		to:   to,
	}
}

func (p *RepeatListItemPat) Match(ctx Context, mapping map[string]Exp, exps []Exp) ([]Exp, error) {
	length := minLen(len(exps), p.to)
	if length < int(p.from) {
		return nil, fmt.Errorf("expect %s, buf found %s", p.String(), ListEx(exps).String())
	}

	collecting := make(map[string][]Exp)
	n := length
	for i := 0; i < length; i++ {
		m := make(map[string]Exp)
		err := p.pat.Match(ctx, m, exps[i])
		if err != nil {
			n = i
			break
		}
		for key, val := range m {
			if _, ok := mapping[key]; ok {
				return nil, fmt.Errorf("duplicated variable name %s", key)
			}
			collecting[key] = append(collecting[key], val)
		}
	}

	if n < int(p.from) {
		return nil, fmt.Errorf("expect %s, buf found %s", p.String(), ListEx(exps).String())
	}

	for key, val := range collecting {
		mapping[key] = ListEx(val)
	}

	return exps[n:], nil
}

func (p *RepeatListItemPat) String() string {
	toStr := "*"
	if p.to >= 0 {
		toStr = fmt.Sprint(int(p.to))
	}
	return fmt.Sprintf(`{"repeat": [%s,%d,%s]}`, p.pat.String(), p.from, toStr)
}

// map

type MapPat struct {
	pats []MapItemPattern
}

func NewMapPattern(pats []MapItemPattern) *MapPat {
	return &MapPat{
		pats: pats,
	}
}

func (p *MapPat) Match(ctx Context, mapping map[string]Exp, exp Exp) error {
	m, err := ToMapExp(exp)
	if err != nil {
		return fmt.Errorf("expect %s, buf found %s", p.String(), exp.String())
	}

	m = copyMapExp(m)
	for _, pat := range p.pats {
		m, err = pat.Match(ctx, mapping, m)
		if err != nil {
			return err
		}
	}

	if len(m) != 0 {
		return fmt.Errorf("expect %s, buf found %s", p.String(), exp.String())
	}

	return nil
}

func (p *MapPat) String() string {
	pStrs := make([]string, len(p.pats))
	for i, p := range p.pats {
		pStrs[i] = p.String()
	}
	return fmt.Sprintf(`{"map": [%s]}`, strings.Join(pStrs, ","))
}

// map item pattern
type MapItemPat struct {
	keyPat *regexp.Regexp
	valPat Pattern
}

func NewMapItemPat(keyPat *regexp.Regexp, valPat Pattern) *MapItemPat {
	return &MapItemPat{
		keyPat: keyPat,
		valPat: valPat,
	}
}

func (p *MapItemPat) Match(ctx Context, mapping map[string]Exp, exp map[string]Exp) (map[string]Exp, error) {
	for k, v := range exp {
		if !p.keyPat.Match([]byte(k)) {
			continue
		}

		if err := p.valPat.Match(ctx, mapping, v); err != nil {
			return nil, err
		} else {
			delete(exp, k)
			return exp, nil
		}
	}

	return nil, fmt.Errorf("expect %s, buf found %s", p.String(), MapEx(exp).String())
}

func (p *MapItemPat) String() string {
	return fmt.Sprintf(`{"item": [%s, %s]}`, p.keyPat.String(), p.valPat.String())
}

// repeat map item

type RepeatMapItemPat struct {
	pat  *MapItemPat
	from Times
	to   Times
}

func NewRepeatMapItemPat(pat *MapItemPat, from, to Times) *RepeatMapItemPat {
	return &RepeatMapItemPat{
		pat:  pat,
		from: from,
		to:   to,
	}
}

func (p *RepeatMapItemPat) Match(ctx Context, mapping map[string]Exp, exp map[string]Exp) (map[string]Exp, error) {
	length := minLen(len(exp), p.to)
	if length < int(p.from) {
		return nil, fmt.Errorf("expect %s, buf found %s", p.String(), MapEx(exp).String())
	}

	mexp := exp
	n := length
	collecting := make(map[string][]Exp)
	for i := 0; i < length; i++ {
		var err error
		m := make(map[string]Exp)
		mexp, err = p.pat.Match(ctx, m, mexp)
		if err != nil {
			n = i
			break
		}
		for key, val := range m {
			if _, ok := mapping[key]; ok {
				return nil, fmt.Errorf("duplicated variable name %s", key)
			}
			collecting[key] = append(collecting[key], val)
		}
	}

	if n < int(p.from) {
		return nil, fmt.Errorf("expect %s, buf found %s", p.String(), MapEx(exp).String())
	}

	for key, val := range collecting {
		mapping[key] = ListEx(val)
	}

	return mexp, nil
}

func (p *RepeatMapItemPat) String() string {
	toStr := "*"
	if p.to >= 0 {
		toStr = fmt.Sprint(int(p.to))
	}
	return fmt.Sprintf(`{"repeat": [%s,%d,%s]}`, p.pat.String(), p.from, toStr)
}
