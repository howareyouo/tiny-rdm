package convutil

import (
	"io"
	"strings"

	"github.com/andybalholm/brotli"
)

type BrotliConvert struct{}

func (BrotliConvert) Encode(str string) (string, bool) {
	return compressWith(str, func(w io.Writer) (io.WriteCloser, error) {
		return brotli.NewWriter(w), nil
	})
}

func (BrotliConvert) Decode(str string) (string, bool) {
	reader := brotli.NewReader(strings.NewReader(str))
	if decompressed, err := io.ReadAll(reader); err == nil {
		return string(decompressed), true
	}
	return str, false
}
