package flex

// frameNumPhases returns how many FLEX phases are active for a baud/level pair.
// Matches decoder.go stData / decodeBlock.
func frameNumPhases(baud, levels int) int {
	n := (baud / 1600) * (levels / 2)
	if n < 1 {
		n = 1
	}
	if n > 4 {
		n = 4
	}
	return n
}

// frameDataSymbols is the number of channel symbols in one FLEX data block.
func frameDataSymbols(baud int) int {
	return baud * 1760 / 1000
}

// codewordsToInterleavedBits expands 88 codewords into the block-interleaved bit order
// used on a single FLEX phase (2816 bits).
func codewordsToInterleavedBits(codewords []uint32) []byte {
	bits := make([]byte, 0, PhaseBits)
	for block := 0; block < 11; block++ {
		base := block * 8
		for bit := 0; bit < 32; bit++ {
			for cwInBlock := 0; cwInBlock < 8; cwInBlock++ {
				cw := base + cwInBlock
				if cw >= len(codewords) {
					bits = append(bits, 0)
					continue
				}
				bits = append(bits, byte((codewords[cw]>>uint(bit))&1))
			}
		}
	}
	return bits
}

func makeIdlePhaseCodewords() []uint32 {
	cws := make([]uint32, flexCodewords)
	for i := range cws {
		cws[i] = idleCodeword(i)
	}
	return cws
}

// makePhaseCodewords builds per-phase codeword arrays; phase A carries the page.
func makePhaseCodewords(primary []uint32, baud, levels int) [][]uint32 {
	numP := frameNumPhases(baud, levels)
	phases := make([][]uint32, numP)
	phases[0] = primary
	for p := 1; p < numP; p++ {
		phases[p] = makeIdlePhaseCodewords()
	}
	return phases
}

func dibitToSymbol(msb, lsb byte) byte {
	return (msb << 1) | lsb
}

// appendMultiphaseSymbols appends the data-block symbol stream (2-level bits or 4-level 0..3).
func appendMultiphaseSymbols(out *[]byte, phaseCodewords [][]uint32, baud, levels int) {
	numP := frameNumPhases(baud, levels)
	symCount := frameDataSymbols(baud)

	phaseBits := make([][]byte, numP)
	for p := 0; p < numP; p++ {
		cw := phaseCodewords[p]
		if len(cw) < flexCodewords {
			padded := makeIdlePhaseCodewords()
			copy(padded, cw)
			cw = padded
		}
		phaseBits[p] = codewordsToInterleavedBits(cw)
	}

	if levels == 2 {
		for sym := 0; sym < symCount; sym++ {
			phase := sym % numP
			bitIdx := sym / numP
			b := byte(0)
			if bitIdx < len(phaseBits[phase]) {
				b = phaseBits[phase][bitIdx]
			}
			*out = append(*out, b)
		}
		return
	}

	pairs := numP / 2
	for sym := 0; sym < symCount; sym++ {
		pair := sym % pairs
		bitIdx := sym / pairs
		msbP := pair * 2
		lsbP := msbP + 1
		msb := byte(0)
		lsb := byte(0)
		if bitIdx < len(phaseBits[msbP]) {
			msb = phaseBits[msbP][bitIdx]
		}
		if bitIdx < len(phaseBits[lsbP]) {
			lsb = phaseBits[lsbP][bitIdx]
		}
		*out = append(*out, dibitToSymbol(msb, lsb))
	}
}
