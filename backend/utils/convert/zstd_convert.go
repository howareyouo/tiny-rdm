package convutil

import (
	"io"
	"strings"

	"github.com/klauspost/compress/zstd"
)

type ZStdConvert struct{}

func (ZStdConvert) Encode(str string) (string, bool) {
	return compressWith(str, func(w io.Writer) (io.WriteCloser, error) {
		return zstd.NewWriter(w)
	})
}

func (ZStdConvert) Decode(str string) (string, bool) {
	if reader, err := zstd.NewReader(strings.NewReader(str)); err == nil {
		defer reader.Close()
		if decompressed, err := io.ReadAll(reader); err == nil {
			return string(decompressed), true
		}
	}
	return str, false
}
