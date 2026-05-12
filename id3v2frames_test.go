// Copyright 2024, David Howden
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tag

import (
	"encoding/binary"
	"testing"
)

func TestDecodeUTF16OddLength(t *testing.T) {
	// "AB" in UTF-16BE with a stray trailing null byte appended by the encoder.
	b := []byte{0x00, 0x41, 0x00, 0x42, 0x00}
	got, err := decodeUTF16(b, binary.BigEndian)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "AB" {
		t.Fatalf("got %q, want %q", got, "AB")
	}
}

func TestDecodeTextUTF16OddLength(t *testing.T) {
	// encoding byte 0x01 (UTF-16 with BOM), little-endian BOM, "AB", stray null.
	b := []byte{0x01, 0xFF, 0xFE, 0x41, 0x00, 0x42, 0x00, 0x00}
	got, err := decodeText(b[0], b[1:])
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "AB" {
		t.Fatalf("got %q, want %q", got, "AB")
	}
}
