package convutil

import (
	"io"
	"strings"

	"github.com/klauspost/compress/gzip"
)

type GZipConvert struct{}

func (GZipConvert) Encode(str string) (string, bool) {
	return compressWith(str, func(w io.Writer) (io.WriteCloser, error) {
		return gzip.NewWriter(w), nil
	})
}

func (GZipConvert) Decode(str string) (string, bool) {
	if reader, err := gzip.NewReader(strings.NewReader(str)); err == nil {
		defer reader.Close()
		var decompressed []byte
		if decompressed, err = io.ReadAll(reader); err == nil {
			return string(decompressed), true
		}
	}
	return str, false
}
