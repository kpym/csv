package scanner

import (
	"bytes"
)

// collector interface used to collect chunks of data to a single field
// mainly used for quoted fields and comments
type collector interface {
	// Start returns true if the current chunk is the start of a new field.
	// The returned chunk is the chunk containgin the actual data of the field.
	// In quoted case, the returned chunk is the chunk without the starting quote.
	// In comment case, the returned chunk is the chunk without the comment prefix.
	Start([]byte) ([]byte, bool)
	// End returns true if the current chunk is the end of the current field.
	// The returned chunk is the chunk containgin the actual data of the field.
	// In quoted case, the returned chunk is the chunk without the ending quote and the separator.
	// In comment case, the returned chunk is the chunk without the ending newline.
	End([]byte) ([]byte, bool)
}

// Util functions
// --------------

// removeSeparator removes the separator from the chunk.
// The last byte should be the separator.
// If the separator is a newline, check for \r too.
// It is used by End of some quote collectors.
func removeSeparator(chunk []byte) []byte {
	if len(chunk) > 1 && chunk[len(chunk)-1] == '\n' && chunk[len(chunk)-2] == '\r' {
		return chunk[:len(chunk)-2]
	}
	return chunk[:len(chunk)-1]
}

// Comment Collector
// -----------------

type commentCollector struct {
	Scanner
}

func (c *commentCollector) Start(chunk []byte) ([]byte, bool) {
	if bytes.HasPrefix(chunk, c.Comment()) {
		return chunk[len(c.Comment()):], true
	}
	return chunk, false
}

func (c *commentCollector) End(chunk []byte) ([]byte, bool) {
	if bytes.HasSuffix(chunk, []byte{'\n'}) {
		return removeSeparator(chunk), true
	}
	return chunk, false
}

// newCommentCollector returns a new comment collector.
// It is used by the scanner to collect comments.
// A comment is a line starting with the comment prefix.
func newCommentCollector(s Scanner) collector {
	return &commentCollector{s}
}

// quoteType is a function that returns a new quote collector for a giver Scanner
// It is used in WithQuote() scanner option.
// There are two types of quote collectors (for the moment): strict and fuzzy.
type quoteType func(Scanner) collector

// Quote Collector
// ---------------

// quoteCollector is a parent type for other quote collectors.
type quoteCollector struct {
	Scanner
}

// end checks if the chunk ends with an unescaped quote, in which case it removes it.
// It is used by End of some derived quote collectors.
func (c *quoteCollector) end(chunk []byte) ([]byte, bool) {
	if len(chunk) == 0 || chunk[len(chunk)-1] != c.Quote() {
		return chunk, false
	}
	escaped := false
	for i := len(chunk) - 2; i >= 0 && chunk[i] == c.Escape(); i-- {
		escaped = !escaped
	}
	if escaped {
		return chunk, false
	}
	return chunk[:len(chunk)-1], true
}

// Quote Collector : Strict
// ------------------------

type quoteCollectorStrict struct {
	quoteCollector
}

func (c *quoteCollectorStrict) Start(chunk []byte) ([]byte, bool) {
	if len(chunk) > 0 && chunk[0] == c.Quote() {
		return chunk[1:], true
	}
	return chunk, false
}

func (c *quoteCollectorStrict) End(chunk []byte) ([]byte, bool) {
	if v, ok := c.end(removeSeparator(chunk)); ok {
		return v, true
	}
	return chunk, false
}

// QuoteStrict can be used as paramter of WithQuote() scanner option.
// QuoteStrict do not allow spaces between the quote and the separator.
var QuoteStrict quoteType = quoteStrict

// quoteStrict is QuoteStrict but hidden from the doc.
func quoteStrict(s Scanner) collector {
	return &quoteCollectorStrict{quoteCollector{s}}
}

// Quote Collector : Fuzzy
// -----------------------

// Some spaces are allowed before and after the separator.
// If tab is used as separator, then we can't find tabs outisde of the quotes.
// If the separator is a space, this collector make no sens because is equivalent to the strict collector.
// So tab is always treated as space.
type quoteCollectorFuzzy struct {
	quoteCollector
}

func (c *quoteCollectorFuzzy) Start(chunk []byte) ([]byte, bool) {
	i := 0
	for i < len(chunk) && (chunk[i] == ' ' || chunk[i] == '\t') {
		i++
	}
	if i < len(chunk) && chunk[i] == c.Quote() {
		return chunk[i+1:], true
	}
	return chunk, false
}

func (c *quoteCollectorFuzzy) End(chunk []byte) ([]byte, bool) {
	v := removeSeparator(chunk)
	i := len(v) - 1
	for i >= 0 && (v[i] == ' ' || v[i] == '\t') {
		i--
	}
	if v, ok := c.end(v[:i+1]); ok {
		return v, true
	}
	return chunk, false
}

// QuoteFuzzy can be used as paramter of WithQuote() scanner option.
// QuoteFuzzy allows spaces between the quote and the separator.
// These spaces are ignored.
var QuoteFuzzy quoteType = quoteFuzzy

// quoteFuzzy is QuoteFuzzy but hidden from the doc.
func quoteFuzzy(s Scanner) collector {
	return &quoteCollectorFuzzy{quoteCollector{s}}
}
