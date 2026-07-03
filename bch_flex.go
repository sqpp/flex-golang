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

func FLEXBCHEncode21(info uint32) uint32 {
	info &= 0x1FFFFF
	poc := BCHEncode31_21(info)
	data := poc >> 10
	parity := poc & 0x3FF
	cw := data | (parity << 21)
	if popCount32(cw)&1 != 0 {
		cw |= 1 << 31
	}
	return reverse32(cw)
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

func FLEXBCHDecode32(cw uint32) (uint32, int) {
	// The assembled FLEX word from the DPLL has the first transmitted bit at LSB.
	// But our BCH decoder expects the first transmitted bit (Even Parity for POCSAG) at LSB.
	// In FLEX, the Even Parity bit is transmitted LAST. So we must reverse the 32-bit word.
	rev := reverse32(cw)
	infoMSB, errs := BCHDecode31_21(rev >> 1)
	
	// The decoded info is 21 bits. Since we reversed the input, the output is reversed.
	// We reverse it back to LSB-first orientation for the rest of the FLEX logic.
	infoLSB := reverse21(infoMSB)
	
	return infoLSB, errs
}

func FLEXChecksum(info uint32) bool {
	if info > 0x3FFFFF {
		return false
	}
	xs := (info & 0x0F) + ((info >> 4) & 0x0F) + ((info >> 8) & 0x0F) + ((info >> 12) & 0x0F) + ((info >> 16) & 0x0F) + ((info >> 20) & 0x01)
	return (xs & 0x0F) == 0x0F
}
