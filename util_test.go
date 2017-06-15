package main

import (
	"bytes"
	"encoding/gob"
	"io"
	"testing"
)

func TestMsgWrite(t *testing.T) {
	var err error
	var buf bytes.Buffer

	tests := []string{"foo", "bar", "baz"}

	wc := msgWrite(gob.NewEncoder(&buf), "test")

	for _, test := range tests {
		if _, err = wc.Write([]byte(test)); err != nil {
			t.Fatal(err)
		}
	}

	dec := gob.NewDecoder(&buf)

	for _, test := range tests {
		var m msg
		err = dec.Decode(&m)
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatal(err)
		}
		if m.Name != "test" {
			t.Fatalf("want %q but %q", "test", m.Name)
		}
		if string(m.Data) != test {
			t.Fatalf("want %q but %q", test, string(m.Data))
		}
	}
}

func TestMakeCmdLine(t *testing.T) {
	got := makeCmdLine([]string{"foo", "bar baz"})
	want := `foo "bar baz"`
	if got != want {
		t.Fatalf("want %q but %q", want, got)
	}
}
