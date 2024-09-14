package scanner

import (
	"bytes"
	"testing"
)

// TestRemoveQuotes tests the removeQuotes transformer.
func TestUnescapeQuotes(t *testing.T) {
	// data driven tests
	data := []struct {
		in       []byte
		quote    byte
		escape   byte
		expected []byte
	}{
		{[]byte{}, 0, 0, []byte{}},
		{[]byte{}, '"', 0, []byte{}},
		{[]byte{}, '"', '"', []byte{}},
		{[]byte{}, '"', '\\', []byte{}},
		{[]byte(` x""y `), 0, 0, []byte(` x""y `)},
		{[]byte(` x""y `), '"', 0, []byte(` x""y `)},
		{[]byte(` x""y `), '"', '"', []byte(` x"y `)},
		{[]byte(`"" x""""y ""`), '"', '"', []byte(`" x""y "`)},
		{[]byte(` x" "y `), '"', '"', []byte(` x" "y `)},
		{[]byte(` "x""y" `), '"', '"', []byte(` "x"y" `)},
		{[]byte(` "x""y" `), '"', '\\', []byte(` "x""y" `)},
		{[]byte(` "x\"y" `), '"', 0, []byte(` "x\"y" `)},
		{[]byte(` "x\"y" `), '"', '"', []byte(` "x\"y" `)},
		{[]byte(` "x\"y" `), '"', '\\', []byte(` "x"y" `)},
	}

	// check QuoteFuzzy
	if QuoteFuzzy == nil {
		t.Errorf("QuoteFuzzy is nil")
		return
	}
	// run tests
	for _, d := range data {
		// create a scanner
		sc, _ := New(nil,
			WithQuote(d.quote, QuoteFuzzy),
			WithEscape(d.escape),
		).(*scanner)
		sc.value = d.in
		sc.unescapeQuotes()
		if !bytes.Equal(sc.value, d.expected) {
			t.Errorf("fo <%s> expected <%s>, got <%s>", d.in, d.expected, sc.value)
		}
	}
}
