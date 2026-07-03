package flex

// testexport.go exposes internal symbols for white-box tests in ./tests.
// These are not part of the stable public API.

func ExportBuildFIWData(cycle, frame int) uint32 { return buildFIWData(cycle, frame) }

func ExportEncodeWord(data21 uint32) uint32 { return encodeWord(data21) }

func ExportBuildBIW(voffset, aoffset int) uint32 { return buildBIW(voffset, aoffset) }

func ExportFlexEncodeWord(logical21 uint32) uint32 { return flexEncodeWord(logical21) }

func ExportIdleCodeword(i int) uint32 { return idleCodeword(i) }

func ExportFlexCodewordCount() int { return flexCodewords }

func ExportBitstreamFromCodewords(codewords []uint32, modeName string, cycle, frame int) ([]byte, error) {
	mode, err := getEncodeMode(modeName)
	if err != nil {
		return nil, err
	}
	tx, err := bitstreamFromCodewords(codewords, mode, cycle, frame)
	if err != nil {
		return nil, err
	}
	return modulateTransmission(tx), nil
}

func ExportBitstreamWithFIW(fiw uint32, fiwMSB bool) ([]byte, error) {
	codewords, mode, err := assembleCodewords(EncodeMessage{
		Capcode: 1913, Type: "alpha", Text: "HELLO WORLD",
	}, Mode1600_2, 0, 0)
	if err != nil {
		return nil, err
	}
	tx := buildTransmission(codewords, mode, 0, 0)
	const fiwOffset = 1040
	rebuilt := append([]byte(nil), tx.header[:fiwOffset]...)
	if fiwMSB {
		appendBitsMSB(&rebuilt, fiw, 32)
	} else {
		appendBitsLSB(&rebuilt, fiw, 32)
	}
	rebuilt = append(rebuilt, tx.header[fiwOffset+32:]...)
	tx.header = rebuilt
	return modulateTransmission(tx), nil
}

func ExportEncodeFIWDirect(logical21 uint32) uint32 {
	poc := BCHEncode31_21(logical21 & 0x1FFFFF)
	cw := (poc >> 10) | ((poc & 0x3FF) << 21)
	if popCount32(cw)&1 != 0 {
		cw |= 1 << 31
	}
	return cw
}

func ExportCheckSync1(buf uint64) (uint16, bool) { return checkSync1(buf) }

func ExportEncodeAlphaPayload(text string, maxWords int, skipFirstChar bool) []uint32 {
	return encodeAlphaPayload(text, maxWords, skipFirstChar)
}

func ExportAlphaSignature(msgWords []uint32) uint32 { return alphaSignature(msgWords) }

func ExportAlphaHeaderChecksum(header uint32, content []uint32) uint32 {
	return alphaHeaderChecksum(header, content)
}

func ExportPDWXsumchk(l int64) bool { return pdwXsumchk(l) }

func ExportReverse21(v uint32) uint32 { return reverse21(v) }

func ExportBCHEncode31_21(v uint32) uint32 { return BCHEncode31_21(v) }

func ExportPopCount32(v uint32) int { return popCount32(v) }

func ExportReverse32(v uint32) uint32 { return reverse32(v) }

func ExportFLEXChecksum(info uint32) bool { return FLEXChecksum(info) }

func ExportBuildBitstreamHeader(msg EncodeMessage, modeName string, cycle, frame int) ([]byte, error) {
	tx, err := BuildBitstream(msg, modeName, cycle, frame)
	if err != nil {
		return nil, err
	}
	return tx.header, nil
}
