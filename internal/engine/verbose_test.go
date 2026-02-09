package engine

import (
	"bytes"
	"testing"
)

func TestVerboseLogger(t *testing.T) {
	t.Run("it writes to writer with verbose prefix when verbose is true", func(t *testing.T) {
		var buf bytes.Buffer
		vl := NewVerboseLogger(&buf, true)
		vl.Log("cache rebuild started")

		got := buf.String()
		want := "verbose: cache rebuild started\n"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("it writes nothing when verbose is false", func(t *testing.T) {
		var buf bytes.Buffer
		vl := NewVerboseLogger(&buf, false)
		vl.Log("cache rebuild started")

		if buf.Len() != 0 {
			t.Errorf("expected no output when verbose off, got %q", buf.String())
		}
	})

	t.Run("it supports formatted output", func(t *testing.T) {
		var buf bytes.Buffer
		vl := NewVerboseLogger(&buf, true)
		vl.Logf("hash %s -> %s", "abc123", "def456")

		got := buf.String()
		want := "verbose: hash abc123 -> def456\n"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("it writes nothing for formatted output when verbose is false", func(t *testing.T) {
		var buf bytes.Buffer
		vl := NewVerboseLogger(&buf, false)
		vl.Logf("hash %s -> %s", "abc123", "def456")

		if buf.Len() != 0 {
			t.Errorf("expected no output when verbose off, got %q", buf.String())
		}
	})

	t.Run("it handles nil writer gracefully when verbose is false", func(t *testing.T) {
		vl := NewVerboseLogger(nil, false)
		// Should not panic
		vl.Log("test")
		vl.Logf("test %s", "arg")
	})
}
