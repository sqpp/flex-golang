package flex

import (
	"math"
	"sync/atomic"
)

// SyncMarker is the 32-bit FLEX frame sync word that appears in the
// middle of the 64-bit Sync-1 header (multimon FLEX_SYNC_MARKER).
const SyncMarker uint32 = 0xA6C6AAAA

// flexModes maps the 16-bit sync speed code to (baud, levels).
var flexModes = []struct {
	code   uint16
	baud   uint32
	levels uint32
}{
	{0x870C, 1600, 2},
	{0xB068, 1600, 4},
	{0x7B18, 3200, 2},
	{0xDEA0, 3200, 4},
	{0x4C7C, 3200, 4},
}

// PhaseWords is the number of 32-bit codewords per phase of a FLEX
// data block. PhaseBits is that count in bits.
const (
	PhaseWords = 88
	PhaseBits  = PhaseWords * 32 // 2816
)

// DPLL Constants from multimon-ng
const (
	DCOffsetFilter    = 0.010
	PhaseLockedRate   = 0.045
	PhaseUnlockedRate = 0.050
	LockLen           = 24
	SliceThreshold    = 0.667
	DemodTimeout      = 100
)

type decState uint8

const (
	stHuntMarker decState = iota // scanning for the 64-bit Sync-1 pattern
	stFIW                        // reading the 16-bit dotting + 32-bit FIW
	stSync2                      // skipping the Sync-2 gap
	stData                       // de-interleaving the 88-codeword data block
)

// Demodulator is a continuous PLL demodulator that processes raw audio samples
// and acts as the FLEX state machine.
type Demodulator struct {
	sampleFreq uint32
	baud       uint32

	// PLL State
	phase         int64
	sampleLast    float64
	locked        bool
	sampleCount   uint32
	symbolCount   uint32
	envelopeSum   float64
	envelopeCount int
	lockBuf       uint64
	symCount      [4]int
	timeout       int
	nonConsec     int
	envelope      float64
	zero          float64

	// FLEX Protocol State
	st         decState
	syncReg    uint64
	syncBaud   uint32
	syncLevels uint32
	polarity   int // 0=pos, 1=neg

	fiwCount int
	fiwReg   uint32
	cycle    int
	frame    int

	sync2Count uint32
	dataCount  uint32
	c          int
	words      [PhaseWords]uint32

	// Cumulative counters.
	framesSynced atomic.Uint64
	pagesEmitted atomic.Uint64
}

// NewDemodulator returns a Demodulator in the marker-hunt state.
func NewDemodulator(sampleFreq uint32) *Demodulator {
	return &Demodulator{
		sampleFreq: sampleFreq,
		baud:       1600,
		st:         stHuntMarker,
	}
}

// PushSample feeds one raw audio float32 sample into the DPLL. Returns any
// pages completed by this sample (may be nil).
func (d *Demodulator) PushSample(sample float32) []Message {
	if d.buildSymbol(float64(sample)) {
		d.nonConsec = 0
		d.symbolCount++

		var decMax int
		var modalSymbol int
		for j := 0; j < 4; j++ {
			if d.symCount[j] > decMax {
				modalSymbol = j
				decMax = d.symCount[j]
			}
		}
		d.symCount[0] = 0
		d.symCount[1] = 0
		d.symCount[2] = 0
		d.symCount[3] = 0

		if d.locked {
			return d.processSymbol(byte(modalSymbol))
		}

		// Check for lock pattern
		d.lockBuf = (d.lockBuf << 2) | uint64(modalSymbol^0x1)
		lockPattern := d.lockBuf ^ 0x6666666666666666
		lockMask := uint64((1 << (2 * LockLen)) - 1)
		if (lockPattern&lockMask) == 0 || ((^lockPattern)&lockMask) == 0 {
			if !d.locked {
				// debug: Locked
			}
			d.locked = true
			d.lockBuf = 0
			d.symbolCount = 0
			d.sampleCount = 0
		}

		d.timeout++
		if d.timeout > DemodTimeout {
			if d.locked {
				// debug: Timeout
			}
			d.locked = false
		}
	}
	return nil
}

func (d *Demodulator) buildSymbol(sample float64) bool {
	phaseMax := int64(100 * d.sampleFreq)
	phaseRate := phaseMax * int64(d.baud) / int64(d.sampleFreq)
	phasePercent := float64(100.0 * float64(d.phase) / float64(phaseMax))

	d.sampleCount++

	if d.st == stHuntMarker {
		filterWeight := float64(d.sampleFreq) * DCOffsetFilter
		d.zero = (d.zero*filterWeight + sample) / (filterWeight + 1)
	}
	sample -= d.zero

	if d.locked {
		if d.st == stHuntMarker {
			d.envelopeSum += math.Abs(sample)
			d.envelopeCount++
			d.envelope = d.envelopeSum / float64(d.envelopeCount)
		}
	} else {
		d.envelope = 0
		d.envelopeSum = 0
		d.envelopeCount = 0
		d.baud = 1600
		d.timeout = 0
		d.nonConsec = 0
		d.st = stHuntMarker
	}

	if phasePercent > 10 && phasePercent < 90 {
		if sample > 0 {
			if sample > d.envelope*SliceThreshold {
				d.symCount[3]++
			} else {
				d.symCount[2]++
			}
		} else {
			if sample < -d.envelope*SliceThreshold {
				d.symCount[0]++
			} else {
				d.symCount[1]++
			}
		}
	}

	if (d.sampleLast < 0 && sample >= 0) || (d.sampleLast >= 0 && sample < 0) {
		phaseError := float64(0)
		if phasePercent < 50 {
			phaseError = float64(d.phase)
		} else {
			phaseError = float64(d.phase - phaseMax)
		}

		if d.locked {
			d.phase -= int64(phaseError * PhaseLockedRate)
		} else {
			d.phase -= int64(phaseError * PhaseUnlockedRate)
		}

		if phasePercent > 10 && phasePercent < 90 {
			d.nonConsec++
			if d.nonConsec > 20 && d.locked {
				// debug: NonConsec
				d.locked = false
			}
		} else {
			d.nonConsec = 0
		}
		d.timeout = 0
	}
	d.sampleLast = sample

	d.phase += phaseRate

	if d.phase > phaseMax {
		d.phase -= phaseMax
		return true
	}
	return false
}

func (d *Demodulator) processSymbol(sym byte) []Message {
	symRectified := sym
	if d.polarity == 1 {
		symRectified = 3 - sym
	}

	switch d.st {
	case stHuntMarker:
		b := uint64(0)
		if sym < 2 { // unrectified symbol for sync detection
			b = 1
		}
		d.syncReg = (d.syncReg << 1) | b

		if code, ok := checkSync1(d.syncReg); ok {
			// debug: Found POS
			d.polarity = 0
			d.enterFIW(code)
		} else if code, ok := checkSync1(^d.syncReg); ok {
			// debug: Found NEG
			d.polarity = 1
			d.enterFIW(code)
		} else {
			d.st = stHuntMarker
		}
		d.fiwCount = 0
		d.fiwReg = 0

	case stFIW:
		d.fiwCount++
		if d.fiwCount > 16 {
			bit := uint32(0)
			if symRectified > 1 {
				bit = 0x80000000
			}
			d.fiwReg = (d.fiwReg >> 1) | bit
		}

		if d.fiwCount == 48 {
			d.processFIW()
			d.sync2Count = 0
			d.baud = d.syncBaud
			d.st = stSync2
		}

	case stSync2:
		d.sync2Count++
		if d.sync2Count == d.syncBaud*25/1000 {
			// debug: Sync2 done
			d.dataCount = 0
			d.c = 0
			for i := range d.words {
				d.words[i] = 0
			}
			d.st = stData
		}

	case stData:
		bit := uint32(0)
		if symRectified > 1 {
			bit = 0x80000000
		}

		word := ((d.c >> 5) & 0xFFF8) | (d.c & 7)
		if word < PhaseWords {
			d.words[word] >>= 1
			d.words[word] |= bit
		}
		d.c++

		d.dataCount++
		if d.dataCount == d.syncBaud*1760/1000 {
			// debug: Data done
			msgs := d.decodeBlock()
			d.baud = 1600
			d.st = stHuntMarker
			d.dataCount = 0
			return msgs
		}
	}
	return nil
}

func countBits32(v uint32) int {
	c := 0
	for v > 0 {
		v &= v - 1
		c++
	}
	return c
}

func countBits16(v uint16) int {
	c := 0
	for v > 0 {
		v &= v - 1
		c++
	}
	return c
}

func checkSync1(buf uint64) (uint16, bool) {
	marker := uint32((buf >> 16) & 0xFFFFFFFF)

	lo := uint16(buf & 0xFFFF)

	dist := countBits32(marker^SyncMarker)
	if dist <= 4 {
		code := ^lo
		if code == 0x870C || code == 0x870D || code == 0x8720 || code == 0x8721 || code == 0x8722 {
			return code, true
		}
	}
	return 0, false
}

func (d *Demodulator) enterFIW(code uint16) {
	d.syncBaud = 1600
	d.syncLevels = 2
	for _, m := range flexModes {
		if m.code == code {
			d.syncBaud = m.baud
			d.syncLevels = m.levels
			break
		}
	}
	d.st = stFIW
}

func (d *Demodulator) processFIW() bool {
	info, errs := FLEXBCHDecode32(d.fiwReg)
	chk := FLEXChecksum(info)
	// debug: FIW raw
	if errs >= 0 && chk {
		d.cycle = int((info >> 4) & 0x0F)
		d.frame = int((info >> 8) & 0x7F)
		d.framesSynced.Add(1)
		return true
	}
	return false
}

func (d *Demodulator) decodeBlock() []Message {
	infos := make([]uint32, PhaseWords)
	corr := make([]int, PhaseWords)

	errCnt := 0
	for i := 0; i < PhaseWords; i++ {
		info, errs := FLEXBCHDecode32(d.words[i])
		infos[i] = info & 0x1FFFFF
		corr[i] = errs
		if errs >= 0 {
			errCnt++
		}
	}
	// debug: Words
	// debug: Block decode

	msgs := DecodePhase(infos, corr, d.frame, d.cycle, int(d.syncBaud), int(d.syncLevels), 'A')
	d.pagesEmitted.Add(uint64(len(msgs)))
	return msgs
}

// Stats reports cumulative decode counters.
type Stats struct {
	FramesSynced uint64
	PagesEmitted uint64
}

// Stats returns the current cumulative counters.
func (d *Demodulator) Stats() Stats {
	return Stats{
		FramesSynced: d.framesSynced.Load(),
		PagesEmitted: d.pagesEmitted.Load(),
	}
}
