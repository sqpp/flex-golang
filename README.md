# FLEX-GO v1.0.0

A complete Go implementation of the Motorola FLEX pager protocol decoder. Inspired by and compatible with the [`pocsag-golang`](https://github.com/sqpp/pocsag-golang) architecture.

## What it can do

- Decode FLEX frames directly from standard WAV audio files (1600/2, 3200/2, 3200/4, 6400/4 modes)
- Handle Frame Information Words (FIW) and Block Information Words (BIW) natively
- Full BCH(31,21) and Motorola 4-bit checksum validation and 2-bit error correction
- Extract Capcode addresses and Alphanumeric/Numeric message payloads
- Pure native Go DPLL demodulator with phase-locking — no CGO, `sox`, or external C tools required
- JSON output for scripting and API integration
- Drop-in CLI replacement perfectly mirroring `pocsag-decode`'s payload format

---

## Installation

```bash
# Decoder
go install github.com/sqpp/flex-golang/cmd/flex-decode@latest
```

Or build from source:

```bash
git clone https://github.com/sqpp/flex-golang.git
cd flex-golang
make build
# Binary lands in: bin/flex-decode
```

---

## Decoder (`flex-decode`)

Decode a WAV file containing 2-FSK or 4-FSK FLEX audio into messages.

**Options:**
- `-i` / `--input` — input WAV file (required)
- `-j` / `--json` — print result as JSON instead of human-readable text
- `-v` / `--version` — show version info

**Examples:**

```bash
flex-decode -i Flex-1600.wav
flex-decode -i Flex-1600.wav --json
flex-decode --version
```

**Normal output:**
```
FLEX-1600/2: Decoded messages:
Address:    1913  Function: 5  ALPHA    Message: NEW JOB: BED: B6 ROOM 19 BED 02
```

**JSON output:**
```json
{
  "baud": 1600,
  "messages": [
    {
      "address": 1913,
      "function": 5,
      "message": "NEW JOB: BED: B6 ROOM 19 BED 02",
      "type": "alphanumeric"
    }
  ],
  "success": true
}
```

---

## Using as a Go library

```go
package main

import (
    "fmt"
    "os"
    flex "github.com/sqpp/flex-golang"
)

func main() {
    wavData, _ := os.ReadFile("Flex-1600.wav")
    
    // Decode audio directly into messages via the native DPLL
    messages, _ := flex.DecodeFromAudio(wavData)
    
    for _, msg := range messages {
        fmt.Printf("Message to %07d: %s\n", msg.Capcode, msg.Text)
    }
}
```

---

## Roadmap

- **Encoder (`flex`)**: Generate FLEX WAV signals (Pending)
- **Burst Encoder (`flex-burst`)**: Pack multiple messages into single frames (Pending)

## License
Built natively for pure-Go portability. Decoding architecture originally inspired by PDW and multimon-ng.
