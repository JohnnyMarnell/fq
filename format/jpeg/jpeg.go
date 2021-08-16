package jpeg

// https://www.w3.org/Graphics/JPEG/itu-t81.pdf
// TODO: warning on junk before marker?
// TODO: extract photohop to own decoder?
// TODO: image/jpeg

import (
	"bytes"
	"fmt"
	"fq/format"
	"fq/format/registry"
	"fq/pkg/bitio"
	"fq/pkg/decode"
)

var exifFormat []*decode.Format
var iccProfileFormat []*decode.Format

func init() {
	registry.MustRegister(&decode.Format{
		Name:        format.JPEG,
		Description: "Joint Photographic Experts Group file",
		Groups:      []string{format.PROBE, format.IMAGE},
		DecodeFn:    jpegDecode,
		Dependencies: []decode.Dependency{
			{Names: []string{format.EXIF}, Formats: &exifFormat},
			{Names: []string{format.ICC_PROFILE}, Formats: &iccProfileFormat},
		},
	})
}

type marker struct {
	symbol      string
	description string // TODO: use as description later
}

const (
	SOF0  = 0xc0
	SOF1  = 0xc1
	SOF2  = 0xc2
	SOF3  = 0xc3
	SOF5  = 0xc5
	SOF6  = 0xc6
	SOF7  = 0xc7
	JPG   = 0xc8
	SOF9  = 0xc9
	SOF10 = 0xca
	SOF11 = 0xcb
	SOF13 = 0xcd
	SOF14 = 0xce
	SOF15 = 0xcf
	DHT   = 0xc4
	DAC   = 0xcc
	RST0  = 0xd0
	RST1  = 0xd1
	RST2  = 0xd2
	RST3  = 0xd3
	RST4  = 0xd4
	RST5  = 0xd5
	RST6  = 0xd6
	RST7  = 0xd7
	SOI   = 0xd8
	EOI   = 0xd9
	SOS   = 0xda
	DQT   = 0xdb
	DNL   = 0xdc
	DRI   = 0xdd
	DHP   = 0xde
	EXP   = 0xdf
	APP0  = 0xe0
	APP1  = 0xe1
	APP2  = 0xe2
	APP3  = 0xe3
	APP4  = 0xe4
	APP5  = 0xe5
	APP6  = 0xe6
	APP7  = 0xe7
	APP8  = 0xe8
	APP9  = 0xe9
	APP10 = 0xea
	APP11 = 0xeb
	APP12 = 0xec
	APP13 = 0xed
	APP14 = 0xee
	APP15 = 0xef
	JPG0  = 0xf0
	JPG1  = 0xf1
	JPG2  = 0xf2
	JPG3  = 0xf3
	JPG4  = 0xf4
	JPG5  = 0xf5
	JPG6  = 0xf6
	JPG7  = 0xf7
	JPG8  = 0xf8
	JPG9  = 0xf9
	JPG10 = 0xfa
	JPG11 = 0xfb
	JPG12 = 0xfc
	JPG13 = 0xfd
	COM   = 0xfe
	TEM   = 0x01
)

var markers = map[uint]marker{
	SOF0:  {"SOF0", "Baseline DCT"},
	SOF1:  {"SOF1", "Extended sequential DCT"},
	SOF2:  {"SOF2", "Progressive DCT"},
	SOF3:  {"SOF3", "Lossless (sequential)"},
	SOF5:  {"SOF5", "Differential sequential DCT"},
	SOF6:  {"SOF6", "Differential progressive DCT"},
	SOF7:  {"SOF7", "Differential lossless (sequential)"},
	JPG:   {"JPG", "Reserved for JPEG extensions"},
	SOF9:  {"SOF9", "Extended sequential DCT"},
	SOF10: {"SOF10", "Progressive DCT"},
	SOF11: {"SOF11", "Lossless (sequential)"},
	SOF13: {"SOF13", "Differential sequential DCT"},
	SOF14: {"SOF14", "Differential progressive DCT"},
	SOF15: {"SOF15", "Differential lossless (sequential)"},
	DHT:   {"DHT", "Define Huffman table(s)"},
	DAC:   {"DAC", "Define arithmetic coding conditioning(s)"},
	RST0:  {"RST0", "Restart with modulo 8 count 0"},
	RST1:  {"RST1", "Restart with modulo 8 count 1"},
	RST2:  {"RST2", "Restart with modulo 8 count 2"},
	RST3:  {"RST3", "Restart with modulo 8 count 3"},
	RST4:  {"RST4", "Restart with modulo 8 count 4"},
	RST5:  {"RST5", "Restart with modulo 8 count 5"},
	RST6:  {"RST6", "Restart with modulo 8 count 6"},
	RST7:  {"RST7", "Restart with modulo 8 count 7"},
	SOI:   {"SOI", "Start of image"},
	EOI:   {"EOI", "End of image true"},
	SOS:   {"SOS", "Start of scan"},
	DQT:   {"DQT", "Define quantization table(s)"},
	DNL:   {"DNL", "Define number of lines"},
	DRI:   {"DRI", "Define restart interval"},
	DHP:   {"DHP", "Define hierarchical progression"},
	EXP:   {"EXP", "Expand reference component(s)"},
	APP0:  {"APP0", "Reserved for application segments"},
	APP1:  {"APP1", "Reserved for application segments"},
	APP2:  {"APP2", "Reserved for application segments"},
	APP3:  {"APP3", "Reserved for application segments"},
	APP4:  {"APP4", "Reserved for application segments"},
	APP5:  {"APP5", "Reserved for application segments"},
	APP6:  {"APP6", "Reserved for application segments"},
	APP7:  {"APP7", "Reserved for application segments"},
	APP8:  {"APP8", "Reserved for application segments"},
	APP9:  {"APP9", "Reserved for application segments"},
	APP10: {"APP10", "Reserved for application segments"},
	APP11: {"APP11", "Reserved for application segments"},
	APP12: {"APP12", "Reserved for application segments"},
	APP13: {"APP13", "Reserved for application segments"},
	APP14: {"APP14", "Reserved for application segments"},
	APP15: {"APP15", "Reserved for application segments"},
	JPG0:  {"JPG0", "Reserved for JPEG extensions"},
	JPG1:  {"JPG1", "Reserved for JPEG extensions"},
	JPG2:  {"JPG2", "Reserved for JPEG extensions"},
	JPG3:  {"JPG3", "Reserved for JPEG extensions"},
	JPG4:  {"JPG4", "Reserved for JPEG extensions"},
	JPG5:  {"JPG5", "Reserved for JPEG extensions"},
	JPG6:  {"JPG6", "Reserved for JPEG extensions"},
	JPG7:  {"JPG7", "Reserved for JPEG extensions"},
	JPG8:  {"JPG8", "Reserved for JPEG extensions"},
	JPG9:  {"JPG9", "Reserved for JPEG extensions"},
	JPG10: {"JPG10", "Reserved for JPEG extensions"},
	JPG11: {"JPG11", "Reserved for JPEG extensions"},
	JPG12: {"JPG12", "Reserved for JPEG extensions"},
	JPG13: {"JPG13", "Reserved for JPEG extensions"},
	COM:   {"COM", "Comment"},
	TEM:   {"TEM", "For temporary private use in arithmetic coding"},
}

func jpegDecode(d *decode.D, in interface{}) interface{} {
	d.ValidateAtLeastBytesLeft(2)
	if !bytes.Equal(d.PeekBytes(2), []byte{0xff, SOI}) {
		d.Invalid("no SOI marker")
	}

	var extendedXMP []byte
	soiMarkerFound := false
	eoiMarkerFound := false

	d.FieldArrayFn("segments", func(d *decode.D) {
		inECD := false
		for d.NotEnd() && !eoiMarkerFound {
			if inECD {
				ecdLen := int64(0)
				for {
					if d.PeekBits(8) == 0xff && d.PeekBits(16) != 0xff00 {
						break
					}
					d.SeekRel(8)
					ecdLen++
				}
				d.SeekRel(-ecdLen * 8)
				d.FieldBitBufLen("entropy_coded_data", ecdLen*8)
				inECD = false
			} else {
				d.FieldStructFn("marker", func(d *decode.D) {
					prefixLen := d.PeekFindByte(0xff, -1) + 1
					d.FieldBytesLen("prefix", int(prefixLen))
					markerFound := false
					markerCode := d.FieldUDescFn("code", func() (uint64, decode.DisplayFormat, string, string) {
						n := uint(d.U8())
						if m, ok := markers[n]; ok {
							markerFound = true
							return uint64(n), decode.NumberDecimal, m.symbol, m.description
						}
						return uint64(n), decode.NumberDecimal, "RES", "Reset"
					})

					// RST*, SOI, EOI, TEM does not have a length field. All others have a
					// 2 byte length read as "Lf", "Ls" etc or in the default case as "length".

					// TODO: warning on 0x00?
					switch markerCode {
					case SOI:
						soiMarkerFound = true
					case SOF0, SOF1, SOF2, SOF3, SOF5, SOF6, SOF7, SOF9, SOF10, SOF11:
						d.FieldU16("Lf")
						d.FieldU8("P")
						d.FieldU16("Y")
						d.FieldU16("X")
						nf := d.FieldU8("Nf")
						d.FieldArrayFn("frame_components", func(d *decode.D) {
							for i := uint64(0); i < nf; i++ {
								d.FieldStructFn("frame_component", func(d *decode.D) {
									d.FieldU8("C")
									d.FieldU4("H")
									d.FieldU4("V")
									d.FieldU8("Tq")
								})
							}
						})
					case COM:
						comLen := d.FieldU16("Lc")
						d.FieldUTF8("Cm", int(comLen)-2)
					case SOS:
						d.FieldU16("Ls")
						ns := d.FieldU8("Ns")
						d.FieldArrayFn("scan_components", func(d *decode.D) {
							for i := uint64(0); i < ns; i++ {
								d.FieldStructFn("scan_component", func(d *decode.D) {
									d.FieldU8("Cs")
									d.FieldU4("Td")
									d.FieldU4("Ta")
								})
							}
						})
						d.FieldU8("Ss")
						d.FieldU8("Se")
						d.FieldU4("Ah")
						d.FieldU4("Al")
						inECD = true
					case DQT:
						lQ := int64(d.FieldU16("Lq"))
						// TODO: how to extract n? spec says lq is 2 + sum for i in 1 to n 65+64*Pq(i)
						d.DecodeLenFn(lQ*8-16, func(d *decode.D) {
							d.FieldArrayFn("Qs", func(d *decode.D) {
								for d.NotEnd() {
									d.FieldStructFn("Q", func(d *decode.D) {
										pQ := d.FieldU4("Pq")
										qBits := 8
										if pQ != 0 {
											qBits = 16
										}
										d.FieldU4("Tq")
										qK := uint64(0)
										d.FieldArrayLoopFn("Q", func() bool { return qK < 64 }, func(d *decode.D) {
											d.FieldU("Q", qBits)
											qK++
										})
									})
								}
							})
						})
					case RST0, RST1, RST2, RST3, RST4, RST5, RST6, RST7:
						inECD = true
					case TEM:
					case EOI:
						eoiMarkerFound = true
					default:
						if !markerFound {
							d.Invalid(fmt.Sprintf("unknown marker %x", markerCode))
						}

						markerLen := d.FieldU16("length")
						d.DecodeLenFn(int64((markerLen-2)*8), func(d *decode.D) {
							// TODO: map lookup and descriptions?
							app0JFIFPrefix := []byte("JFIF\x00")
							app1ExifPrefix := []byte("Exif\x00\x00")
							extendedXMPPrefix := []byte("http://ns.adobe.com/xmp/extension/\x00")
							app2ICCProfile := []byte("ICC_PROFILE\x00")
							// TODO: other version? generic?
							app13PhotoshopPrefix := []byte("Photoshop 3.0\x00")

							switch {
							case markerCode == APP0 && d.TryHasBytes(app0JFIFPrefix):
								d.FieldUTF8("identifier", len(app0JFIFPrefix))
								d.FieldStructFn("version", func(d *decode.D) {
									d.FieldU8("major")
									d.FieldU8("minor")
								})
								d.FieldU8("density_units")
								d.FieldU16("xdensity")
								d.FieldU16("ydensity")
								xThumbnail := d.FieldU8("xthumbnail")
								yThumbnail := d.FieldU8("ythumbnail")
								d.FieldBitBufLen("data", int64(xThumbnail*yThumbnail)*3*8)
							case markerCode == APP1 && d.TryHasBytes(app1ExifPrefix):
								d.FieldUTF8("exif_prefix", len(app1ExifPrefix))
								d.FieldFormatLen("exif", d.BitsLeft(), exifFormat)
							case markerCode == APP1 && d.TryHasBytes(extendedXMPPrefix):
								d.FieldStructFn("extended_xmp_chunk", func(d *decode.D) {
									d.FieldUTF8("signature", len(extendedXMPPrefix))
									d.FieldUTF8("guid", 32)
									fullLength := d.FieldU32("full_length")
									offset := d.FieldU32("offset")
									// TODO: FieldBitsLen? concat bitbuf?
									chunk := d.FieldBytesLen("data", int(d.BitsLeft()/8))

									if extendedXMP == nil {
										extendedXMP = make([]byte, fullLength)
									}
									copy(extendedXMP[offset:], chunk)
								})
							case markerCode == APP2 && d.TryHasBytes(app2ICCProfile):
								d.FieldUTF8("icc_profile_prefix", len(app2ICCProfile))
								// TODO: support multimarker?
								d.FieldU8("cur_marker")
								d.FieldU8("num_markers")
								d.FieldFormatLen("icc_profile", d.BitsLeft(), iccProfileFormat)
							case markerCode == APP13 && d.TryHasBytes(app13PhotoshopPrefix):
								d.FieldUTF8("identifier", len(app13PhotoshopPrefix))
								signature := d.FieldUTF8("signature", 4)
								switch signature {
								case "8BIM":
									// TODO: description?
									d.FieldStringMapFn("block", psImageResourceBlockNames, "Unknown", d.U16, decode.NumberDecimal)
									d.FieldBitBufLen("data", d.BitsLeft())
								default:
								}
							default:
								// TODO: FieldBitsLen?
								d.FieldBitBufLen("data", d.BitsLeft())
							}
						})
					}
				})
			}
		}
	})

	if !soiMarkerFound {
		d.Invalid("no SOI marker found")
	}

	if extendedXMP != nil {
		bb := bitio.NewBufferFromBytes(extendedXMP, -1)
		// TODO: bit pos, better bitbhuf api?
		d.FieldBitBufFn("extended_xmp", 0, int64(len(extendedXMP))*8, func() (*bitio.Buffer, string) {
			return bb, ""
		})
	}

	return nil
}
