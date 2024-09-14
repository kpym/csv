// this par of the package contains the main part of the sniffer
// where the stats about (<separator>, <quote>) are collected.
package sniffer

// The bonus are used to attribute a score to a pair of separator and quote character.
// When a character is found a minimal score of 1 is attributed to it.
const (
	// bonus if the data starts with a quote character
	veryFirstQuoteBonus = 8
	// bonus for the first quote character after a separator
	firstQuoteBonus = 4
	// bonus if separator and quote character are next to each other
	besideBonus = 2
	// bonus if separator and quote character are separated only by spaces
	spaceBonus = 1
)

// sqPair is a pair of separator and quote character
// used in the map tempStats.pairs.
type sqPair struct {
	sep   byte
	quote byte
}

// tempStats contains scores for
// separator and quote characters (separately and together).
type tempStats struct {
	// map : possible separator → score
	seps map[byte]int
	// map : possible quote → score
	quotes map[byte]int
	// map : sqPair of separator and quote character → score
	pairs map[sqPair]int
}

// newTempStats collects stats about the data and returns a tempStats.
// newTempStats is used by GuessSepQuoteScore.
func (s *Sniffer) newTempStats() *tempStats {
	// create a new tempStats
	t := initTempStats(s)
	// collect stats
	t.collectTempStats(s.data)
	// clean maps
	t.cleanTempStats()
	return t
}

// initTempStats returns a new zeroed tempStats.
func initTempStats(s *Sniffer) *tempStats {
	// create a new tempStats
	t := &tempStats{
		seps:   make(map[byte]int, len(s.seps)),
		quotes: make(map[byte]int, len(s.quotes)),
		pairs:  make(map[sqPair]int),
	}
	// init maps
	for _, sep := range s.seps {
		t.seps[sep] = 0
	}
	for _, quote := range s.quotes {
		t.quotes[quote] = 0
	}

	return t
}

// collectTempStats loops over the data and when it meets a separator or a quote character
// it attributes a score based on the previous characters using the bonus constants.
// A score is attributed to the single characters and eventually to the pair of separator and quote character.
// The scores are added to the maps Sniffer.seps, Sniffer.quotes and Sniffer.pairs.
func (t *tempStats) collectTempStats(data []byte) {
	if len(data) == 0 {
		return
	}

	// newline is a utility constant
	const newline = byte('\n')

	// useed to attribute the firstBonus
	isFirstSep := true
	isFirstQuote := true

	// used to attribute the noSpaceBonus
	prevChar := newline   // we are at the beginning of a new line
	prevCharIsSep := true // newline is a separator
	prevCharIsQuote := false

	// used to attribute the spaceBonus
	prevNonSpace := newline   // we are at the beginning of a new line
	prevNonSpaceIsSep := true // newline is a separator
	prevNonSpaceIsQuote := false

	// check if the very first character is a quote character
	if c := data[0]; t.isQuoteChar(c) {
		t.quotes[c] += veryFirstQuoteBonus
	}
	// loop over the data byte by byte, scanning for separators and quote characters
	for _, c := range data {
		// if data[i] is a quote character
		if t.isQuoteChar(c) {
			// if this is the first quote character (after separator) in the data
			if isFirstQuote && (prevCharIsSep || prevNonSpaceIsSep) {
				t.quotes[c] += firstQuoteBonus
				isFirstQuote = false
			}
			// if the previous character is a separator
			if prevCharIsSep {
				// append
				t.quotes[c] += besideBonus
				if prevChar != newline {
					t.pairs[sqPair{prevChar, c}] += besideBonus
				}
			}
			// if the previous non-space character is a separator
			if prevNonSpaceIsSep {
				t.quotes[c] += spaceBonus
				if prevNonSpace != newline {
					t.pairs[sqPair{prevNonSpace, c}] += spaceBonus
				}
			}
			prevChar = c
			prevCharIsSep = false
			prevCharIsQuote = true
			prevNonSpace = c
			prevNonSpaceIsSep = false
			prevNonSpaceIsQuote = true
			continue
		}

		// if data[i] is a separator
		if t.isSepChar(c) {
			t.seps[c]++
			// if this is the first separator in the data
			if isFirstSep {
				t.seps[c] += firstQuoteBonus
				isFirstSep = false
			}
			// if the previous character is a quote character
			if prevCharIsQuote {
				// append
				t.seps[c] += besideBonus
				t.pairs[sqPair{c, prevChar}] += besideBonus
			}
			// if the previous non-space character is a quote character
			if prevNonSpaceIsQuote {
				t.seps[c] += spaceBonus
				t.pairs[sqPair{c, prevNonSpace}] += spaceBonus
			}
			prevChar = c
			prevCharIsSep = true
			prevCharIsQuote = false
			prevNonSpace = c
			prevNonSpaceIsSep = true
			prevNonSpaceIsQuote = false
			continue
		}
		// neither a separator nor a quote character
		prevChar = c
		prevCharIsSep = c == newline // newline is a separator
		prevCharIsQuote = false
		if c != ' ' {
			prevNonSpace = c
			prevNonSpaceIsSep = c == newline // newline is a separator
			prevNonSpaceIsQuote = false
		}
	}
}

// isSepChar returns true if c is a separator character.
// It is used by stats()
func (t *tempStats) isSepChar(c byte) bool {
	_, ok := t.seps[c]
	return ok
}

// isQuoteChar returns true if c is a quote character.
// It is used by stats()
func (t *tempStats) isQuoteChar(c byte) bool {
	_, ok := t.quotes[c]
	return ok
}

// cleanTempStats removes from the maps Sniffer.seps and Sniffer.quotes the characters with no score.
// It should be called after Sniffer.stats.
// As a result t.seps and t.quotes could be empty.
// Remark : in go it is safe to delete entries in a range loop.
func (t *tempStats) cleanTempStats() {
	// remove quotes with no score
	for quote := range t.quotes {
		if t.quotes[quote] == 0 {
			delete(t.quotes, quote)
		}
	}
	// remove separators with no score
	for sep := range t.seps {
		if t.seps[sep] == 0 {
			delete(t.seps, sep)
		}
	}
}
