package tag

import (
	"bytes"
	"encoding/binary"
	"runtime"
	"testing"
)

// flacWithPictureDataLen builds a minimal FLAC file whose single (last)
// metadata block is a PICTURE block advertising dataLen bytes of image data
// while carrying none. It exercises readPictureBlock with an untrusted length.
func flacWithPictureDataLen(dataLen uint32) []byte {
	b := &bytes.Buffer{}
	b.WriteString("fLaC")
	b.WriteByte(0x80 | byte(pictureBlock)) // last-block flag + picture block type
	b.Write([]byte{0, 0, 0})               // block length (unused for picture)
	be := func(v uint32) { binary.Write(b, binary.BigEndian, v) }
	be(0)       // picture type = Other
	be(0)       // mime length
	be(0)       // description length
	be(0)       // width
	be(0)       // height
	be(0)       // color depth
	be(0)       // colors used
	be(dataLen) // picture data length (bogus)
	return b.Bytes()
}

// TestFLACPictureAllocationBounded ensures a bogus picture-data length can't
// force a large allocation. Before the fix, readPictureBlock did
// make([]byte, dataLen) directly, so a ~40 byte file advertising a large
// picture forced that many bytes to be allocated (memory-amplification DoS).
func TestFLACPictureAllocationBounded(t *testing.T) {
	const maxAlloc = 20 << 20 // decoding a tiny file must stay well under this
	data := flacWithPictureDataLen(100 << 20)

	if len(data) > 4096 {
		t.Fatalf("test input unexpectedly large: %d bytes", len(data))
	}

	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	// A truncated picture block is expected to error; we only care that it
	// does not allocate based on the advertised length.
	_, _ = ReadFrom(bytes.NewReader(data))

	runtime.ReadMemStats(&m2)
	if alloc := m2.TotalAlloc - m1.TotalAlloc; alloc > maxAlloc {
		t.Fatalf("decoding a %d-byte file allocated %d bytes (%.1f MiB); want < %d",
			len(data), alloc, float64(alloc)/(1<<20), maxAlloc)
	}
}
