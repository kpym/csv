package sniffer

import (
	"testing"
)

func TestLenBOM(t *testing.T) {
	tests := []struct {
		data   []byte
		bomlen int
	}{
		{[]byte("a,b,c\nd,e,f"), 0},             // no BOM
		{[]byte("\xEF\xBB\xBFa,b,c\nd,e,f"), 3}, // UTF-8 BOM is skipped
		{[]byte("\xFF\xFEa,b,c\nd,e,f"), 0},     // UTF-16 BOM (LE) is not skipped
		{[]byte("\xFE\xFFa,b,c\nd,e,f"), 0},     // UTF-16 BOM (BE) is not skipped
	}
	for _, test := range tests {
		if got := lenBOM(test.data); got != test.bomlen {
			t.Errorf("BOMLen(%q) = %d, want %d", test.data, got, test.bomlen)
		}
	}
}

func TestPreambleLen(t *testing.T) {
	tests := []struct {
		data   []byte
		prelen int
	}{
		{[]byte("a,b,c\nd,e,f\n"), 0},                    // no preamble
		{[]byte("a,b,c\nd,e,f\n\n\n"), 0},                // no preamble
		{[]byte("a,b,c\nd,e,f\n\t\n \n"), 0},             // no preamble
		{[]byte("a,b,c\n\nd,e,f\n"), 7},                  // preamble and data
		{[]byte(" \t \na,b,c\nd,e,f\n"), 4},              // no preamble
		{[]byte("a,b,c\n\nd,e,f\n\ng"), 14},              // preamble and data
		{[]byte("\xEF\xBB\xBF\na,b,c\nd,e,f\n."), 4},     // with UTF-8 BOM
		{[]byte("\xEF\xBB\xBF \t \na,b,c\nd,e,f\n."), 7}, // with UTF-8 BOM
		{[]byte("\xEF\xBB\xBFa,b,c\n\nd,e,f\n."), 10},    // with UTF-8 BOM
		{[]byte("\xFE\xFF\na,b,c\nd,e,f\n."), 0},         // with UTF-16 BOM (line not empty)
	}
	for _, test := range tests {
		if pre := LenPreamble(test.data); pre != test.prelen {
			t.Errorf("PreambleLen(%q) = %d, want %d", test.data, pre, test.prelen)
		}
	}
}
