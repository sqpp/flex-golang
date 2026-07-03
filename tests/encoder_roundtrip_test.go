package flex_test

import (
	"testing"

	flex "github.com/sqpp/flex-golang"
)

func TestBIWEncoded(t *testing.T) {
	cw := flex.ExportBuildBIW(2, 0)
	info, errs := flex.FLEXBCHDecode32(cw)
	if errs < 0 || !flex.ExportFLEXChecksum(info) {
		t.Fatalf("BIW decode failed: info=0x%X errs=%d", info, errs)
	}
}
