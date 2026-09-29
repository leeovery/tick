package jsonl

import (
	"errors"
	"io"
	"strings"
	"testing"
)

type gotLine struct {
	num  int
	text string
}

func collectLines(t *testing.T, r io.Reader) ([]gotLine, error) {
	t.Helper()
	var got []gotLine
	for line, err := range Lines(r) {
		if err != nil {
			return got, err
		}
		got = append(got, gotLine{num: line.Num, text: string(line.Text)})
	}
	return got, nil
}

func assertLines(t *testing.T, input string, want []gotLine) {
	t.Helper()
	got, err := collectLines(t, strings.NewReader(input))
	if err != nil {
		t.Fatalf("Lines returned error: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("got %d lines %v, want %d lines %v", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d = {%d, %q}, want {%d, %q}", i, got[i].num, got[i].text, want[i].num, want[i].text)
		}
	}
}

type failingReader struct {
	data []byte
	err  error
}

func (r *failingReader) Read(p []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, r.err
	}
	n := copy(p, r.data)
	r.data = r.data[n:]
	return n, nil
}

func TestLines(t *testing.T) {
	t.Run("it returns lines of any length whole", func(t *testing.T) {
		for _, size := range []int{65535, 65536, 1 << 20} {
			long := strings.Repeat("x", size)
			assertLines(t, "a\n"+long+"\nb\n", []gotLine{{1, "a"}, {2, long}, {3, "b"}})
		}
	})

	t.Run("it returns a final long line with no trailing newline whole", func(t *testing.T) {
		long := strings.Repeat("y", 1<<20)
		assertLines(t, "a\n"+long, []gotLine{{1, "a"}, {2, long}})
	})

	t.Run("it reads CRLF lines the same as LF lines", func(t *testing.T) {
		assertLines(t, "a\r\nb\r\n", []gotLine{{1, "a"}, {2, "b"}})
	})

	t.Run("it strips only one carriage return before the newline", func(t *testing.T) {
		assertLines(t, "a\r\r\nb\n", []gotLine{{1, "a\r"}, {2, "b"}})
	})

	t.Run("it keeps a carriage return that does not end the line", func(t *testing.T) {
		assertLines(t, "a\rb\n", []gotLine{{1, "a\rb"}})
	})

	t.Run("it reads a final line with no trailing newline", func(t *testing.T) {
		assertLines(t, "a\nb", []gotLine{{1, "a"}, {2, "b"}})
	})

	t.Run("it strips one carriage return from a final line with no trailing newline", func(t *testing.T) {
		assertLines(t, "a\nb\r", []gotLine{{1, "a"}, {2, "b"}})
	})

	t.Run("it skips a final lone carriage return as an empty line", func(t *testing.T) {
		assertLines(t, "a\n\r", []gotLine{{1, "a"}})
	})

	t.Run("it skips empty lines and counts them in the numbering", func(t *testing.T) {
		assertLines(t, "\na\n\n\r\nb\n\n", []gotLine{{2, "a"}, {5, "b"}})
	})

	t.Run("it returns a whitespace-only line rather than skipping it", func(t *testing.T) {
		assertLines(t, "a\n\n   \n", []gotLine{{1, "a"}, {3, "   "}})
	})

	t.Run("it returns no lines for empty input", func(t *testing.T) {
		assertLines(t, "", nil)
	})

	t.Run("it reports a read error with the line it could not read", func(t *testing.T) {
		boom := errors.New("boom")
		got, err := collectLines(t, &failingReader{data: []byte("a\n\nc"), err: boom})
		if !errors.Is(err, boom) {
			t.Fatalf("err = %v, want wrapping %v", err, boom)
		}
		if want := "read line 3: boom"; err.Error() != want {
			t.Errorf("err = %q, want %q", err.Error(), want)
		}
		if len(got) != 1 || got[0] != (gotLine{1, "a"}) {
			t.Errorf("lines before error = %v, want [{1 a}]", got)
		}
	})

	t.Run("it stops when the consumer breaks", func(t *testing.T) {
		count := 0
		for range Lines(strings.NewReader("a\nb\nc\n")) {
			count++
			break
		}
		if count != 1 {
			t.Errorf("iterations = %d, want 1", count)
		}
	})
}
