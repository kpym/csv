package sniffer

import (
	"bytes"
	"io"
	"sort"

	"github.com/kpym/csv/scanner"
)

// Parameters contains the parameters to configure the scanner.
// These parameters are guessed by the Sniffer.
type Parameters struct {
	Separator byte
	Quote     byte
	Escape    byte
	Comment   []byte
}

// NewScanner creates a new scanner with the guessed parameters.
func (p *Parameters) NewScanner(r io.Reader) scanner.Scanner {
	if p == nil {
		// default parameters
		return scanner.New(r)
	}
	return scanner.New(r,
		scanner.WithSeparator(p.Separator),
		scanner.WithQuote(p.Quote, scanner.QuoteFuzzy),
		scanner.WithEscape(p.Escape),
		scanner.WithComment(p.Comment),
	)
}

// SepQuoteScore is a pair of separator and quote character with a score.
// The score is used to determine the best pair.
// Ordered slice of SepQuoteScore are generated by Sniffer.Sniff().
type SepQuoteScore struct {
	Sep   byte
	Quote byte
	Score int
}

// Sniffer is used to guess the parameters of a CSV file from a sample of the beginning of the file.
// The Sniffer is used to guess the separator, quote, escape, and comment characters.
// If strict mode is enabled, the Sniffer will return 0 or nil if it can't guess some parameters.
type Sniffer struct {
	// data contains the data to sniff
	data []byte

	// set of possible separator characters (e.g. ',', ';', '|', '\t', ' ')
	seps []byte
	// set of possible quote characters (e.g. '"', '\'', '`')
	quotes []byte
	// set of possible escape characters (e.g. '"'(EscapeSameAsQuote), '\')
	escapes []byte
	// set of possible comment prefixes (e.g. '#', '//')
	comments [][]byte
	// strict mode
	strict bool
}

// Options for Sniffer.
type Option func(*Sniffer)

// NewSniffer creates a new Sniffer taking the possible separator and quote characters as optional arguments.
// If no arguments are passed, the default values are used.
func NewSniffer(data []byte, opts ...Option) *Sniffer {
	s := Sniffer{data: data}
	s.Options(DefaultOptions...)
	s.Options(opts...)
	return &s
}

// Options sets the options for the Sniffer.
func (s *Sniffer) Options(opts ...Option) {
	for _, opt := range opts {
		opt(s)
	}
}

// DefaultOptions are the default options for a Sniffer.
var DefaultOptions = []Option{
	PossibleSeparators([]byte{',', ';', '\t', '|', '&'}),
	PossibleQuotes([]byte{'"', '\'', '`'}),
	PossibleEscapes([]byte{EscapeSameAsQuote, '\\'}),
	PossibleComments([][]byte{{'#'}, {'/', '/'}}),
	Strict(false),
}

// duplicate returns a copy of the byte slice.
// This is an utility function used by the Options.
func duplicate(b []byte) []byte {
	return append([]byte(nil), b...)
}

// PossibleSeparators sets the possible separator characters.
// Example: PossibleSeparators([]byte(",;\t|&"))
func PossibleSeparators(seps []byte) Option {
	return func(s *Sniffer) {
		s.seps = duplicate(seps)
	}
}

// PossibleQuotes sets the possible quote characters.
// Example: PossibleQuotes([]byte("\"'`"))
func PossibleQuotes(quotes []byte) Option {
	return func(s *Sniffer) {
		s.quotes = duplicate(quotes)
	}
}

// EscapeSameAsQuote is used to indicate that the escape character is the same as the quote character.
const EscapeSameAsQuote = byte(0xFF)

// PossibleEscapes sets the possible escape characters.
// Example: PossibleEscapes([]byte{sniffer.EscapeSameAsQuote, '\\'})
func PossibleEscapes(escapes []byte) Option {
	return func(s *Sniffer) {
		s.escapes = duplicate(escapes)
	}
}

// PossibleComments sets the possible comment prefixes.
// Example: PossibleComments([][]byte{{'#'}, {'/', '/'}})
func PossibleComments(comments [][]byte) Option {
	return func(s *Sniffer) {
		s.comments = make([][]byte, len(comments))
		for i := range comments {
			s.comments[i] = duplicate(comments[i])
		}
	}
}

// Strict sets the strict mode.
// If strict is true, the Sniffer will return 0 or nil if it can't guess some parameters.
func Strict(strict bool) Option {
	return func(s *Sniffer) {
		s.strict = strict
	}
}

// GuessParameters returns the most probable parameters.
// In strict mode, it will return nil if it can't verify the parameters.
func (s *Sniffer) GuessParameters() (p *Parameters, verified bool) {
	comment := s.GuessComment()      // could be nil
	scores := s.GuessSepQuoteScore() // could be [{0,0,0}]
	toVerify := []bool{true, false}
	if s.strict {
		toVerify = []bool{true}
	}
	for _, verify := range toVerify {
		for _, sqs := range scores {
			escape := s.GuessEscape(byte(sqs.Quote)) // could be 0
			p := &Parameters{
				Separator: sqs.Sep,   // could be 0
				Quote:     sqs.Quote, // could be 0
				Escape:    escape,    // could be 0
				Comment:   comment,   // could be nil
			}
			// in the second pass, we return the most probable (not verified) parameters
			if !verify || checkRowsLen(s.data, p) {
				return p, verify
			}
		}
	}

	// can't guess the parameters
	return nil, false
}

const (
	// bonus if the data starts with a comment character
	commentStartBonus = 10
	// bonus if a line starts with a comment character followed by a space
	commentSpaceBonus = 2
	// bonus if a line starts with a comment character
	commentBonus = 1
)

// GuessComment returns the most probable comment character.
// If no comment character is found, in strict mode, it will return nil,
// else it will return the first possible prefix or nil.
func (s *Sniffer) GuessComment() []byte {
	if len(s.comments) == 0 {
		return nil
	}
	// find the most probable comment line prefix
	var (
		max     int
		comment []byte
	)
	for p := range s.comments {
		score := 0
		// if it starts with s.comments[p]
		if bytes.HasPrefix(s.data, s.comments[p]) {
			score += commentStartBonus
		}
		// count the number of lines starting with s.comments[p]
		nlcom := append([]byte{'\n'}, s.comments[p]...)
		score += bytes.Count(s.data, nlcom) * commentBonus
		nlcom = append(nlcom, ' ')
		score += bytes.Count(s.data, nlcom) * commentSpaceBonus
		if score > max {
			max = score
			comment = s.comments[p]
		}
	}
	if max == 0 {
		// no comment character found
		if s.strict {
			return nil
		} else {
			return s.comments[0]
		}
	}
	// return the most probable comment line prefix
	return comment
}

// BestSepQuote returns the most probable separator and quote character.
// If no separator is found and the mode is strict, 0 is returned,
// else the first possible separator is returned.
// If no quote is found and the mode is strict, 0 is returned,
// else the first possible quote is returned.
func (s *Sniffer) BestSepQuote() (sep, quote byte) {
	sqs := s.GuessSepQuoteScore()
	if len(sqs) > 0 {
		return sqs[0].Sep, sqs[0].Quote
	}
	// we should never reach this point because
	// GuessSepQuoteScore returns at least one element,
	// but in case...
	if len(s.seps) > 0 && !s.strict {
		sep = s.seps[0]
	}
	if len(s.quotes) > 0 && !s.strict {
		quote = s.quotes[0]
	}
	return sep, quote
}

// GuessSepQuoteScore returns the ordered list of pairs of separator and quote character.
// If no separator is found and the mode is strict, 0 is used,
// else the first possible separator is used.
// If no quote is found and the mode is strict, 0 is used,
// else the first possible quote is used.
// The result contains at least one element, that could be {0, 0, 0}.
func (s *Sniffer) GuessSepQuoteScore() []SepQuoteScore {
	// prepare the maps
	t := s.newTempStats()

	if len(t.seps) == 0 {
		// no separator character found
		if len(s.seps) > 0 && !s.strict {
			t.seps[s.seps[0]] = 0
		} else {
			t.seps[0] = 0
		}
	}
	if len(t.quotes) == 0 {
		// no quote character found
		if len(s.quotes) > 0 && !s.strict {
			t.quotes[s.quotes[0]] = 0
		} else {
			t.quotes[0] = 0
		}
	}

	// the number of all pairs of separator and quote characters
	lenSQS := len(t.seps) * len(t.quotes)

	// convert the tempStats to a slice of SepQuoteScore
	sqs := make([]SepQuoteScore, 0, lenSQS)
	for quote, qvalue := range t.quotes {
		for sep, svalue := range t.seps {
			value := qvalue + svalue + t.pairs[sqPair{sep, quote}]
			sqs = append(sqs, SepQuoteScore{sep, quote, value})
		}
	}
	// sort by score
	sort.Slice(sqs, func(i, j int) bool {
		return sqs[i].Score > sqs[j].Score
	})

	return sqs
}

// GuessEscape returns the most probable escape character for the given quote character.
// If no possible escape character is given, returns 0 (no-escape)
// If no escape character is found and the mode is strict, 0 is returned,
// else the first possible escape character is returned.
func (s *Sniffer) GuessEscape(quote byte) byte {
	switch len(s.escapes) {
	case 0:
		// nothing to guess
		return 0
	case 1:
		// no need to guess
		if !s.strict {
			return eq(s.escapes[0], quote)
		}
	}
	// collect the score of each escape character
	score := make(map[byte]int, len(s.escapes))
	for _, c := range s.escapes {
		score[eq(c, quote)] = 0
	}
	for i := 1; i < len(s.data); i++ {
		if s.data[i] == quote {
			if _, ok := score[s.data[i-1]]; ok {
				score[s.data[i-1]]++
			}
		}
	}
	// find the escape character with the highest score
	var escape byte = eq(s.escapes[0], quote)
	var max int
	for c, s := range score {
		if s > max {
			max = s
			escape = c
		}
	}
	if max == 0 && s.strict {
		// can't guess the escape character
		return 0
	}
	// return the most probable escape character
	return escape
}

// eq normalizes the escape character,
// ie replace EscapeSameAsQuote by the quote character.
func eq(escape, quote byte) byte {
	if escape == EscapeSameAsQuote {
		return quote
	}
	return escape
}

// checkRowsLen checks if all rows have the same number of columns.
// This is used to check if parameters are correct.
// If oneIsOk is true, it will return true even if only one row is present.
func checkRowsLen(data []byte, p *Parameters) bool {
	scan := p.NewScanner(bytes.NewReader(data))
	numCols := 0
	numRows := 0
	colsInThisRow := 0
	for scan.Scan() {
		// skip comments and empty lines
		if scan.IsComment() || scan.IsEmptyLine() {
			continue
		}
		// we use AtRowStart to avoid an incomplete row at the end of the data
		if scan.AtRowStart() {
			// if the last row has a different number of columns than the first row
			if numRows > 1 && colsInThisRow != numCols {
				return false
			}
			// move to next row, if not enough rows have been checked
			numRows++
			colsInThisRow = 0
		}
		// count the number of columns
		if numRows == 1 {
			numCols++
		} else {
			colsInThisRow++
		}
	}
	// in case of error, we can't verify
	if scan.Err() != nil {
		return false
	}
	// if at least two rows have same number of columns > 1
	if numRows > 2 && numCols > 1 {
		return true
	}
	// let's give a chance to the last row if it is the second one
	if numRows == 2 && numCols > 1 && colsInThisRow == numCols {
		return true
	}
	// only one row or only one column (no separator) => can't verify
	return false
}
