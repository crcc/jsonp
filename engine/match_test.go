package engine

import "testing"

func TestAny(t *testing.T) {
	pat := Pat.Any.Build()

	exp := NewString("a")
	m, err := Match(exp, pat)
	if err != nil {
		t.Fatal(err.Error())
	}
	if len(m) != 0 {
		t.Fatal("mapping should be empty")
	}
}

func TestExpOfKind(t *testing.T) {
	pat := Pat.OfKind(StringValue, ListExp).Build()

	exp1 := NewString("a")
	m, err := Match(exp1, pat)
	if err != nil {
		t.Fatal(err.Error())
	}
	if len(m) != 0 {
		t.Fatal("mapping should be empty")
	}

	exp2 := NewListExp([]Exp{})
	m, err = Match(exp2, pat)
	if err != nil {
		t.Fatal(err.Error())
	}
	if len(m) != 0 {
		t.Fatal("mapping should be empty")
	}

	exp3 := NewNumber(1)
	_, err = Match(exp3, pat)
	if err == nil {
		t.Fatalf("should not match %s with number", pat.String())
	}
}

func TestEqual(t *testing.T) {
	exp := NewListExp([]Exp{
		NewMapExp(map[string]Exp{
			"a": NewNumber(1),
		}),
		NewBoolean(true),
	})

	pat := Pat.Equal(exp).Build()

	m, err := Match(exp, pat)
	if err != nil {
		t.Fatal(err.Error())
	}
	if len(m) != 0 {
		t.Fatal("mapping should be empty")
	}

	exp2 := NewListExp([]Exp{
		NewBoolean(true),
		NewMapExp(map[string]Exp{
			"a": NewNumber(1),
		}),
	})
	_, err = Match(exp2, pat)
	if err == nil {
		t.Fatalf("should not match %s with %s", pat.String(), exp2.String())
	}
}

func TestSeqOr(t *testing.T) {
	pat := Pat.Equal(NewBoolean(false)).SeqOr(Pat.OfKind(StringValue).Build()).Build()

	exp1 := NewBoolean(false)
	m, err := Match(exp1, pat)
	if err != nil {
		t.Fatal(err.Error())
	}
	if len(m) != 0 {
		t.Fatal("mapping should be empty")
	}

	exp2 := NewString("b")
	m, err = Match(exp2, pat)
	if err != nil {
		t.Fatal(err.Error())
	}
	if len(m) != 0 {
		t.Fatal("mapping should be empty")
	}

	exp3 := NewNumber(1)
	_, err = Match(exp3, pat)
	if err == nil {
		t.Fatalf("should not match %s with %s", pat.String(), exp3.String())
	}
}

func TestCapture(t *testing.T) {
	pat1 := Pat.Any.As("x").Build()

	exp1 := NewBoolean(false)
	m, err := Match(exp1, pat1)
	if err != nil {
		t.Fatal(err.Error())
	}
	if len(m) != 1 {
		t.Fatal("mapping has one capture")
	}
	if !exp1.Equal(m["x"]) {
		t.Fatalf("expect x mapping to %s", exp1.String())
	}

	pat2 := Pat.Equal(NewBoolean(false)).SeqOr(Pat.OfKind(StringValue).Build()).As("x").Build()
	exp2 := NewString("b")
	m, err = Match(exp2, pat2)
	if err != nil {
		t.Fatal(err.Error())
	}
	if len(m) != 1 {
		t.Fatal("mapping has one capture")
	}
	if !exp2.Equal(m["x"]) {
		t.Fatalf("expect x mapping to %s", exp2.String())
	}
}

func TestListPattern(t *testing.T) {
	pat := Pat.List(
		Pat.Any.As("x").BuildListItem(),
		Pat.Any.BuildListItem(),
		Pat.Any.BuildListItem(),
	).Build()

	exp := NewListExp([]Exp{
		NewString("a"),
		NewString("b"),
		NewString("c"),
	})

	m, err := Match(exp, pat)
	if err != nil {
		t.Fatal(err.Error())
	}
	if len(m) != 1 {
		t.Fatal("mapping has one capture")
	}
	if !NewString("a").Equal(m["x"]) {
		t.Fatal("expect x mapping to a")
	}
}

func TestListPattern_Repeat(t *testing.T) {
	pat := Pat.List(
		Pat.Any.As("x").BuildListItem(),
		Pat.Any.As("y").BuildListRepeat(0, InfiniteTimes),
	).Build()

	exp := NewListExp([]Exp{
		NewString("a"),
		NewString("b"),
		NewString("c"),
	})

	m, err := Match(exp, pat)
	if err != nil {
		t.Fatal(err.Error())
	}
	if len(m) != 2 {
		t.Fatal("mapping has 2 captures")
	}
	if !NewString("a").Equal(m["x"]) {
		t.Fatal("expect x mapping to a")
	}
	if !NewListExp([]Exp{NewString("b"), NewString("c")}).Equal(m["y"]) {
		t.Fatal("expect y mapping to [b, c]")
	}
}

func TestMapPattern(t *testing.T) {
	pat := Pat.Map(
		Pat.Equal(NewString("b")).BuildMapItem("^a$"),
		Pat.Any.As("x").BuildMapItem("x.*"),
	).Build()

	exp := NewMapExp(map[string]Exp{
		"a":  NewString("b"),
		"xa": NewString("c"),
	})

	m, err := Match(exp, pat)
	if err != nil {
		t.Fatal(err.Error())
	}
	if len(m) != 1 {
		t.Fatal("mapping has one capture")
	}
	if !NewString("c").Equal(m["x"]) {
		t.Fatal("expect x mapping to c")
	}
}

func TestMapPattern_Repeat(t *testing.T) {
	pat := Pat.Map(
		Pat.Equal(NewString("b")).BuildMapItem("^a$"),
		Pat.Any.As("x").BuildMapRepeat("^x.*", 0, InfiniteTimes),
	).Build()

	exp := NewMapExp(map[string]Exp{
		"a":  NewString("b"),
		"xa": NewString("c"),
		"xb": NewString("d"),
	})

	m, err := Match(exp, pat)
	if err != nil {
		t.Fatal(err.Error())
	}
	if len(m) != 1 {
		t.Fatal("mapping has one capture")
	}

	expectExp1 := NewListExp([]Exp{NewString("c"), NewString("d")})
	expectExp2 := NewListExp([]Exp{NewString("d"), NewString("c")})
	if !expectExp1.Equal(m["x"]) && !expectExp2.Equal(m["x"]) {
		t.Fatal("expect x mapping to [c, d] or [d, c]")
	}
}

func TestRedexPattern(t *testing.T) {
	pat := Pat.List(
		Pat.Any.As("cond").BuildListItem(),
		Pat.Any.As("then").BuildListItem(),
		Pat.Any.As("else").BuildListItem(),
	).Redex("if").Build()

	exp := NewRedex("if", NewListExp([]Exp{
		NewBoolean(true),
		NewString("a"),
		NewString("b"),
	}))
	m, err := Match(exp, pat)
	if err != nil {
		t.Fatal(err.Error())
	}
	if len(m) != 3 {
		t.Fatal("mapping has 3 captures")
	}
	if !NewBoolean(true).Equal(m["cond"]) {
		t.Fatal("expect cond mapping to true")
	}
	if !NewString("a").Equal(m["then"]) {
		t.Fatal("expect then mapping to a")
	}
	if !NewString("b").Equal(m["else"]) {
		t.Fatal("expect else mapping to b")
	}

	exp2 := NewSuspendExp(exp)
	_, err = Match(exp2, pat)
	if err == nil {
		t.Fatalf("should not match %s with %s", pat.String(), exp2.String())
	}

	exp3 := NewSuspendExp(exp)
	_, err = Match(exp3, pat)
	if err == nil {
		t.Fatalf("should not match %s with %s", pat.String(), exp3.String())
	}
}

func TestSuspendExpPattern(t *testing.T) {
	printVar := NewRedex("var", NewString("print"))
	pat := Pat.List(
		Pat.Equal(printVar).BuildListItem(),
		Pat.List(
			Pat.Any.As("cond").BuildListItem(),
			Pat.Any.As("then").BuildListItem(),
			Pat.Any.As("else").BuildListItem(),
		).Redex("if").BuildListItem(),
	).SuspendExp("apply").Build()

	ifExp := NewRedex("if", NewListExp([]Exp{
		NewBoolean(true),
		NewString("a"),
		NewString("b"),
	}))
	exp1 := NewRedex("apply", NewListExp([]Exp{
		printVar, ifExp,
	}))
	_, err := Match(exp1, pat)
	if err == nil {
		t.Fatalf("should not match %s with %s", pat.String(), exp1.String())
	}

	exp2 := NewSuspendExp(exp1)
	m, err := Match(exp2, pat)
	if err != nil {
		t.Fatal(err.Error())
	}
	if len(m) != 3 {
		t.Fatal("mapping has 3 captures")
	}
	if !NewBoolean(true).Equal(m["cond"]) {
		t.Fatal("expect cond mapping to true")
	}
	if !NewString("a").Equal(m["then"]) {
		t.Fatal("expect then mapping to a")
	}
	if !NewString("b").Equal(m["else"]) {
		t.Fatal("expect else mapping to b")
	}

	exp3 := NewSuspendValue(exp1)
	_, err = Match(exp3, pat)
	if err == nil {
		t.Fatalf("should not match %s with %s", pat.String(), exp3.String())
	}
}

func TestSuspendValuePattern(t *testing.T) {
	printVar := NewRedex("var", NewString("print"))
	pat := Pat.List(
		Pat.Equal(printVar).BuildListItem(),
		Pat.List(
			Pat.Any.As("cond").BuildListItem(),
			Pat.Any.As("then").BuildListItem(),
			Pat.Any.As("else").BuildListItem(),
		).Redex("if").BuildListItem(),
	).SuspendValue("apply").Build()

	ifExp := NewRedex("if", NewListExp([]Exp{
		NewBoolean(true),
		NewString("a"),
		NewString("b"),
	}))
	exp1 := NewRedex("apply", NewListExp([]Exp{
		printVar, ifExp,
	}))
	_, err := Match(exp1, pat)
	if err == nil {
		t.Fatalf("should not match %s with %s", pat.String(), exp1.String())
	}

	exp2 := NewSuspendExp(exp1)
	_, err = Match(exp2, pat)
	if err == nil {
		t.Fatalf("should not match %s with %s", pat.String(), exp1.String())
	}

	exp3 := NewSuspendValue(exp1)
	m, err := Match(exp3, pat)
	if err != nil {
		t.Fatal(err.Error())
	}
	if len(m) != 3 {
		t.Fatal("mapping has 3 captures")
	}
	if !NewBoolean(true).Equal(m["cond"]) {
		t.Fatal("expect cond mapping to true")
	}
	if !NewString("a").Equal(m["then"]) {
		t.Fatal("expect then mapping to a")
	}
	if !NewString("b").Equal(m["else"]) {
		t.Fatal("expect else mapping to b")
	}

	exp4 := NewSuspendExp(NewRedex("apply", NewListExp([]Exp{
		printVar, NewSuspendExp(ifExp),
	})))
	m, err = Match(exp4, pat)
	if err != nil {
		t.Fatal(err.Error())
	}
	if len(m) != 3 {
		t.Fatal("mapping has 3 captures")
	}
	if !NewBoolean(true).Equal(m["cond"]) {
		t.Fatal("expect cond mapping to true")
	}
	if !NewString("a").Equal(m["then"]) {
		t.Fatal("expect then mapping to a")
	}
	if !NewString("b").Equal(m["else"]) {
		t.Fatal("expect else mapping to b")
	}
}
