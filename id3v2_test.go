package tag

import (
	"bytes"
	"reflect"
	"testing"
)

func TestUnsynchroniser(t *testing.T) {
	tests := []struct {
		input  []byte
		output []byte
	}{
		{
			input:  []byte{},
			output: []byte{},
		},

		{
			input:  []byte{0x00},
			output: []byte{0x00},
		},

		{
			input:  []byte{0xFF},
			output: []byte{0xFF},
		},

		{
			input:  []byte{0xFF, 0x00},
			output: []byte{0xFF},
		},

		{
			input:  []byte{0xFF, 0x00, 0x00},
			output: []byte{0xFF, 0x00},
		},

		{
			input:  []byte{0xFF, 0x00, 0x01},
			output: []byte{0xFF, 0x01},
		},

		{
			input:  []byte{0xFF, 0x00, 0xFF, 0x00},
			output: []byte{0xFF, 0xFF},
		},

		{
			input:  []byte{0xFF, 0x00, 0xFF, 0xFF, 0x00},
			output: []byte{0xFF, 0xFF, 0xFF},
		},

		{
			input:  []byte{0x00, 0x01, 0x02},
			output: []byte{0x00, 0x01, 0x02},
		},
	}

	for ii, tt := range tests {
		r := bytes.NewReader(tt.input)
		ur := unsynchroniser{Reader: r}
		got := make([]byte, len(tt.output))
		n, err := ur.Read(got)
		if n != len(got) || err != nil {
			t.Errorf("[%d] got: n = %d, err = %v, expected: n = %d, err = nil", ii, n, err, len(got))
		}
		if !reflect.DeepEqual(got, tt.output) {
			t.Errorf("[%d] got: %v, expected %v", ii, got, tt.output)
		}
	}
}

func TestUnsynchroniserSplitReads(t *testing.T) {
	tests := []struct {
		input  []byte
		output []byte
		split  []int
	}{
		{
			input:  []byte{0x00, 0xFF, 0x00},
			output: []byte{0x00, 0xFF},
			split:  []int{1, 1},
		},

		{
			input:  []byte{0xFF, 0x00, 0x01},
			output: []byte{0xFF, 0x01},
			split:  []int{1, 1},
		},

		{
			input:  []byte{0xFF, 0x00, 0x01, 0x02},
			output: []byte{0xFF, 0x01, 0x02},
			split:  []int{1, 1, 1},
		},

		{
			input:  []byte{0xFF, 0x00, 0x01, 0x02},
			output: []byte{0xFF, 0x01, 0x02},
			split:  []int{2, 1},
		},

		{
			input:  []byte{0xFF, 0x00, 0x01, 0x02},
			output: []byte{0xFF, 0x01, 0x02},
			split:  []int{1, 2},
		},
	}

	for ii, tt := range tests {
		r := bytes.NewReader(tt.input)
		ur := unsynchroniser{Reader: r}
		var got []byte
		for i, l := range tt.split {
			chunk := make([]byte, l)
			n, err := ur.Read(chunk)
			if n != len(chunk) || err != nil {
				t.Errorf("[%d : %d] got: n = %d, err = %v, expected: n = %d, err = nil", ii, i, n, err, l)
			}
			got = append(got, chunk...)
		}
		if !reflect.DeepEqual(got, tt.output) {
			t.Errorf("[%d] got: %v, expected %v", ii, got, tt.output)
		}
	}
}
