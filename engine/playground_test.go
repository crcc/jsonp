package engine

import (
	"testing"
)

type ts struct {
	name string
	val  interface{}
}

func TestPlayground(t *testing.T) {
	t1 := ts{
		name: "a",
		val:  2,
	}

	t2 := ts{
		name: "a",
		val:  2,
	}

	t.Log(t1 == t2)
}
