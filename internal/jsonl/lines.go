// Package jsonl reads the lines of tick's tasks.jsonl, with no limit on a
// line's length.
package jsonl

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"iter"
)

// Line is one non-empty line of a JSONL file, its terminator removed.
type Line struct {
	// Num is the 1-based line number; skipped empty lines count toward it.
	Num int
	// Text is the line's content, owned by the caller.
	Text []byte
}

// Lines yields every non-empty line r holds, of any length. A line ends at
// "\n", and one "\r" before it is dropped; a final line with no "\n" is
// still yielded, with one trailing "\r" dropped. A line that is empty once
// its terminator is removed is skipped; a whitespace-only line is not.
//
// A read failure is yielded once, as the last pair, naming the line that
// could not be read.
func Lines(r io.Reader) iter.Seq2[Line, error] {
	return func(yield func(Line, error) bool) {
		br := bufio.NewReader(r)
		for num := 1; ; num++ {
			raw, err := br.ReadBytes('\n')
			if err != nil && !errors.Is(err, io.EOF) {
				yield(Line{}, fmt.Errorf("read line %d: %w", num, err))
				return
			}
			if text := trimTerminator(raw); len(text) > 0 {
				if !yield(Line{Num: num, Text: text}, nil) {
					return
				}
			}
			if err != nil {
				return
			}
		}
	}
}

func trimTerminator(raw []byte) []byte {
	raw = bytes.TrimSuffix(raw, []byte("\n"))
	return bytes.TrimSuffix(raw, []byte("\r"))
}
