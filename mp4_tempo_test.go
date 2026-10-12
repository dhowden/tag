package tag

import (
	"bytes"
	"testing"
)

func TestMP4TempoReadsBigEndianInteger(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value []byte
		want  int
	}{
		{name: "below 256", value: []byte{0x00, 0x78}, want: 120},
		{name: "above 255", value: []byte{0x01, 0x2c}, want: 300},
	} {
		t.Run(tc.name, func(t *testing.T) {
			payload := append([]byte{
				0x00, 0x00, 0x00, byte(16 + len(tc.value)),
				'd', 'a', 't', 'a',
				0x00,
				0x00, 0x00, 0x15,
				0x00, 0x00, 0x00, 0x00,
			}, tc.value...)
			payload[3] = byte(len(payload))
			m := metadataMP4{data: map[string]interface{}{}}
			if err := m.readAtomData(bytes.NewReader(payload), "tmpo", uint32(len(payload)), nil); err != nil {
				t.Fatal(err)
			}
			got, ok := m.data["tmpo"].(int)
			if !ok {
				t.Fatalf("tmpo = %#v, want int %d", m.data["tmpo"], tc.want)
			}
			if got != tc.want {
				t.Fatalf("tmpo = %d, want %d", got, tc.want)
			}
		})
	}
}
