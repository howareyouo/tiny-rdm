package convutil

import (
	"bytes"
	"io"
)

// compressWith compresses str with the writer returned by newWriter
func compressWith(str string, newWriter func(io.Writer) (io.WriteCloser, error)) (string, bool) {
	var buf bytes.Buffer
	writer, err := newWriter(&buf)
	if err != nil {
		return str, false
	}
	if _, err = writer.Write([]byte(str)); err != nil {
		writer.Close()
		return str, false
	}
	writer.Close()
	return buf.String(), true
}
