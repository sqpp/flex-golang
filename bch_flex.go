package flex

// reverse32 reverses the full 32 bits of x.
func reverse32(x uint32) uint32 {
	var r uint32
	for i := 0; i < 32; i++ {
		r = (r << 1) | (x & 1)
		x >>= 1
	}
	return r
}

// reverse21 reverses the lower 21 bits of x.
func reverse21(x uint32) uint32 {
	var r uint32
	for i := 0; i < 21; i++ {
		r = (r << 1) | (x & 1)
		x >>= 1
	}
	return r
}

// flexEncodeWord builds a FLEX wire codeword matching real OTA captures (test_1600.wav).
func flexEncodeWord(logical21 uint32) uint32 {
	infoMSB := reverse21(logical21 & 0x1FFFFF)
	poc := BCHEncode31_21(infoMSB) & 0x7FFFFFFF
	rev := (poc << 1) | uint32(popCount32(poc)&1)
	return reverse32(rev)
}

func FLEXBCHDecode32(cw uint32) (uint32, int) {
	// Primary path matches real FLEX captures (reverse32 + BCH MSB layout).
	rev := reverse32(cw)
	infoMSB, errs := BCHDecode31_21(rev >> 1)
	if errs >= 0 {
		return reverse21(infoMSB), errs
	}

	// Fallback: bit-0 parity layout.
	code31 := (cw >> 1) & 0x7FFFFFFF
	info := code31 >> 10
	enc := BCHEncode31_21(info) & 0x7FFFFFFF
	if enc == code31 {
		return info & 0x1FFFFF, 0
	}
	return info & 0x1FFFFF, -1
}

func FLEXChecksum(info uint32) bool {
	if info > 0x3FFFFF {
		return false
	}
	xs := (info & 0x0F) + ((info >> 4) & 0x0F) + ((info >> 8) & 0x0F) + ((info >> 12) & 0x0F) + ((info >> 16) & 0x0F) + ((info >> 20) & 0x01)
	return (xs & 0x0F) == 0x0F
}
