package main

import (
	"encoding/gob"
	"io"
	"testing"
)

func TestMsgWrite(t *testing.T) {
	tests := []string{"foo", "bar", "baz"}

	pr, pw := io.Pipe()
	wc := msgWrite(gob.NewEncoder(pw), "test")

	go func() {
		for _, test := range tests {
			if _, err := wc.Write([]byte(test)); err != nil {
				pw.CloseWithError(err)
				return
			}
		}
		wc.Close()
	}()

	dec := gob.NewDecoder(pr)

	for _, test := range tests {
		var m msg
		if err := dec.Decode(&m); err != nil {
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
