package flex

import "testing"

func TestSyncInBitstream(t *testing.T) {
	bits, err := BuildBitstream(EncodeMessage{
		Capcode: 1913, Type: "alpha", Text: "X",
	}, Mode1600_2, 0, 0)
	if err != nil {
		t.Fatal(err)
	}

	var syncReg uint64
	syncAt := -1
	for i, bit := range bits {
		b := uint64(0)
		if bit == 0 {
			b = 1
		}
		syncReg = (syncReg << 1) | b
		if i >= 63 {
			if _, ok := checkSync1(syncReg); ok {
				syncAt = i
				break
			}
		}
	}
	if syncAt < 0 {
		t.Fatal("sync not found")
	}
	t.Logf("sync at bit %d (total bits %d)", syncAt, len(bits))
}

func TestPushPerfectSquares(t *testing.T) {
	bits, err := BuildBitstream(EncodeMessage{
		Capcode: 1913, Type: "alpha", Text: "HELLO WORLD",
	}, Mode1600_2, 0, 0)
	if err != nil {
		t.Fatal(err)
	}

	d := NewDemodulator(SampleRate)
	const spb = SampleRate / 1600
	for _, bit := range bits {
		v := float32(-12000)
		if bit != 0 {
			v = 12000
		}
		for s := 0; s < spb; s++ {
			if msgs := d.PushSample(v); len(msgs) > 0 {
				t.Logf("decoded: %+v", msgs)
			}
		}
	}
	st := d.Stats()
	t.Logf("frames=%d pages=%d", st.FramesSynced, st.PagesEmitted)
	if st.FramesSynced == 0 {
		t.Fatal("no sync on ideal square wave feed")
	}
}
