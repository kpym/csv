package writer

import (
	"bufio"
	"strings"
	"testing"
)

func TestHasQuoteSep(t *testing.T) {
	// data driven tests
	data := []struct {
		in       string
		sep      byte
		quote    byte
		expected bool
	}{
		{"", ',', '"', false},
		{"test", ',', '"', false},
		{",test", ',', '"', true},
		{"test\"", ',', '"', true},
		{"test\r", ',', '"', true},
		{"test\n", ',', '"', true},
	}

	// run tests
	for _, d := range data {
		// create a writer
		w := &writer{
			sep:   d.sep,
			quote: d.quote,
		}
		w.setqsnl()
		got := w.hasQuoteSep([]byte(d.in))
		if got != d.expected {
			t.Errorf("for <%s> expected <%v>, got <%v>", d.in, d.expected, got)
		}
	}
}

func TestWriteEscaped(t *testing.T) {
	data := []struct {
		in       string
		sep      byte
		quote    byte
		expected string
	}{
		{"", ',', '"', ""},
		{"test", ',', '"', "test"},
		{",test\"test,test", ',', '"', ",test\"\"test,test"},
	}

	// run tests
	for _, d := range data {
		gotw := strings.Builder{}
		// create a writer
		w := &writer{
			bufw:   bufio.NewWriter(&gotw),
			sep:    d.sep,
			quote:  d.quote,
			escape: d.quote,
		}
		w.writeEscaped([]byte(d.in))
		w.bufw.Flush()
		got := gotw.String()
		if got != d.expected {
			t.Errorf("for <%s> expected <%s>, got <%s>", d.in, d.expected, got)
		}
	}
}
