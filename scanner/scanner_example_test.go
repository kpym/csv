package scanner_test

import (
	"fmt"
	"strings"

	"github.com/kpym/csv/scanner"
)

func ExampleScanner() {
	// Create a scanner
	csv := `# This, is a comment
 foo ,  bar  ,  baz, # This is not a comment

", foo ",  "b'ar,"," b"",az " ,
`

	scanner := scanner.New(strings.NewReader(csv)) // scanner.WithSeparator(','), // default
	// scanner.WithQuote('"', scanner.QuoteCollectorFuzzy),
	// scanner.WithEscape('"'), // default
	// scanner.WithComment([]byte("#")), // default
	// scanner.Without(scanner.RemoveQuotes),

	// Scan the input
	for scanner.Scan() {
		field := scanner.Bytes()
		if scanner.IsComment() {
			// This is a comment
			fmt.Printf("💬 Comment: <%s>\n", field)
			continue
		}
		if scanner.IsEmptyLine() {
			// This is a comment
			fmt.Println("∅ (empty line)")
			continue
		}
		if scanner.AtRowStart() {
			fmt.Println("┎ Row start")
		}
		if scanner.IsQuoted() {
			fmt.Printf("┠ Enquoted: <%s>\n", field)
		} else {
			fmt.Printf("┠ Not enquoted: <%s>\n", field)
		}
		if scanner.AtRowEnd() {
			fmt.Println("┖ Row end")
		}
	}
	// Check for errors
	if err := scanner.Err(); err != nil {
		fmt.Println("Error:", err)
	}
	// Output:
	// 💬 Comment: < This, is a comment>
	// ┎ Row start
	// ┠ Not enquoted: < foo >
	// ┠ Not enquoted: <  bar  >
	// ┠ Not enquoted: <  baz>
	// ┠ Not enquoted: < # This is not a comment>
	// ┖ Row end
	// ∅ (empty line)
	// ┎ Row start
	// ┠ Enquoted: <, foo >
	// ┠ Enquoted: <b'ar,>
	// ┠ Enquoted: < b",az >
	// ┠ Not enquoted: <>
	// ┖ Row end
}
