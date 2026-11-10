package convutil

import (
	"sort"
	"tinyrdm/backend/types"
	strutil "tinyrdm/backend/utils/string"
)

// DataConvert converts a value between its displayed and its stored form.
type DataConvert interface {
	Encode(string) (string, bool)
	Decode(string) (string, bool)
}

type convertKind uint8

const (
	formatKind convertKind = iota + 1
	decodeKind
)

// builtinConverter describes one built-in formatter or decoder.
// Rank orders auto-detection within its kind, zero disables it.
// Convert is nil when the converter is unavailable on this system.
// Precheck restricts auto-detection to inputs the converter can handle,
// nil means every input may be tried.
type builtinConverter struct {
	Label    string
	Kind     convertKind
	Rank     int
	Convert  DataConvert
	Precheck func(string) bool
}

func (c *builtinConverter) available() bool {
	return c.Convert != nil
}

func (c *builtinConverter) canDetect(str string) bool {
	return c.available() && (c.Precheck == nil || c.Precheck(str))
}

var builtinConverters = []builtinConverter{
	{Label: types.FORMAT_JSON, Kind: formatKind, Rank: 1, Convert: JsonConvert{}},
	{Label: types.FORMAT_UNICODE_JSON, Kind: formatKind, Convert: UnicodeJsonConvert{}},
	{Label: types.FORMAT_YAML, Kind: formatKind, Rank: 2, Convert: YamlConvert{}},
	{Label: types.FORMAT_XML, Kind: formatKind, Rank: 3, Convert: XmlConvert{}},
	{Label: types.FORMAT_HEX, Kind: formatKind, Rank: 4, Convert: HexConvert{}, Precheck: strutil.ContainsBinary},
	{Label: types.FORMAT_BINARY, Kind: formatKind, Convert: BinaryConvert{}},
	{Label: types.FORMAT_BITSET, Kind: formatKind, Convert: BitSetConvert{}},

	{Label: types.DECODE_BASE64, Kind: decodeKind, Rank: 1, Convert: Base64Convert{}, Precheck: canAutoDecodeBase64},
	{Label: types.DECODE_GZIP, Kind: decodeKind, Rank: 2, Convert: GZipConvert{}},
	{Label: types.DECODE_DEFLATE, Kind: decodeKind, Convert: DeflateConvert{}},
	{Label: types.DECODE_ZSTD, Kind: decodeKind, Rank: 3, Convert: ZStdConvert{}},
	{Label: types.DECODE_LZ4, Kind: decodeKind, Rank: 4, Convert: LZ4Convert{}},
	{Label: types.DECODE_BROTLI, Kind: decodeKind, Convert: BrotliConvert{}},
	{Label: types.DECODE_MSGPACK, Kind: decodeKind, Rank: 5, Convert: MsgpackConvert{}},
	{Label: types.DECODE_PHP, Kind: decodeKind, Rank: 6, Convert: NewPhpConvert()},
	{Label: types.DECODE_PICKLE, Kind: decodeKind, Rank: 7, Convert: NewPickleConvert()},
}

var (
	formatters     map[string]*builtinConverter
	decoders       map[string]*builtinConverter
	autoFormatters []*builtinConverter
	autoDecoders   []*builtinConverter
)

func init() {
	formatters = make(map[string]*builtinConverter, len(builtinConverters))
	decoders = make(map[string]*builtinConverter, len(builtinConverters))
	for i := range builtinConverters {
		converter := &builtinConverters[i]
		registry, auto := formatters, &autoFormatters
		if converter.Kind == decodeKind {
			registry, auto = decoders, &autoDecoders
		}
		registry[converter.Label] = converter
		if converter.Rank > 0 {
			*auto = append(*auto, converter)
		}
	}
	sort.SliceStable(autoFormatters, func(i, j int) bool {
		return autoFormatters[i].Rank < autoFormatters[j].Rank
	})
	sort.SliceStable(autoDecoders, func(i, j int) bool {
		return autoDecoders[i].Rank < autoDecoders[j].Rank
	})
}

// AvailableDecoders lists the built-in decoders that can run on this system.
func AvailableDecoders() []string {
	names := make([]string, 0, len(decoders))
	for i := range builtinConverters {
		if converter := &builtinConverters[i]; converter.Kind == decodeKind && converter.available() {
			names = append(names, converter.Label)
		}
	}
	return names
}

func canAutoDecodeBase64(str string) bool {
	return len(str)%4 == 0 && len(str) >= 12 && !strutil.IsSameChar(str)
}