package flex_test

import (
	"os"
	"strings"
	"testing"

	flex "github.com/sqpp/flex-golang"
)

// TestDecodeFlexWAV ensures the new native PLL demodulator correctly
// extracts the known FLEX alphanumeric messages from Flex-1600.wav.
// multimon-ng reference output for the same file:
// FLEX: 1600/2/A/21/045 A:0000516 ALN: 800-444-4444   [30/30]  (frame 107)
// FLEX: 1600/2/A/22/048 A:0001789 ALN: HELP! HELP! HELP! HELP! HELP! HELP! HELP!  (frame 110)
// FLEX: 1600/2/A/22/049 A:0001913 ALN: Call Me ASAP!   (frame 111)
func TestDecodeFlexWAV(t *testing.T) {
	wavData, err := os.ReadFile("./test_1600.wav")
	if err != nil {
		t.Skip("./test_1600.wav not found:", err)
	}

	msgs, err := flex.DecodeFromAudio(wavData)
	if err != nil {
		t.Fatalf("DecodeFromAudio error: %v", err)
	}

	found1789 := false
	found1913 := false

	t.Logf("Found %d total messages", len(msgs))

	for _, m := range msgs {
		if m.Capcode == 1789 && strings.Contains(m.Text, "CANCELED JOB:") {
			found1789 = true
			t.Logf("SUCCESS: Found capcode 1789 ALN: %q", m.Text)
		}
		if m.Capcode == 1913 && strings.Contains(m.Text, "NEW JOB:") {
			found1913 = true
			t.Logf("SUCCESS: Found capcode 1913 ALN: %q", m.Text)
		}
		t.Logf("ALN: Capcode=%07d text=%q", m.Capcode, m.Text)
	}

	if !found1789 {
		t.Errorf("Failed to decode message for capcode 1789")
	}
	if !found1913 {
		t.Errorf("Failed to decode message for capcode 1913")
	}
}
