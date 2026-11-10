package convutil

import (
	"errors"
	"regexp"
	"tinyrdm/backend/types"
)

// ConvertTo convert string to specified type
// @param decodeType empty string indicates automatic detection
// @param formatType empty string indicates automatic detection
// @param custom decoder if any
func ConvertTo(str, decodeType, formatType string, customDecoder []CmdConvert) (value, resultDecode, resultFormat string) {
	if len(str) <= 0 {
		// empty content
		if len(formatType) <= 0 {
			resultFormat = types.FORMAT_RAW
		} else {
			resultFormat = formatType
		}
		if len(decodeType) <= 0 {
			resultDecode = types.DECODE_NONE
		} else {
			resultDecode = decodeType
		}
		return
	}

	// decode first
	value, resultDecode = decodeWith(str, decodeType, customDecoder)
	// then format content
	if len(formatType) <= 0 {
		value, resultFormat = autoViewAs(value)
	} else {
		value, resultFormat = viewAs(value, formatType)
	}
	return
}

func decodeWith(str, decodeType string, customDecoder []CmdConvert) (value, resultDecode string) {
	if len(decodeType) > 0 {
		value = str

		if decoder, ok := decoders[decodeType]; ok {
			if decoder.available() {
				if decodedStr, ok := decoder.Convert.Decode(str); ok {
					value = decodedStr
				}
			}
		} else if decodeType != types.DECODE_NONE {
			for _, decoder := range customDecoder {
				if decoder.Name == decodeType {
					if decodedStr, ok := decoder.Decode(str); ok {
						value = decodedStr
					}
					break
				}
			}
		}

		resultDecode = decodeType
		return
	}

	value, resultDecode = autoDecode(str, customDecoder)
	return
}

// attempt try possible decode method
// if no decode is possible, it will return the origin string value and "none" decode type
func autoDecode(str string, customDecoder []CmdConvert) (value, resultDecode string) {
	if len(str) > 0 {
		// pure digit content may incorrect regard as some encoded type, skip decode
		if match, _ := regexp.MatchString(`^\d+$`, str); !match {
			for _, converter := range autoDecoders {
				// FIXME: deflate and brotli keep disabled for auto detection
				// due to incorrect format checking
				if !converter.canDetect(str) {
					continue
				}
				if decodedStr, ok := converter.Convert.Decode(str); ok {
					return decodedStr, converter.Label
				}
			}

			// try decode with custom decoder
			for _, decoder := range customDecoder {
				if decoder.Auto {
					if decodedStr, ok := decoder.Decode(str); ok {
						return decodedStr, decoder.Name
					}
				}
			}
		}
	}

	return str, types.DECODE_NONE
}

func viewAs(str, formatType string) (value, resultFormat string) {
	if len(formatType) > 0 {
		value = str
		if formatter, ok := formatters[formatType]; ok {
			if formattedStr, ok := formatter.Convert.Decode(str); ok {
				value = formattedStr
			}
		}
		resultFormat = formatType
		return
	}
	return
}

// attempt automatic convert to possible types
// if no conversion is possible, it will return the origin string value and "plain text" type
func autoViewAs(str string) (value, resultFormat string) {
	if len(str) > 0 {
		for _, converter := range autoFormatters {
			if !converter.canDetect(str) {
				continue
			}
			if formattedStr, ok := converter.Convert.Decode(str); ok {
				return formattedStr, converter.Label
			}
		}
	}

	return str, types.FORMAT_RAW
}

// SaveAs convert the displayed value back to the stored one, the format is
// applied first and the decoder encodes the formatted content.
func SaveAs(str, format, decode string, customDecoder []CmdConvert) (value string, err error) {
	value = str
	if formatter, ok := formatters[format]; ok {
		if formattedStr, ok := formatter.Convert.Encode(value); ok {
			value = formattedStr
		} else {
			err = errors.New("invalid " + format + " data")
			return
		}
	}

	if decoder, ok := decoders[decode]; ok {
		if !decoder.available() {
			err = errors.New("fail to build " + decode)
			return
		}
		if encodedValue, ok := decoder.Convert.Encode(value); ok {
			value = encodedValue
		} else {
			err = errors.New("fail to build " + decode)
		}
		return
	} else if decode != types.DECODE_NONE {
		for _, decoder := range customDecoder {
			if decoder.Name == decode {
				if encodedStr, ok := decoder.Encode(value); ok {
					value = encodedStr
				} else {
					err = errors.New("fail to build " + decode)
				}
				return
			}
		}
	}
	return
}