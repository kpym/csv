package writer_test

import (
	"fmt"
	"os"

	"github.com/kpym/csv/writer"
)

func ExampleWriter() {
	// Create a writer
	w := writer.New(os.Stdout, writer.WithSeparator(';'))

	// Write the data
	data := [][]string{
		{"a", "b", "c"},
		{},
		{"d\ne", "f\"g", "h;i"},
	}
	w.WriteStringComment("This is a coment line")
	for _, record := range data {
		if len(record) == 0 {
			w.EmptyRow()
			continue
		}
		for _, field := range record {
			w.WriteStringField(field)
		}
	}
	// Flush the writer
	w.Flush()
	if w.Error() != nil {
		fmt.Println("Error:", w.Error())
		return
	}
	// Output:
	// # This is a coment line
	// a;b;c
	//
	// "d
	// e";"f""g";"h;i"
}
