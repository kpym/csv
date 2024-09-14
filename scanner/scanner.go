package scanner

import (
	"bufio"
	"bytes"
	"io"
)

// Scanner interface
type Scanner interface {
	// Separator returns the separator character (like ',', ';' or '\t').
	Separator() byte
	// Quote returns the quote character (like '"' or "'" or 0 if not quoted).
	Quote() byte
	// Escape returns the escape character (like '"' or '\' or 0 if not quoted).
	Escape() byte
	// Comment returns the comment prefix (like '#' or '\\' or nil if no comment).
	Comment() []byte

	// Scan recover next field, if false then error or end of file is reached.
	Scan() bool
	// Err() returns the first non-EOF error that was encountered by the Scanner.
	Err() error

	// Bytes return the current field as a byte slice.
	// Ordinary fields are returned without separators,
	// without bording quotes and with all other quotes unescaped.
	// Comment fields are returned without the comment prefix.
	// This value is valid only until the next call to Scan().
	Bytes() []byte
	// Offset returns the offset in bytes of the current field in the input.
	Offset() int

	// AtRowStart returns true if the current field is the first field of the row.
	AtRowStart() bool
	// AtRowEnd returns true if the current field is the last field of the row.
	AtRowEnd() bool

	// IsComment returns true if the current field is a comment.
	IsComment() bool
	// IsQuoted returns true if the current field is quoted.
	IsQuoted() bool
	// IsEmptyLine returns true if the current field is an empty line.
	IsEmptyLine() bool
}

// scanner is the default implementation of Scanner.
// It trats only the standard case (no space separated fields)
type scanner struct {
	// Parameters
	src     bufio.Scanner // source scanner that scans to separator or end of line
	sep     byte          // separator character (default ',')
	quote   byte          // quote character (default '"')
	escape  byte          // escape character (default '"')
	comment []byte        // comment characters (default "#")

	// Collectors
	quoteCollector   collector
	commentCollector collector

	// check if the field is empty (depends on the separator)
	empty func([]byte) bool

	// State variables that are set during scanning
	value      []byte // the field value returned by Bytes() (without delimiters, comment prefix, bording quotes and escapes)
	rawlen     int    // length of the raw value (including quotes and separator) used only to compute offset
	offset     int    // offset of the field in the input (starting at 0)
	isComment  bool   // true if the field is a comment
	isQuoted   bool   // true if the field is enquoted (first and last bytes are quotes)
	atRowStart bool   // true if the field is the first one in the row
	atRowEnd   bool   // true if the field is the last one in the row
}

// sepScan is a function that returns a split function for bufio.Scanner.
// This function stops at the first separator or line end.
// All fields end with a delimiter or newline (`\n`).
// The last field of the last row is always followed by a `\n` (even if it's empty or missing).
// If the separator is `\n` or 0, only the end of line is used as a separator.
func sepScan(s byte) bufio.SplitFunc {
	// indexAny looks for the first separator or end of line
	// it could be implemented with bytes.IndexAny but it's faster (I think) this way
	// check https://github.com/golang/go/issues/60550
	var indexAny func(data []byte) int
	switch s {
	case '\n', 0:
		// no separator, only end of line
		indexAny = func(data []byte) int {
			return bytes.IndexByte(data, '\n')
		}
	default:
		indexAny = func(data []byte) int {
			sepi := bytes.IndexByte(data, s)
			if sepi == -1 {
				return bytes.IndexByte(data, '\n')
			}
			if nli := bytes.IndexByte(data[:sepi], '\n'); uint(nli) < uint(sepi) {
				return nli
			}
			return sepi
		}
	}

	return func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}
		if i := indexAny(data); i >= 0 {
			// return up to the separator (including it)
			return i + 1, data[:i+1], nil
		}
		// If we're at EOF, the remaining data has no separator
		// and is the last field. Return it with '\n appended.
		if atEOF {
			data = append(data, '\n')
			return 0, data, bufio.ErrFinalToken
		}
		// Request more data.
		return 0, nil, nil
	}
}

// unescapeQuotes transform escaped quotes to quotes for the current field s.value.
// Only <esc><quote> → <quote> is done. The non escaped quotes are preserved.
// The starting and ending quotes should be removed before calling this function,
// but even if they are not, they will stay unchanged (as they are not escaped).
func (s *scanner) unescapeQuotes() {
	// if the field is not quoted, return it as is
	if len(s.value) == 0 || s.escape == 0 {
		return
	}
	eq := []byte{s.Escape(), s.Quote()}
	n, m := 0, 0
	for {
		m = bytes.Index(s.value[n:], eq)
		if m == -1 {
			break
		}
		s.value = append(s.value[:n+m], s.value[n+m+1:]...)
		n += m + 1
	}
}

// isEmpty returns true if the data is empty
// used to check if a line is empty if the separator is space
func isEmpty(data []byte) bool {
	return len(data) == 0
}

// onlySpaces returns true if the data is only spaces
// used to check if a line is empty if the separator is tab
func onlySpaces(data []byte) bool {
	for _, b := range data {
		if b != ' ' {
			return false
		}
	}
	return true
}

// onlyWhiteSpaces returns true if the data is only spaces or tabs
// used to check if a line is empty if the separator is not space or tab
func onlyWhiteSpaces(data []byte) bool {
	for _, b := range data {
		if b != ' ' && b != '\t' {
			return false
		}
	}
	return true
}

type Option func(*scanner)

// WithSeparator sets the separator character.
// If sep is '\n' or 0, the csv data is expected to have only one column.
func WithSeparator(sep byte) Option {
	return func(s *scanner) {
		s.sep = sep
		// set the split function for bufio.Scanner
		s.src.Split(sepScan(sep))
		// set the empty function to check if a field is empty
		switch sep {
		case ' ':
			s.empty = isEmpty
		case '\t':
			s.empty = onlySpaces
		default:
			s.empty = onlyWhiteSpaces
		}
	}
}

// WithQuote sets the quote and escape characters and the quote type.
// The quote type could be QuoteStrict or QuoteFuzzy.
// If quote is 0 or qt is nil, no unquoting is done.
func WithQuote(quote byte, qt quoteType) Option {
	return func(s *scanner) {
		s.quote = quote
		s.escape = quote
		if quote != 0 && qt != nil {
			s.quoteCollector = qt(s)
		} else {
			s.quoteCollector = nil
		}
	}
}

// WithEscape sets the escape character.
// It should be called after WithQuote and only if it is different from the quote character.
func WithEscape(escape byte) Option {
	return func(s *scanner) {
		s.escape = escape
	}
}

// WithComment sets the comment prefix.
func WithComment(comment []byte) Option {
	return func(s *scanner) {
		s.comment = comment
		if len(comment) > 0 {
			s.commentCollector = newCommentCollector(s)
		} else {
			s.commentCollector = nil
		}
	}
}

var DefaultOptions = []Option{
	WithSeparator(','),
	WithQuote('"', QuoteFuzzy),
	WithComment([]byte("#")),
}

// NewScanner returns a new Scanner to read from r.
func New(r io.Reader, options ...Option) Scanner {
	s := &scanner{
		// underlying bufio.Scanner
		src: *bufio.NewScanner(r),
		// initial state
		// the first call to Scan() will switch AtRowStart to true and AtRowEnd to false
		// because this is what happens after the last field of a row
		atRowEnd: true,
	}

	// set default options
	s.Options(DefaultOptions...)
	// set custom options
	s.Options(options...)

	return s
}

// Options run the given options on the scanner
func (s *scanner) Options(options ...Option) {
	for _, opt := range options {
		opt(s)
	}
}

// Separator returns the separator character
func (s *scanner) Separator() byte {
	return s.sep
}

// Quote returns the quote character
func (s *scanner) Quote() byte {
	return s.quote
}

// Escape returns the escape character
func (s *scanner) Escape() byte {
	return s.escape
}

// Comment returns the comment prefix
func (s *scanner) Comment() []byte {
	return s.comment
}

func (s *scanner) Scan() bool {
	// if we were at the end of the row, we are now at the start of the next row
	s.atRowStart = s.atRowEnd
	// add the length of the previous field to the offset
	s.offset += s.rawlen
	// reset field values
	// these values are set during the scan
	s.value = s.value[:0]
	s.rawlen = 0
	s.atRowEnd = false
	s.isComment = false
	s.isQuoted = false
	// start collecting data
	var collector collector = nil
	var start, stop bool // temporary variables for the collector
	var ready bool       // ready to deliver the field ?
	for s.src.Scan() {
		data := s.src.Bytes()
		s.rawlen += len(data)
		// check if we are at the end of the line
		// the chunk data is always terminated by a separator
		s.atRowEnd = data[len(data)-1] == '\n'
		// are we in the middle of a field?
		if collector != nil {
			// we are collecting data for a field
			data, stop = collector.End(data)
			s.value = append(s.value, data...)
			// do we need more data to end the field?
			if !stop {
				continue
			}
		} else {
			// check if we are starting a comment
			if s.atRowStart && s.commentCollector != nil {
				data, start = s.commentCollector.Start(data)
				if start {
					// we are starting a comment and data is without the comment prefix
					s.isComment = true
					data, stop = s.commentCollector.End(data)
					s.value = append(s.value, data...)
					// do we need more data to end the comment?
					if !stop {
						collector = s.commentCollector
						continue
					}
				}
			}
			// check if we are starting a quoted field (only if comment was not collected)
			if !s.isComment && s.quoteCollector != nil {
				data, start = s.quoteCollector.Start(data)
				if start {
					// we are starting a quoted field
					s.isQuoted = true
					data, stop = s.quoteCollector.End(data)
					s.value = append(s.value, data...)
					// do we need more data to end the quoted field?
					if !stop {
						collector = s.quoteCollector
						continue
					}
				}
			}
			// normal field
			if !s.isComment && !s.isQuoted {
				s.value = append(s.value, removeSeparator(data)...)
			}
		}
		// end collecting data
		ready = true
		break
	}
	if !ready && s.src.Err() != nil {
		// an error occurred during the scan
		return false
	}
	if !ready && collector != nil {
		// we were collecting a field but the end of the file was reached
		// this could be a comment without a line break at the end of the file or
		// a quoted field without a closing quote (we hides this error)
		s.atRowEnd = true
		ready = true
	}
	if !ready {
		// no more data to deliver
		return false
	}
	// do we need to unescape quotes?
	if s.isQuoted {
		s.unescapeQuotes()
	}
	// we have a field
	return true
}

func (s *scanner) Err() error {
	return s.src.Err()
}

func (s *scanner) Bytes() []byte {
	return s.value
}

func (s *scanner) Offset() int {
	return s.offset
}

func (s *scanner) AtRowStart() bool {
	return s.atRowStart
}

func (s *scanner) AtRowEnd() bool {
	return s.atRowEnd
}

func (s *scanner) IsComment() bool {
	return s.isComment
}

func (s *scanner) IsQuoted() bool {
	return s.isQuoted
}

func (s *scanner) IsEmptyLine() bool {
	return s.AtRowStart() && s.AtRowEnd() && s.empty(s.Bytes())
}
