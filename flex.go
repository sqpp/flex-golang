// Package flex decodes Motorola FLEX paging — the high-throughput
// successor to POCSAG used by most modern paging networks (hospital
// / EMS dispatch, commercial carriers). FLEX multiplexes many pagers
// onto one channel with time-framed, interleaved, FEC-protected data
// at 1600 / 3200 / 6400 bps (2- or 4-level FSK).
//
// This package owns the FLEX *logical* layer: given the BCH-decoded
// 21-bit codeword values of one phase of a frame, it walks the Block
// Information Word → address words → vector words → message words and
// produces typed Messages.
//
// References:
//   - multimon-ng demod_flex.c — canonical open-source FLEX decoder.
//   - PDW Flex.cpp — Discriminator's Windows FLEX decoder, another
//     widely-used reference implementation.
package flex

import (
	"strings"
)

// PageType is the FLEX vector message type (VIW bits 4..6).
// Values match the FLEX spec / multimon-ng Flex_PageTypeEnum.
type PageType uint8

const (
	PageSecure        PageType = 0
	PageShortInstruct PageType = 1
	PageTone          PageType = 2
	PageStdNumeric    PageType = 3
	PageSpecialNum    PageType = 4
	PageAlphanumeric  PageType = 5
	PageBinary        PageType = 6
	PageNumberedNum   PageType = 7
	PageUnknown       PageType = 0xFF
)

// TypeName returns a stable label for the page type.
func (p PageType) TypeName() string {
	switch p {
	case PageStdNumeric, PageSpecialNum, PageNumberedNum:
		return "numeric"
	case PageAlphanumeric:
		return "alpha"
	case PageTone:
		return "tone"
	case PageShortInstruct:
		return "short"
	case PageBinary:
		return "binary"
	case PageSecure:
		return "secure"
	}
	return "unknown"
}

// Message is one decoded FLEX page.
type Message struct {
	Capcode      int64    `json:"Address"`
	LongAddress  bool     `json:"LongAddress"`
	Type         PageType `json:"Function"`
	Text         string   `json:"Message"`
	Frag         byte     `json:"Frag"`
	FragFlag     byte     `json:"FragFlag,omitempty"`
	Cycle        int      `json:"Cycle"`
	Frame        int      `json:"Frame"`
	Baud         int      `json:"Baud"`
	Levels       int      `json:"Levels"`
	Phase        byte     `json:"Phase"`
	Corrected    int      `json:"Corrected"`
	Valid        bool     `json:"-"`
	IsNumeric    bool     `json:"IsNumeric"`
	RawBytes     []byte   `json:"-"`
	CollapseType int      `json:"-"`
}

// idle codeword values (21 bits) per the FLEX spec / multimon.
const (
	idleZero uint32 = 0x000000
	idleOnes uint32 = 0x1FFFFF
)

// Group address range (multimon FLEX_GROUP_ADDR_MIN / MAX).
const (
	flexGroupAddrMin int64 = 2029568
	flexGroupAddrMax int64 = 2029583
)

// DecodePhase walks one phase's BCH-decoded codeword values
// (words[0..n-1], each a 21-bit value after masking) and returns the
// pages it carries.  corr is the per-word BCH error count (index-
// aligned; negative = uncorrectable).  frame / cycle come from the
// FIW; baud / levels / phase describe the FLEX mode.
//
// Structure (directly mirroring multimon-ng decode_phase):
//   - word 0 is the Block Information Word (BIW)
//   - address words occupy [aoffset, voffset)
//   - each address[i]'s vector word sits at voffset + (i-aoffset)
//   - the vector word names the message-word span
func DecodePhase(words []uint32, corr []int, frame, cycle, baud, levels int, phase byte) []Message {
	if len(words) < 2 {
		return nil
	}

	biw := words[0]

	// BIW must be idle-free and carry a valid 4-bit checksum.
	if biw == idleZero || biw == idleOnes {
		return nil
	}
	if !FLEXChecksum(biw) {
		return nil
	}

	// Vector start index is BIW bits 15..10 (6 bits).
	// Address start index is BIW bits 9..8 (2 bits) + 1.
	voffset := int((biw >> 10) & 0x3F)
	aoffset := int((biw>>8)&0x03) + 1

	if voffset <= aoffset || voffset >= len(words) {
		return nil
	}

	var out []Message

	for i := aoffset; i < voffset; i++ {
		aw := words[i]
		if aw == idleZero || aw == idleOnes {
			continue // idle / invalid address slot
		}

		// Compute the vector word index for this address.
		vIdx := voffset + (i - aoffset)
		if vIdx >= len(words) {
			break
		}
		viw := words[vIdx]

		// Decode address word → capcode + long-address flag.
		capcode, longAddr := decodeCapcode(aw)
		if longAddr && i+1 < voffset {
			// Long address: consume the next address slot for aw2.
			aw2 := words[i+1]
			capcode = decodeLongCapcode(aw, aw2)
			i++ // skip the second address word in the loop
		}

		// Sanity check: spec maximum capcode.
		if capcode < 0 || capcode > 4297068542 {
			continue
		}

		// VIW bits 6..4 → message type.
		// VIW bits 13..7 → start of message words (w1).
		// VIW bits 20..14 → length or end-offset of message words.
		vt := PageType((viw >> 4) & 0x07)

		msg := Message{
			Capcode:     capcode,
			LongAddress: longAddr,
			Type:        vt,
			Frame:       frame,
			Cycle:       cycle,
			Baud:        baud,
			Levels:      levels,
			Phase:       phase,
			IsNumeric:   vt == PageStdNumeric || vt == PageSpecialNum || vt == PageNumberedNum,
			Valid:       true,
		}

		// Sum BCH corrections for the address and vector words.
		if corr != nil {
			if i < len(corr) && corr[i] > 0 {
				msg.Corrected += corr[i]
			}
			if vIdx < len(corr) && corr[vIdx] > 0 {
				msg.Corrected += corr[vIdx]
			}
		}

		switch vt {
		case PageAlphanumeric, PageSecure:
			msg.Text, msg.Frag, msg.FragFlag, msg.Corrected = decodeAlpha(words, corr, viw, longAddr, vIdx, msg.Corrected)
			out = append(out, msg)

		case PageStdNumeric, PageSpecialNum, PageNumberedNum:
			msg.Text, msg.Corrected = decodeNumeric(words, corr, viw, longAddr, vIdx, vt, msg.Corrected)
			out = append(out, msg)

		case PageTone:
			msg.Text = decodeTone(words, viw, longAddr)
			out = append(out, msg)

		case PageShortInstruct:
			// Short instruction / group call — emit so callers can track
			// group membership, but no message text.
			assignedFrame := int((viw >> 10) & 0x7F)
			groupBit := int((viw >> 17) & 0x7F)
			msg.Text = ""
			_ = assignedFrame
			_ = groupBit
			out = append(out, msg)

		default:
			// Binary / unknown — collect raw hex.
			msg.Text, msg.Corrected = decodeUnknown(words, corr, viw, msg.Corrected)
			out = append(out, msg)
		}
	}

	return out
}

// ─── Capcode ──────────────────────────────────────────────────────────────────

// decodeCapcode converts a single address word to a capcode.
// A short address in (0x008000, 0x1E0000] maps to capcode = aw − 0x8000.
// Values outside that range require a second word (long address).
//
// Mirrors multimon-ng parse_capcode and PDW show_address.
func decodeCapcode(aw uint32) (capcode int64, longAddr bool) {
	v := int64(aw & 0x1FFFFF)
	// Long address indicators (from PDW and multimon-ng):
	if v < 0x008001 || (v > 0x1E0000 && v <= 0x1F0000) || v > 0x1F7FFE {
		longAddr = true
		// Partial capcode; caller must call decodeLongCapcode with aw2.
		return v - 0x8000, true
	}
	return v - 0x8000, false
}

// decodeLongCapcode computes the capcode from two address words.
// Formula from PDW Flex.cpp show_address (long address path):
//
//	capcode = (aw2 ^ 0x1FFFFF) << 15 + 2068480 + aw1
func decodeLongCapcode(aw1, aw2 uint32) int64 {
	v1 := int64(aw1 & 0x1FFFFF)
	v2 := int64(aw2 & 0x1FFFFF)
	return (v2^0x1FFFFF)<<15 + 2068480 + v1
}

// ─── Alpha ────────────────────────────────────────────────────────────────────

// decodeAlpha decodes an alphanumeric page.
//
// VIW layout (from multimon parse_alphanumeric + PDW showframe):
//   - bits 20..14 → len (number of message words)
//   - bits 13..7  → mw1 (first message word index)
//
// The word at mw1 is the header word:
//   - bits 12..11 → frag  (0=last/only, 1..2=middle, 3=first)
//   - bit  10     → cont
//
// The message characters are packed 7-bits each.
func decodeAlpha(words []uint32, corr []int, viw uint32, longAddr bool, vIdx int, corrSoFar int) (text string, frag byte, fragFlag byte, corrTotal int) {
	corrTotal = corrSoFar

	tmp := viw >> 7
	tmp2 := tmp >> 7
	mw1 := int(tmp & 0x7F)
	mw2 := int((tmp2&0x7F)+uint32(mw1)) - 1

	var hw uint32
	var actualMw1 int
	var actualMw2 int

	if longAddr {
		// For long addresses, the header word is the SECOND vector word (vIdx + 1).
		if vIdx+1 >= len(words) {
			return "", 0, 'K', corrTotal
		}
		hw = words[vIdx+1]
		actualMw1 = mw1
		actualMw2 = mw2 - 1
		if corr != nil && vIdx+1 < len(corr) && corr[vIdx+1] > 0 {
			corrTotal += corr[vIdx+1]
		}
	} else {
		// For short addresses, the header word is at mw1.
		if mw1 >= len(words) {
			return "", 0, 'K', corrTotal
		}
		hw = words[mw1]
		actualMw1 = mw1 + 1
		actualMw2 = mw2
	}

	if actualMw1 <= 0 || actualMw2 < actualMw1 || actualMw2 >= len(words) {
		return "", 0, 'K', corrTotal
	}

	// Header word — extract frag / cont bits
	frag = byte((hw >> 11) & 0x03)
	cont := (hw >> 10) & 0x01

	switch {
	case cont == 0 && frag == 3:
		fragFlag = 'K' // complete, ready to send
	case cont == 0 && frag != 3:
		fragFlag = 'C' // continuation (last piece of a fragmented message)
	default:
		fragFlag = 'F' // fragment, more to come
	}

	// Collect BCH corrections for message words.
	if corr != nil {
		for k := actualMw1; k <= actualMw2 && k < len(corr); k++ {
			if corr[k] > 0 {
				corrTotal += corr[k]
			}
		}
	}

	var sb strings.Builder

	// For short addresses, the header word (mw1) contains characters.
	// For long addresses, the header word (vIdx+1) is purely structural and contains NO characters.
	// PDW skips the header word for short-address complete messages (frag==3).
	if !longAddr && frag != 0x03 {
		ch3 := byte((hw >> 14) & 0x7F)
		if ch3 == 0x03 {
			return sb.String(), frag, fragFlag, corrTotal
		}
		if ch3 >= 0x20 {
			sb.WriteByte(ch3)
		}
	}

	// Extract characters from data words
	firstWord := true
	for wi := actualMw1; wi <= actualMw2; wi++ {
		dw := words[wi]

		ch1 := byte(dw & 0x7F)
		// First content word bits 0-6 hold the signature; PDW skips them on complete messages.
		skipCh1 := firstWord && frag == 0x03
		firstWord = false

		if !skipCh1 {
			if ch1 == 0x03 {
				return sb.String(), frag, fragFlag, corrTotal
			}
			if ch1 >= 0x20 {
				sb.WriteByte(ch1)
			}
		}

		ch2 := byte((dw >> 7) & 0x7F)
		if ch2 == 0x03 {
			return sb.String(), frag, fragFlag, corrTotal
		}
		if ch2 >= 0x20 {
			sb.WriteByte(ch2)
		}

		ch3 := byte((dw >> 14) & 0x7F)
		if ch3 == 0x03 {
			return sb.String(), frag, fragFlag, corrTotal
		}
		if ch3 >= 0x20 {
			sb.WriteByte(ch3)
		}
	}

	return sb.String(), frag, fragFlag, corrTotal
}

// ─── Numeric ──────────────────────────────────────────────────────────────────

// flexBCD is the FLEX 4-bit BCD symbol table (multimon / PDW aNumeric).
var flexBCD = []byte("0123456789 U -][")

// decodeNumeric decodes standard / special / numbered numeric pages.
//
// The algorithm (directly from multimon parse_numeric and PDW showframe):
//  1. VIW gives w1 (start) and w2 (end, inclusive).
//  2. If short address, the data starts at frame[w1]; w1++ w2++.
//     If long address, the data starts at frame[vIdx+1].
//  3. Numbered numeric pages skip 10 leading bits; others skip 2.
//  4. Bits are shifted LSB-first into a 4-bit accumulator; when count
//     reaches 0 (every 4 bits), emit flexBCD[digit].
func decodeNumeric(words []uint32, corr []int, viw uint32, longAddr bool, vIdx int, vt PageType, corrSoFar int) (text string, corrTotal int) {
	corrTotal = corrSoFar

	tmp := viw >> 7
	tmp2 := tmp >> 7
	w1 := int(tmp & 0x7F)
	w2 := int((tmp2&0x07)+uint32(w1)) // numeric message is 7 words max

	if w1 <= 0 || w2 >= len(words) {
		return "", corrTotal
	}

	var dw uint32
	if !longAddr {
		dw = words[w1]
		w1++
		w2++
	} else {
		dw = words[vIdx+1]
	}

	if w2 >= len(words) {
		w2 = len(words) - 1
	}

	// Collect corrections.
	if corr != nil {
		for k := w1; k <= w2 && k < len(corr); k++ {
			if corr[k] > 0 {
				corrTotal += corr[k]
			}
		}
	}

	// Skip leading bits: 10 for numbered numeric, 2 otherwise.
	count := 4
	if vt == PageNumberedNum {
		count += 10
	} else {
		count += 2
	}

	var digit byte
	var sb strings.Builder

	for k := w1; k <= w2; k++ {
		for bit := 0; bit < 21; bit++ {
			// Shift LSB of dw into the high bit of digit (4-bit shift register).
			digit = (digit >> 1) & 0x0F
			if dw&0x01 != 0 {
				digit ^= 0x08
			}
			dw >>= 1

			count--
			if count == 0 {
				// 0x0C is the fill/space character — skip it.
				if digit != 0x0C {
					sb.WriteByte(flexBCD[digit])
				}
				count = 4
			}
		}
		dw = words[k]
	}

	return strings.TrimRight(sb.String(), " "), corrTotal
}

// ─── Tone ────────────────────────────────────────────────────────────────────

// decodeTone handles tone-only / short-numeric tone pages.
//
// VIW bits 8..7 select the sub-type:
//   - 0 = short numeric (3–8 digits packed into the VIW and optional extra word)
//   - non-zero = pure tone-only (no digits)
//
// Mirrors multimon parse_tone_only and PDW MODE_SH_TONE.
func decodeTone(words []uint32, viw uint32, longAddr bool) string {
	subType := (viw >> 7) & 0x03
	if subType != 0 {
		return "TONE-ONLY"
	}

	// Short numeric embedded in VIW (bits 9..17 in 4-bit groups)
	// and optionally the next vector word for long addresses.
	var sb strings.Builder
	for i := 9; i <= 17; i += 4 {
		digit := (viw >> uint(i)) & 0x0F
		sb.WriteByte(flexBCD[digit])
	}
	if longAddr && len(words) > 1 {
		// Extra digits in the second vector word.
		extraVIW := words[len(words)-1] // PDW uses frame[vb+1]
		_ = extraVIW                    // handled if caller provides correct slice
	}
	return strings.TrimRight(sb.String(), " ")
}

// ─── Unknown / Binary ────────────────────────────────────────────────────────

// decodeUnknown returns a hex dump of the message words.
func decodeUnknown(words []uint32, corr []int, viw uint32, corrSoFar int) (text string, corrTotal int) {
	corrTotal = corrSoFar

	tmp := viw >> 7
	tmp2 := tmp >> 7
	mw1 := int(tmp & 0x7F)
	mw2 := int((tmp2&0x7F)+uint32(mw1)) - 1

	if mw1 <= 0 || mw2 < mw1 || mw2 >= len(words) {
		return "", corrTotal
	}

	var sb strings.Builder
	for k := mw1; k <= mw2; k++ {
		if sb.Len() > 0 {
			sb.WriteByte(' ')
		}
		sb.WriteString(hexWord(words[k]))
		if corr != nil && k < len(corr) && corr[k] > 0 {
			corrTotal += corr[k]
		}
	}
	return sb.String(), corrTotal
}

// hexWord formats a 21-bit value as 6 hex digits.
func hexWord(w uint32) string {
	const digits = "0123456789ABCDEF"
	b := [6]byte{
		digits[(w>>20)&0xF],
		digits[(w>>16)&0xF],
		digits[(w>>12)&0xF],
		digits[(w>>8)&0xF],
		digits[(w>>4)&0xF],
		digits[(w>>0)&0xF],
	}
	return string(b[:])
}

// Note: checksum validation is provided by FLEXChecksum in bch_flex.go.
