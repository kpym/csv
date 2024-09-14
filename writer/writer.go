package writer

import (
	"bufio"
	"bytes"
	"errors"
	"io"
)

// Writer interface
type Writer interface {
	// WriteByteField writes a single CSV record along with any necessary quoting and escaping.
	WriteByteField(field []byte)

	// WriteStringField writes a single CSV record along with any necessary quoting and escaping.
	WriteStringField(field string)

	// NewRow writes the end-of-line marker only if not at the beginning of a line.
	NewRow()

	// WriteByteComment writes a (multi-line) comment
	WriteByteComment(comment []byte)

	// WriteStringComment writes a (multi-line) comment
	WriteStringComment(comment string)

	// EmptyRow writes an empty line followed by the end-of-line marker.
	EmptyRow()

	// Flush writes any buffered data to the underlying io.Writer.
	Flush()

	// Error reports any error that has occurred during a previous Write or Flush.
	Error() error

	// AtRowStart returns true if NewRow() or WriteByteComment() was called before
	AtRowStart() bool
}

type writer struct {
	bufw    *bufio.Writer // underlying buffered writer
	err     error         // error encountered by the writer
	sep     byte          // separator character (default ',')
	quote   byte          // quote character (default '"')
	escape  byte          // escape character (default '"')
	comment []byte        // comment characters (default "#")

	qsnl      string            // string used by bytes.indexAny to find quote, sep, \n or \r
	toEnquote func([]byte) bool // function to enquote a field

	atRowStart bool // true if at the beginning of a line
}

// Option is a function that sets an option on the writer.
// Option is in general the return value of With... functions.
type Option func(*writer)

// WithSeparator sets the field separator character.
func WithSeparator(sep byte) Option {
	return func(w *writer) {
		w.sep = sep
	}
}

// WithQuote sets the quote and the escape characters to the same value quote.
// Quote character cannot be newline or carriage return or same as the separator.
func WithQuote(quote byte) Option {
	return func(w *writer) {
		w.quote = quote
		w.escape = quote
	}
}

// WithEscape sets the escape character.
// It should be called after WithQuote and only if it is different from the quote character.
// Escape character cannot be newline or carriage return or same as the separator.
func WithEscape(escape byte) Option {
	return func(w *writer) {
		w.escape = escape
	}
}

// WithComment sets the comment characters.
// Comment characters should not be the same as the separator, the quote, newline or carriage return.
func WithComment(comment []byte) Option {
	return func(w *writer) {
		w.comment = comment
	}
}

// WithEnquoteAny force enquote any field.
func WithEnquoteAny() Option {
	return func(w *writer) {
		w.toEnquote = func([]byte) bool { return true }
	}
}

// WithEnquoteMinimal enquote only if necessary.
func WithEnquoteMinimal() Option {
	return func(w *writer) {
		w.toEnquote = func(data []byte) bool {
			return w.hasQuoteSep(data)
		}
	}
}

// WithEnquoteNonNumeric enquote all non-numeric fields.
// TODO: implement

// validate checks if the options are valid
// and set an error if they are not.
// It also sets the qsnl string used by hasQuoteSep.
// It is called after all options are processed.
func (w *writer) validate() {
	if w.quote == '\n' || w.quote == '\r' || w.quote == w.sep {
		w.err = errors.New("quote character cannot be newline or carriage return or same as the separator")
	}
	if w.escape == '\n' || w.escape == '\r' || w.escape == w.sep {
		w.err = errors.New("escape character cannot be newline or carriage return or same as the separator")
	}
	if w.sep == '\n' || w.sep == '\r' {
		w.err = errors.New("separator character cannot be newline or carriage return")
	}
	if bytes.ContainsAny(w.comment, string([]byte{w.sep, w.quote, '\n', '\r'})) {
		w.err = errors.New("comment character should not be the same as the separator, quote, newline or carriage return")
	}

	w.setqsnl()
}

// options run the given options on the writer
func (w *writer) options(options ...Option) {
	for _, opt := range options {
		opt(w)
	}
	w.validate()
}

// DefaultOptions are the default options for a writer.
var DefaultOptions = []Option{
	WithSeparator(','),
	WithQuote('"'),
	WithEnquoteMinimal(),
	WithComment([]byte("# ")),
}

// New returns a new Writer that writes to w.
func New(w io.Writer, opts ...Option) Writer {
	csvw := &writer{
		bufw:       bufio.NewWriter(w),
		atRowStart: true,
	}
	csvw.options(DefaultOptions...)
	csvw.options(opts...)
	return csvw
}

// setsqnl sets the qsnl string used by hasQuoteSep.
// It is called after all options are processed.
func (w *writer) setqsnl() {
	w.qsnl = string([]byte{w.quote, w.sep, '\n', '\r'})
}

// hasQuoteSep returns true if data contains any of the quote, sep, \n or \r characters.
func (w *writer) hasQuoteSep(data []byte) bool {
	return bytes.ContainsAny(data, w.qsnl)
}

// write is an internal function to write data to the underlying writer and set the error.
// If an error is already set, it does nothing.
func (w *writer) write(data []byte) {
	if w.err != nil {
		return
	}
	_, w.err = w.bufw.Write(data)
}

// writeByte is an internal function to write a byte to the underlying writer and set the error.
// If an error is already set, it does nothing.
func (w *writer) writeByte(c byte) {
	if w.err != nil {
		return
	}
	w.err = w.bufw.WriteByte(c)
}

// writeEscaped writes data to the underlying writer with escaped quote characters.
// If an error is encountered, it is saved and can be recovered using Error().
func (w *writer) writeEscaped(b []byte) {
	for len(b) > 0 {
		n := bytes.IndexByte(b, w.quote)
		if n == -1 {
			w.write(b)
			return
		}
		w.write(b[:n])
		w.writeByte(w.escape)
		w.writeByte(w.quote)
		b = b[n+1:]
	}
}

// WriteByteField writes a single CSV record to w along with any necessary quoting and escaping.
func (w *writer) WriteByteField(field []byte) {
	if !w.atRowStart {
		w.writeByte(w.sep)
	}
	w.atRowStart = false
	if w.toEnquote(field) {
		w.writeByte(w.quote)
		w.writeEscaped(field)
		w.writeByte(w.quote)
	} else {
		w.write(field)
	}
}

// WriteStringField writes a single CSV record to w along with any necessary quoting and escaping.
func (w *writer) WriteStringField(field string) {
	w.WriteByteField([]byte(field))
}

// NewRow writes the end-of-line marker only if not at the beginning of a line.
func (w *writer) NewRow() {
	if !w.atRowStart {
		w.writeByte('\n')
	}
	w.atRowStart = true
}

// writeCommentLine writes a comment line followed by the end-of-line marker.
func (w *writer) writeCommentLine(data []byte) {
	if !w.atRowStart {
		w.writeByte('\n')
	}
	w.write(w.comment)
	w.write(data)
	w.writeByte('\n')
	w.atRowStart = true
}

// WriteByteComment writes a sequence of comment lines followed by the end-of-line marker.
func (w *writer) WriteByteComment(comment []byte) {
	comment = bytes.TrimRight(comment, "\r\n\t ")
	lines := bytes.Split(comment, []byte{'\n'})
	for _, line := range lines {
		line = bytes.Trim(line, "\r")
		w.writeCommentLine(line)
	}
}

// WriteStringComment writes a sequence of comment lines followed by the end-of-line marker.
func (w *writer) WriteStringComment(comment string) {
	w.WriteByteComment([]byte(comment))
}

// EmptyRow writes an empty row.
func (w *writer) EmptyRow() {
	if !w.atRowStart {
		w.writeByte('\n')
	}
	w.writeByte('\n')
	w.atRowStart = true
}

// Error returns any error encountered by the writer.
func (w *writer) Error() error {
	return w.err
}

// Flush writes any buffered data to the underlying io.Writer.
func (w *writer) Flush() {
	if w.err != nil {
		return
	}
	w.err = w.bufw.Flush()
}

// AtRowStart returns true if the writer is at the start of a row.
func (w *writer) AtRowStart() bool {
	return w.atRowStart
}
