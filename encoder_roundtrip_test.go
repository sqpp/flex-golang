package flex

import "testing"

func TestBIWEncoded(t *testing.T) {
	cw := buildBIW(2, 0)
	info, errs := FLEXBCHDecode32(cw)
	if errs < 0 || !FLEXChecksum(info) {
		t.Fatalf("BIW decode failed: info=0x%X errs=%d", info, errs)
	}
}
