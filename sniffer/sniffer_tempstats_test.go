package sniffer

import (
	"testing"
)

func TestSniffer_newTempStats(t *testing.T) {
	// data, expected result pairs
	data := []struct {
		sniff     *Sniffer
		numSeps   int
		numQuotes int
		numPairs  int
	}{
		{NewSniffer(nil), 0, 0, 0},
		{NewSniffer(nil, PossibleSeparators([]byte{',', ';', '|', '\t'}), PossibleQuotes([]byte{'"', '\'', '`'})), 0, 0, 0},
		{NewSniffer([]byte(`"a","b","c"`)), 1, 1, 1},
		{NewSniffer([]byte(`"a","b";"c"`)), 2, 1, 2},
		{NewSniffer([]byte(`"a","b";'c'"`)), 2, 2, 3}, // , is never next to '
	}
	for _, d := range data {
		tmpSts := d.sniff.newTempStats()
		if tmpSts == nil {
			t.Errorf("Expected a TempStats object, got nil")
		} else {
			if len(tmpSts.seps) != d.numSeps {
				t.Errorf("Expected seps %v, got %v", d.sniff.seps, tmpSts.seps)
			}
			if len(tmpSts.quotes) != d.numQuotes {
				t.Errorf("Expected quotes %v, got %v", d.sniff.quotes, tmpSts.quotes)
			}
			if len(tmpSts.pairs) != d.numPairs {
				t.Errorf("Expected %d pairs, got %d", d.numPairs, len(tmpSts.pairs))
			}
		}
	}
}

func TestSniffer_initTempStats(t *testing.T) {
	// data, expected result pairs
	data := []struct {
		sniff *Sniffer
	}{
		{NewSniffer(nil)},
		{NewSniffer(nil, PossibleSeparators([]byte{',', ';', '|', '\t'}), PossibleQuotes([]byte{'"', '\'', '`'}))},
		{NewSniffer([]byte(`"a","b","c"`))},
	}
	for _, d := range data {
		tmpSts := initTempStats(d.sniff)
		if tmpSts == nil {
			t.Errorf("Expected a TempStats object, got nil")
		} else {
			if len(tmpSts.seps) != len(d.sniff.seps) {
				t.Errorf("Expected seps %v, got %v", d.sniff.seps, tmpSts.seps)
			}
			if len(tmpSts.quotes) != len(d.sniff.quotes) {
				t.Errorf("Expected quotes %v, got %v", d.sniff.quotes, tmpSts.quotes)
			}
			if len(tmpSts.pairs) != 0 {
				t.Errorf("Expected 0 pairs, got %d", len(tmpSts.pairs))
			}
		}
	}
}

func TestIsSepChar(t *testing.T) {
	// data, expected result pairs
	data := []struct {
		b byte
		r bool
	}{
		{',', true},
		{';', true},
		{'|', true},
		{'\t', true},
		{'\r', false},
		{'\n', false},
		{'"', false},
		{'\'', false},
		{'`', false},
		{'a', false},
		{'1', false},
	}
	tmpSts := initTempStats(NewSniffer(nil))
	for _, d := range data {
		if tmpSts.isSepChar(d.b) != d.r {
			t.Errorf("For %q, expected %v, got %v", d.b, d.r, tmpSts.isSepChar(d.b))
		}
	}
}

func TestIsQuoteChar(t *testing.T) {
	// data, expected result pairs
	data := []struct {
		b byte
		r bool
	}{
		{'"', true},
		{'\'', true},
		{'`', true},
		{',', false},
		{';', false},
		{'|', false},
		{'\t', false},
		{'\r', false},
		{'\n', false},
		{'a', false},
		{'1', false},
	}
	sniff := NewSniffer(nil, PossibleSeparators([]byte{',', ';', '|', '\t'}), PossibleQuotes([]byte{'"', '\'', '`'}))
	tmpSts := initTempStats(sniff)
	for _, d := range data {
		if tmpSts.isQuoteChar(d.b) != d.r {
			t.Errorf("Expected %v, got %v", d.r, tmpSts.isQuoteChar(d.b))
		}
	}
}

func TestCleanTempStats(t *testing.T) {
	// data, expected result pairs
	data := []struct {
		tempStats *tempStats
		lenSeps   int
		lenQuotes int
		lenPairs  int
	}{
		{&tempStats{
			seps:   map[byte]int{',': 1, ';': 2, '|': 3, '\t': 4},
			quotes: map[byte]int{'"': 1, '\'': 2, '`': 3},
			pairs:  map[sqPair]int{{',', '"'}: 1, {';', '\''}: 2, {'|', '`'}: 3}}, 4, 3, 3},
		{&tempStats{
			seps:   map[byte]int{',': 0, ';': 0, '|': 3, '\t': 4},
			quotes: map[byte]int{'"': 0, '\'': 0, '`': 3},
			pairs:  map[sqPair]int{{',', '"'}: 0, {';', '\''}: 0, {'|', '`'}: 3}}, 2, 1, 3},
	}
	for _, d := range data {
		d.tempStats.cleanTempStats()
		if len(d.tempStats.seps) != d.lenSeps {
			t.Errorf("Expected %d seps, got %d", d.lenSeps, len(d.tempStats.seps))
		}
		if len(d.tempStats.quotes) != d.lenQuotes {
			t.Errorf("Expected %d quotes, got %d", d.lenQuotes, len(d.tempStats.quotes))
		}
		if len(d.tempStats.pairs) != d.lenPairs {
			t.Errorf("Expected %d pairs, got %d", d.lenPairs, len(d.tempStats.pairs))
		}
	}
}
