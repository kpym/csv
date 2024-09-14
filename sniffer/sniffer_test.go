package sniffer

import (
	"bytes"
	"fmt"
	"testing"
)

func TestGuessParametersNoData(t *testing.T) {
	data := [][]byte{
		{},
		[]byte("    "),
		[]byte(" a b c "),
	}

	for _, d := range data {
		s := NewSniffer(
			d,
			PossibleSeparators([]byte{',', ';', '|', '\t'}),
			PossibleQuotes([]byte{'"', '\'', '`'}),
			PossibleComments([][]byte{[]byte("#"), []byte("//")}),
			Strict(true),
		)
		p, v := s.GuessParameters()
		if p != nil || v {
			t.Errorf("Expected nil, false; got %v, %t", p, v)
		}
	}
}

func TestGuessParameters(t *testing.T) {
	equal := func(a, b *Parameters) bool {
		if a == nil || b == nil {
			return a == b
		}
		return a.Separator == b.Separator && a.Quote == b.Quote && a.Escape == b.Escape && bytes.Equal(a.Comment, b.Comment)
	}

	tests := []struct {
		data     []byte
		mode     bool
		want     *Parameters
		verified bool
	}{
		{
			[]byte(`a,b,c`),
			false,
			&Parameters{Separator: ',', Quote: '"', Escape: '"', Comment: []byte{'#'}},
			false,
		},
		{
			[]byte("a;'b''c''';d\ne;f;g\n"),
			false,
			&Parameters{Separator: ';', Quote: '\'', Escape: '\'', Comment: []byte{'#'}},
			true,
		},
		{
			[]byte(`a,b,c`),
			true,
			nil,
			false,
		},
		{
			[]byte("a;'b''c''';d\ne;f;g\n"),
			true,
			&Parameters{Separator: ';', Quote: '\'', Escape: '\'', Comment: nil},
			true,
		},
	}
	for _, test := range tests {
		s := NewSniffer(test.data, Strict(test.mode))
		if p, v := s.GuessParameters(); !equal(p, test.want) || v != test.verified {
			t.Errorf("GuessParameters(%q) = %v, %t, want %v, %t", test.data, p, v, test.want, true)
		}
	}
}

func TestBestSepQuoteEmptyData(t *testing.T) {
	data := [][]byte{
		{},
		[]byte("    "),
		[]byte(" a b c "),
	}

	for _, d := range data {
		s := NewSniffer(
			d,
			PossibleSeparators([]byte{',', ';', '|', '\t'}),
			PossibleQuotes([]byte{'"', '\'', '`'}))
		sep, quote := s.BestSepQuote()
		if sep != byte(',') || quote != byte('"') {
			t.Errorf("Expected ',', '\"'; got %c, %c", sep, quote)
		}
	}
}

func TestGuessCommentNoData(t *testing.T) {
	data := [][]byte{
		{},
		[]byte("    "),
		[]byte(" a b c "),
	}

	for _, d := range data {
		s := NewSniffer(
			d,
			PossibleSeparators([]byte{',', ';', '|', '\t'}),
			PossibleQuotes([]byte{'"', '\'', '`'}),
			PossibleComments([][]byte{[]byte("#"), []byte("//")}),
			Strict(true),
		)
		comment := s.GuessComment()
		if comment != nil {
			t.Errorf("Expected nil; got %s", comment)
		}
	}
	for _, d := range data {
		s := NewSniffer(
			d,
			PossibleSeparators([]byte{',', ';', '|', '\t'}),
			PossibleQuotes([]byte{'"', '\'', '`'}),
			PossibleComments([][]byte{[]byte("#"), []byte("//")}),
			Strict(false),
		)
		comment := s.GuessComment()
		if !bytes.Equal(comment, []byte{'#'}) {
			t.Errorf("Expected #; got %s", comment)
		}
	}
}

func TestGuessComment(t *testing.T) {
	tests := []struct {
		data     []byte
		possible [][]byte
		strict   bool
		want     []byte
	}{
		{[]byte("a,b,c"), [][]byte{{'#'}, {'%'}, {'/', '/'}}, true, nil},                        // no comment
		{[]byte("a,b,c"), [][]byte{{'#'}, {'%'}, {'/', '/'}}, false, []byte{'#'}},               // no comment
		{[]byte("a,b,c\n\n//\n%"), [][]byte{{'#'}, {'%'}, {'/', '/'}}, true, []byte{'%'}},       // first comment is the winner
		{[]byte("a,b,c\n\n// \n%"), [][]byte{{'#'}, {'%'}, {'/', '/'}}, true, []byte{'/', '/'}}, // coment with space is the winner
		{[]byte("#a,b,c\n\n// \n%"), [][]byte{{'#'}, {'%'}, {'/', '/'}}, true, []byte{'#'}},     // starting with comment is the winner
	}
	for _, test := range tests {
		s := NewSniffer(test.data, PossibleComments(test.possible), Strict(test.strict))
		if got := s.GuessComment(); !bytes.Equal(got, test.want) {
			t.Errorf("GuessComment(%q, %q) = %q, want %q", test.data, test.possible, got, test.want)
		}
	}
}

func TestBestSepQuote(t *testing.T) {
	tests := []struct {
		data      []byte
		seps      []byte
		quotes    []byte
		wantSep   byte
		wantQuote byte
	}{
		{[]byte(`a,b,c`), []byte{',', ';'}, []byte{'"', '\''}, ',', '"'},
		{[]byte(`a,b,c`), []byte{',', ';'}, []byte{'\'', '"'}, ',', '\''},
		{[]byte(`a;"b""c""";"d\n\\e"`), []byte{',', ';'}, []byte{'"', '\''}, ';', '"'},
	}
	for _, test := range tests {
		s := NewSniffer(test.data, PossibleQuotes(test.quotes), PossibleSeparators(test.seps))
		if gotSep, gotQuote := s.BestSepQuote(); gotSep != test.wantSep || gotQuote != test.wantQuote {
			t.Errorf("GuessSepQuote(%q, %q, %q) = %q, %q, want %q, %q", test.data, test.seps, test.quotes, gotSep, gotQuote, test.wantSep, test.wantQuote)
		}
	}
}

func ExampleSniffer_GuessSepQuoteScore() {
	data := []byte(`a; "b|c" ; " d' ";e
1,1;2,2;3,3;4,4
`)
	s := NewSniffer(data, PossibleSeparators([]byte{',', ';', '|', '\t'}), PossibleQuotes([]byte{'"', '\'', '`'}))
	for _, sqs := range s.GuessSepQuoteScore() {
		fmt.Printf("<%c><%c> → %d\n", sqs.Sep, sqs.Quote, sqs.Score)
	}
	// Output:
	// <;><"> → 26
	// <,><"> → 10
	// <|><"> → 7
}

func TestGuessEscape(t *testing.T) {
	tests := []struct {
		data     []byte
		quote    byte
		possible []byte
		want     byte
	}{
		{[]byte(`a,"b""c""","d\n\\e`), '"', nil, 0},
		{[]byte(`a,"b""c""","d\n\\e`), '"', []byte{EscapeSameAsQuote}, '"'},
		{[]byte(`a,"b""c""","d\n\\e`), '"', []byte{EscapeSameAsQuote, '\\'}, '"'},
		{[]byte(`a,'b''c''','d\n\\e`), '\'', []byte{EscapeSameAsQuote, '\\'}, '\''},
		{[]byte(`a,"b\"c\"","d\n\\e`), '"', []byte{EscapeSameAsQuote, '\\', '\''}, '\\'},
	}
	for _, test := range tests {
		s := NewSniffer(test.data, PossibleEscapes(test.possible))
		if got := s.GuessEscape(test.quote); got != test.want {
			t.Errorf("GuessEscape(%q, %q, %q) = %q, want %q", test.data, test.quote, test.possible, got, test.want)
		}
	}
}
