package flex

import (
	"fmt"
	"os"
)

const (
	Mode1600_2 = "1600/2"
	Mode3200_2 = "3200/2"
	Mode3200_4 = "3200/4"
	Mode6400_4 = "6400/4"

	flexSync1600_2 = 0x870C
	flexCodewords  = PhaseWords // 88
)

// EncodeMessage is one pager message to encode into a FLEX frame.
type EncodeMessage struct {
	Capcode int
	Type    string // "alpha", "numeric", "tone"
	Text    string
}

type encodeMode struct {
	name     string
	syncCode uint16
	baud     int
	levels   int
}

var encodeModes = map[string]encodeMode{
	Mode1600_2: {Mode1600_2, flexSync1600_2, 1600, 2},
}

// EncodeModeNames returns modes supported by the encoder.
func EncodeModeNames() []string {
	return []string{Mode1600_2}
}

func getEncodeMode(name string) (encodeMode, error) {
	m, ok := encodeModes[name]
	if !ok {
		return encodeMode{}, fmt.Errorf("encoder mode %q not supported yet (available: %v)", name, EncodeModeNames())
	}
	return m, nil
}

func flexChecksumSum(info uint32) uint32 {
	return (info & 0x0F) + ((info >> 4) & 0x0F) + ((info >> 8) & 0x0F) +
		((info >> 12) & 0x0F) + ((info >> 16) & 0x0F) + ((info >> 20) & 0x1)
}

func flexSetChecksum(info uint32) uint32 {
	info &^= 0x0F
	info |= (0xF - flexChecksumSum(info)) & 0x0F
	return info
}

func encodeWord(data21 uint32) uint32 {
	return flexEncodeWord(data21)
}

func buildFIWData(cycle, frame int) uint32 {
	fiw := uint32((cycle&0xF)<<4) | uint32((frame&0x7F)<<8)
	return flexSetChecksum(fiw)
}

func buildBIW(voffset, aoffset int) uint32 {
	biw := uint32((aoffset&0x3)<<8) | uint32((voffset&0x3F)<<10)
	return encodeWord(flexSetChecksum(biw))
}

func buildAddressWord(capcode int) uint32 {
	return encodeWord(uint32((capcode + 0x8000) & 0x1FFFFF))
}

func buildVectorWord(msgType, msgStart, msgLen int) uint32 {
	vec := uint32((msgType&0x7)<<4) | uint32((msgStart&0x7F)<<7) | uint32((msgLen&0x7F)<<14)
	return encodeWord(flexSetChecksum(vec))
}

func buildMessageWord(data21 uint32) uint32 {
	return encodeWord(data21 & 0x1FFFFF)
}

func idleCodeword(i int) uint32 {
	if i%2 == 0 {
		return encodeWord(0x0AAAAA)
	}
	return encodeWord(0x155555)
}

// encodeAlphaPayload packs 7-bit ASCII into 21-bit message words.
// skipFirstChar leaves bits 0-6 empty in the first word for the signature slot.
func encodeAlphaPayload(text string, maxWords int, skipFirstChar bool) []uint32 {
	bitPos := 0
	if skipFirstChar {
		bitPos = 7
	}

	var words []uint32
	var current uint32

	for i := 0; i < len(text) && len(words) < maxWords; i++ {
		ch := uint32(text[i]) & 0x7F
		current |= ch << bitPos
		bitPos += 7

		if bitPos >= 21 {
			words = append(words, current&0x1FFFFF)
			overflow := bitPos - 21
			if overflow > 0 {
				current = ch >> (7 - overflow)
				bitPos = overflow
			} else {
				current = 0
				bitPos = 0
			}
		}
	}

	if bitPos > 0 && len(words) < maxWords {
		words = append(words, current&0x1FFFFF)
	}

	// ETX padding (gr-mixalot / FLEX spec).
	if bitPos == 7 && len(words) > 0 {
		words[len(words)-1] |= (0x03 << 7) | (0x03 << 14)
	} else if bitPos == 14 && len(words) > 0 {
		words[len(words)-1] |= 0x03 << 14
	}

	return words
}

func alphaSignature(msgWords []uint32) uint32 {
	var sigSum uint32
	if len(msgWords) > 0 {
		for _, slot := range []int{7, 14} {
			ch := (msgWords[0] >> slot) & 0x7F
			if ch != 0x03 {
				sigSum += ch
			}
		}
		for i := 1; i < len(msgWords); i++ {
			for _, slot := range []int{0, 7, 14} {
				ch := (msgWords[i] >> slot) & 0x7F
				if ch != 0x03 {
					sigSum += ch
				}
			}
		}
	}
	return (^sigSum) & 0x7F
}

func alphaHeaderChecksum(header uint32, content []uint32) uint32 {
	kSum := (header & 0xFF) + ((header >> 8) & 0xFF) + ((header >> 16) & 0x1F)
	for _, w := range content {
		kSum += (w & 0xFF) + ((w >> 8) & 0xFF) + ((w >> 16) & 0x1F)
	}
	return (^kSum) & 0x3FF
}

func buildAlphaCodewords(text string, codewords []uint32) error {
	const maxContent = 84
	content := encodeAlphaPayload(text, maxContent, true)
	if len(content) == 0 && text != "" {
		content = []uint32{0}
	}
	content[0] |= alphaSignature(content)

	const (
		voffset  = 2
		aoffset  = 0
		msgStart = 3
	)
	totalMsgWords := len(content) + 1

	codewords[0] = buildBIW(voffset, aoffset)
	codewords[2] = buildVectorWord(int(PageAlphanumeric), msgStart, totalMsgWords)

	header := uint32(0x1800)
	header |= alphaHeaderChecksum(header, content)
	codewords[msgStart] = buildMessageWord(header)

	for i, w := range content {
		idx := msgStart + 1 + i
		if idx >= flexCodewords {
			break
		}
		codewords[idx] = buildMessageWord(w)
	}
	return nil
}

func buildToneCodewords(codewords []uint32) {
	const (
		voffset  = 2
		aoffset  = 0
		msgStart = 3
	)
	codewords[0] = buildBIW(voffset, aoffset)
	codewords[2] = buildVectorWord(int(PageTone), msgStart, 1)
	codewords[msgStart] = buildMessageWord(0)
}

func buildSync1(syncCode uint16) uint64 {
	complement := syncCode ^ 0xFFFF
	return (uint64(syncCode) << 48) | (uint64(SyncMarker) << 16) | uint64(complement)
}

func appendBitsMSBInv(bits *[]byte, value uint64, n int) {
	for i := n - 1; i >= 0; i-- {
		b := byte((value >> uint(i)) & 1)
		*bits = append(*bits, b^1)
	}
}

func appendBitsLSB(bits *[]byte, value uint32, n int) {
	for i := 0; i < n; i++ {
		*bits = append(*bits, byte((value>>uint(i))&1))
	}
}

func appendInterleavedData(bits *[]byte, codewords []uint32) {
	for block := 0; block < 11; block++ {
		base := block * 8
		for bit := 0; bit < 32; bit++ {
			for cwInBlock := 0; cwInBlock < 8; cwInBlock++ {
				cw := base + cwInBlock
				if cw >= len(codewords) {
					continue
				}
				*bits = append(*bits, byte((codewords[cw]>>uint(bit))&1))
			}
		}
	}
}

// BuildBitstream assembles a complete FLEX 1600/2 bitstream.
func BuildBitstream(msg EncodeMessage, modeName string, cycle, frame int) ([]byte, error) {
	codewords, mode, err := assembleCodewords(msg, modeName, cycle, frame)
	if err != nil {
		return nil, err
	}
	return bitstreamFromCodewords(codewords, mode, cycle, frame)
}

func assembleCodewords(msg EncodeMessage, modeName string, cycle, frame int) ([]uint32, encodeMode, error) {
	mode, err := getEncodeMode(modeName)
	if err != nil {
		return nil, encodeMode{}, err
	}
	if msg.Capcode <= 0 {
		return nil, encodeMode{}, fmt.Errorf("capcode must be positive")
	}

	codewords := make([]uint32, flexCodewords)
	for i := range codewords {
		codewords[i] = idleCodeword(i)
	}

	switch msg.Type {
	case "alpha", "":
		if err := buildAlphaCodewords(msg.Text, codewords); err != nil {
			return nil, encodeMode{}, err
		}
	case "tone":
		buildToneCodewords(codewords)
	default:
		return nil, encodeMode{}, fmt.Errorf("message type %q not supported yet", msg.Type)
	}
	codewords[1] = buildAddressWord(msg.Capcode)
	return codewords, mode, nil
}

func bitstreamFromCodewords(codewords []uint32, mode encodeMode, cycle, frame int) ([]byte, error) {
	var bits []byte

	for i := 0; i < 960; i++ {
		if i&1 == 0 {
			bits = append(bits, 0)
		} else {
			bits = append(bits, 1)
		}
	}

	appendBitsMSBInv(&bits, buildSync1(mode.syncCode), 64)

	for i := 0; i < 16; i++ {
		if i&1 == 0 {
			bits = append(bits, 0)
		} else {
			bits = append(bits, 1)
		}
	}

	fiw := encodeWord(buildFIWData(cycle, frame))
	appendBitsLSB(&bits, fiw, 32)

	for i := 0; i < 4; i++ {
		if i&1 == 0 {
			bits = append(bits, 0)
		} else {
			bits = append(bits, 1)
		}
	}
	const cPat = uint16(0xED84)
	for i := 15; i >= 0; i-- {
		bits = append(bits, byte((cPat>>uint(i))&1))
	}
	for i := 0; i < 4; i++ {
		if i&1 == 0 {
			bits = append(bits, 1)
		} else {
			bits = append(bits, 0)
		}
	}
	const cInvPat = uint16(0x127B)
	for i := 15; i >= 0; i-- {
		bits = append(bits, byte((cInvPat>>uint(i))&1))
	}

	appendInterleavedData(&bits, codewords)

	for i := 0; i < 64; i++ {
		if i&1 == 0 {
			bits = append(bits, 0)
		} else {
			bits = append(bits, 1)
		}
	}

	return bits, nil
}

// EncodeToWAVBytes encodes messages into a FLEX WAV file.
func EncodeToWAVBytes(messages []EncodeMessage, modeName string, cycle, frame int) ([]byte, int, int, error) {
	if len(messages) == 0 {
		return nil, 0, 0, fmt.Errorf("no messages to encode")
	}
	mode, err := getEncodeMode(modeName)
	if err != nil {
		return nil, 0, 0, err
	}

	bits, err := BuildBitstream(messages[0], modeName, cycle, frame)
	if err != nil {
		return nil, 0, 0, err
	}

	wav := modulateBits(bits, mode.baud)
	numSamples := 0
	if len(wav) >= 44 {
		dataSize := int(wav[40]) | int(wav[41])<<8 | int(wav[42])<<16 | int(wav[43])<<24
		numSamples = dataSize / 2
	}
	return wav, len(bits), numSamples, nil
}

// EncodeToWAVFile writes a FLEX WAV file.
func EncodeToWAVFile(messages []EncodeMessage, outPath, modeName string, cycle, frame int) (int, int, error) {
	wav, nBits, nSamples, err := EncodeToWAVBytes(messages, modeName, cycle, frame)
	if err != nil {
		return 0, 0, err
	}
	if err := os.WriteFile(outPath, wav, 0644); err != nil {
		return 0, 0, err
	}
	return nBits, nSamples, nil
}
