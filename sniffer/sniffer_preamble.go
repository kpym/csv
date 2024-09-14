package sniffer

// lenBOM returns 3 if the data starts with a UTF-8 BOM, 0 otherwise.
func lenBOM(data []byte) int {
	// skip BOM if present
	if len(data) >= 3 && (data)[0] == 0xEF && (data)[1] == 0xBB && (data)[2] == 0xBF {
		return 3 // skip UTF-8 BOM
	}
	return 0
}

// LenPreamble return the estimated length of the preamble in bytes.
// This is a very simple method that returns the index of the last empty line
// that is followed by a non-empty line.
// A line is considered as empty if it has only white spaces (' ' or '\t').
// If UTF-8 BOM is present, it is considered as part of the preamble.
func LenPreamble(data []byte) int {
	var i, bom int
	// skip BOM if present
	if bom = lenBOM(data); bom > 0 {
		data = data[bom:]
	}
	// skip the ending white spaces
	for i = len(data) - 1; i >= 0; i-- {
		if data[i] != '\n' && data[i] != '\r' && data[i] != ' ' && data[i] != '\t' {
			break
		}
	}
	inEmptyLine := false
	l := i + 1
	for ; i >= 0; i-- {
		if data[i] == '\n' {
			if inEmptyLine {
				return l + bom
			}
			inEmptyLine = true
			l = i + 1 // include the newline
		} else if data[i] != ' ' && data[i] != '\t' {
			inEmptyLine = false
		}
	}
	if inEmptyLine {
		return l + bom
	} else {
		return bom
	}
}
