package joinederr_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/mvndaai/ctxerr/joinederr"
)

func TestBreadthFirst(t *testing.T) {
	/*
				a
			   / \
			  /   \
			b      f
		   / \     |
		  c   e    g
		  |       / \
		  d	     h   i
	*/

	i := errors.New("i")
	h := errors.New("h")
	g := fmt.Errorf("g\n%w", errors.Join(h, i))
	f := fmt.Errorf("f\n%w", g)
	d := errors.New("d")
	c := fmt.Errorf("c\n%w", d)
	e := errors.New("e")
	b := fmt.Errorf("b\n%w", errors.Join(c, e))
	a := fmt.Errorf("a\n%w", errors.Join(b, f))

	msgs := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i"}
	actualMsg := a.Error()
	expectedMessage := strings.Join(msgs, "\n")
	if actualMsg != expectedMessage {
		t.Errorf("message did not match\n%#v\n%#v", actualMsg, expectedMessage)
	}

	expectedTrees := [][]string{
		{"a", "b", "c", "d", "e", "f", "g", "h", "i"},
		{"b", "c", "d", "e"},
		{"c", "d"},
		{"d"},
		{"e"},
		{"f", "g", "h", "i"},
		{"g", "h", "i"},
		{"h"},
		{"i"},
	}
	var expectTreeCount int

	iter := joinederr.NewDepthFirstIterator(a)
	for iter.HasNext() {
		actualMsg := strings.ReplaceAll(iter.Next().Error(), "\n", ":")
		expectedMessage := strings.Join(expectedTrees[expectTreeCount], ":")
		if actualMsg != expectedMessage {
			t.Errorf("message [%v] did not match\n%#v\n%#v", expectTreeCount, actualMsg, expectedMessage)
		}
		expectTreeCount++
	}

	if expectTreeCount != len(expectedTrees) {
		t.Errorf("stopped iterating [%v] setps early", len(expectedTrees)-expectTreeCount)
	}

	if iter.Next() != nil {
		t.Error("this there should be nothing left")
	}
}
