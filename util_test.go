// Copyright 2015, David Howden
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tag

import (
	"testing"
)

func TestGetBit(t *testing.T) {
	for i := uint(0); i < 8; i++ {
		b := byte(1 << i)
		got := getBit(b, i)
		if !got {
			t.Errorf("getBit(%v, %v) = %v, expected %v", b, i, got, true)
		}
	}
}

func TestGet7BitChunkedInt(t *testing.T) {
	tests := []struct {
		input  []byte
		output int
	}{
		{
			[]byte{},
			0,
		},
		{
			[]byte{0x01},
			1,
		},
		{
			[]byte{0x7F, 0x7F},
			0x3FFF,
		},
	}

	for ii, tt := range tests {
		got := get7BitChunkedInt(tt.input)
		if got != tt.output {
			t.Errorf("[%d] get7BitChunkedInt(%v) = %v, expected %v", ii, tt.input, got, tt.output)
		}
	}
}

func TestGetInt(t *testing.T) {
	tests := []struct {
		input  []byte
		output int
	}{
		{
			[]byte{},
			0,
		},
		{
			[]byte{0x01},
			1,
		},
		{
			[]byte{0xF1, 0xF2},
			0xF1F2,
		},
		{
			[]byte{0xF1, 0xF2, 0xF3},
			0xF1F2F3,
		},
		{
			[]byte{0xF1, 0xF2, 0xF3, 0xF4},
			0xF1F2F3F4,
		},
	}

	for ii, tt := range tests {
		got := getInt(tt.input)
		if got != tt.output {
			t.Errorf("[%d] getInt(%v) = %v, expected %v", ii, tt.input, got, tt.output)
		}
	}
}
