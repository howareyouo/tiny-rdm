package convutil

import (
	"bytes"
	"io"

	"github.com/pierrec/lz4/v4"
)

type LZ4Convert struct{}

func (LZ4Convert) Encode(str string) (string, bool) {
	return compressWith(str, func(w io.Writer) (io.WriteCloser, error) {
		return lz4.NewWriter(w), nil
	})
}

func (LZ4Convert) Decode(str string) (string, bool) {
	reader := lz4.NewReader(bytes.NewReader([]byte(str)))
	if decompressed, err := io.ReadAll(reader); err == nil {
		return string(decompressed), true
	}
	return str, false
}
