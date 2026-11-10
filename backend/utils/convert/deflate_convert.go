package convutil

import (
	"io"
	"strings"

	"github.com/klauspost/compress/flate"
)

type DeflateConvert struct{}

func (d DeflateConvert) Encode(str string) (string, bool) {
	return compressWith(str, func(w io.Writer) (io.WriteCloser, error) {
		return flate.NewWriter(w, flate.DefaultCompression)
	})
}

func (d DeflateConvert) Decode(str string) (string, bool) {
	reader := flate.NewReader(strings.NewReader(str))
	defer reader.Close()
	if decompressed, err := io.ReadAll(reader); err == nil {
		return string(decompressed), true
	}
	return str, false
}
