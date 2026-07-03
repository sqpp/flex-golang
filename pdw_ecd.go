package flex

// PDW FLEX codeword decode path (Flex.cpp showblock + Misc.cpp ecd).

var (
	pdwECS [21]int
	pdwBCH [1024]int
)

func init() {
	setupPDWECC()
}

func setupPDWECC() {
	srr := 0x3B4
	for i := 0; i <= 20; i++ {
		pdwECS[i] = srr
		if srr&0x01 != 0 {
			srr = (srr >> 1) ^ 0x3B4
		} else {
			srr >>= 1
		}
	}

	for i := range pdwBCH {
		pdwBCH[i] = 0
	}

	for n := 0; n <= 20; n++ {
		for i := 0; i <= 20; i++ {
			j := (i << 5) + n
			k := pdwECS[n] ^ pdwECS[i]
			pdwBCH[k] = j + 0x2000
		}
	}

	for n := 0; n <= 20; n++ {
		k := pdwECS[n]
		j := n + (0x1f << 5)
		pdwBCH[k] = j + 0x1000
	}

	for n := 0; n <= 20; n++ {
		for i := 0; i < 10; i++ {
			k := pdwECS[n] ^ (1 << i)
			j := n + (0x1f << 5)
			pdwBCH[k] = j + 0x2000
		}
	}

	for n := 0; n < 10; n++ {
		pdwBCH[1<<n] = 0x3ff + 0x1000
	}

	for n := 0; n < 10; n++ {
		for i := 0; i < 10; i++ {
			if i != n {
				k := (1 << n) ^ (1 << i)
				pdwBCH[k] = 0x3ff + 0x2000
			}
		}
	}
}

func pdwBit10(gin int) int {
	k := 0
	for i := 0; i < 10; i++ {
		if gin&0x01 != 0 {
			k++
		}
		gin >>= 1
	}
	return k
}

func pdwECD(ob *[32]byte) int {
	ecc := 0
	parity := 0

	for i := 0; i <= 20; i++ {
		if ob[i] == 1 {
			ecc ^= pdwECS[i]
			parity ^= 0x01
		}
	}

	acc := 0
	for i := 21; i <= 30; i++ {
		acc <<= 1
		if ob[i] == 1 {
			acc ^= 0x01
		}
	}

	synd := ecc ^ acc
	errors := 0

	if synd != 0 {
		entry := pdwBCH[synd&0x3FF]
		if entry != 0 {
			b1 := entry & 0x1f
			b2 := (entry >> 5) & 0x1f

			if b2 != 0x1f {
				ob[b2] ^= 1
				ecc ^= pdwECS[b2]
			}
			if b1 != 0x1f {
				ob[b1] ^= 1
				ecc ^= pdwECS[b1]
			}
			errors = entry >> 12
		} else {
			errors = 3
		}

		if errors == 1 {
			parity ^= 0x01
		}
	}

	parity = (parity + pdwBit10(ecc)) & 0x01
	if parity != int(ob[31]) {
		errors++
	}
	if errors > 3 {
		errors = 3
	}
	return errors
}

func wireToOB(wire uint32) [32]byte {
	var ob [32]byte
	for i := 0; i < 32; i++ {
		bit := (wire >> uint(i)) & 1
		ob[i] = byte(bit ^ 1) // PDW ob[] is active-low vs demod wire bits
	}
	return ob
}

// PDWDecodeWire decodes a demodulated 32-bit wire word the way PDW does.
func PDWDecodeWire(wire uint32) (logical int64, errLevel int) {
	ob := wireToOB(wire)
	errLevel = pdwECD(&ob)

	var cc int64
	for j := 0; j < 21; j++ {
		cc >>= 1
		if ob[j] == 0 {
			cc ^= 0x100000
		}
	}
	if errLevel == 3 {
		cc ^= 0x400000
	}
	return cc, errLevel
}

func pdwXsumchk(l int64) bool {
	if l > 0x3FFFFF {
		return false
	}
	xs := int(l & 0x0F)
	xs += int((l >> 4) & 0x0F)
	xs += int((l >> 8) & 0x0F)
	xs += int((l >> 12) & 0x0F)
	xs += int((l >> 16) & 0x0F)
	xs += int((l >> 20) & 0x01)
	return (xs & 0x0F) == 0x0F
}
