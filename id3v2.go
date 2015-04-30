// Copyright 2015, David Howden
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tag

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// ID3v2Header is a type which represents an ID3v2 tag header.
type ID3v2Header struct {
	Version           Format
	Unsynchronisation bool
	ExtendedHeader    bool
	Experimental      bool
	Size              int
}

// readID3v2Header reads the ID3v2 header from the given io.Reader.
func readID3v2Header(r io.Reader) (*ID3v2Header, error) {
	b, err := readBytes(r, 10)
	if err != nil {
		return nil, fmt.Errorf("expected to read 10 bytes (ID3v2Header): %v", err)
	}

	if string(b[0:3]) != "ID3" {
		return nil, fmt.Errorf("expected to read \"ID3\"")
	}

	b = b[3:]
	var vers Format
	switch uint(b[0]) {
	case 2:
		vers = ID3v2_2
	case 3:
		vers = ID3v2_3
	case 4:
		vers = ID3v2_4
	case 0, 1:
		fallthrough
	default:
		return nil, fmt.Errorf("ID3 version: %v, expected: 2, 3 or 4", uint(b[0]))
	}

	// NB: We ignore b[1] (the revision) as we don't currently rely on it.
	return &ID3v2Header{
		Version:           vers,
		Unsynchronisation: getBit(b[2], 7),
		ExtendedHeader:    getBit(b[2], 6),
		Experimental:      getBit(b[2], 5),
		Size:              get7BitChunkedInt(b[3:7]),
	}, nil
}

// ID3v2FrameFlags is a type which represents the flags which can be set on an ID3v2 frame.
type ID3v2FrameFlags struct {
	// Message
	TagAlterPreservation  bool
	FileAlterPreservation bool
	ReadOnly              bool

	// Format
	GroupIdentity       bool
	Compression         bool
	Encryption          bool
	Unsynchronisation   bool
	DataLengthIndicator bool
}

func readID3v2FrameFlags(r io.Reader) (*ID3v2FrameFlags, error) {
	b, err := readBytes(r, 2)
	if err != nil {
		return nil, err
	}

	msg := b[0]
	fmt := b[1]

	return &ID3v2FrameFlags{
		TagAlterPreservation:  getBit(msg, 6),
		FileAlterPreservation: getBit(msg, 5),
		ReadOnly:              getBit(msg, 4),
		GroupIdentity:         getBit(fmt, 7),
		Compression:           getBit(fmt, 3),
		Encryption:            getBit(fmt, 2),
		Unsynchronisation:     getBit(fmt, 1),
		DataLengthIndicator:   getBit(fmt, 0),
	}, nil
}

func readID3v2_2FrameHeader(r io.Reader) (name string, size int, headerSize int, err error) {
	name, err = readString(r, 3)
	if err != nil {
		return
	}
	size, err = readInt(r, 3)
	if err != nil {
		return
	}
	headerSize = 6
	return
}

func readID3v2_3FrameHeader(r io.Reader) (name string, size int, headerSize int, err error) {
	name, err = readString(r, 4)
	if err != nil {
		return
	}
	size, err = readInt(r, 4)
	if err != nil {
		return
	}
	headerSize = 8
	return
}

func readID3v2_4FrameHeader(r io.Reader) (name string, size int, headerSize int, err error) {
	name, err = readString(r, 4)
	if err != nil {
		return
	}
	size, err = read7BitChunkedInt(r, 4)
	if err != nil {
		return
	}
	headerSize = 8
	return
}

// readID3v2Frames reads ID3v2 frames from the given reader using the ID3v2Header.
func readID3v2Frames(r io.Reader, h *ID3v2Header) (map[string]interface{}, error) {
	offset := 10 // the size of the header
	result := make(map[string]interface{})

	for offset < h.Size {
		var err error
		var name string
		var size, headerSize int
		var flags *ID3v2FrameFlags

		switch h.Version {
		case ID3v2_2:
			name, size, headerSize, err = readID3v2_2FrameHeader(r)

		case ID3v2_3:
			name, size, headerSize, err = readID3v2_3FrameHeader(r)
			if err != nil {
				return nil, err
			}
			flags, err = readID3v2FrameFlags(r)
			headerSize += 2

		case ID3v2_4:
			name, size, headerSize, err = readID3v2_4FrameHeader(r)
			if err != nil {
				return nil, err
			}
			flags, err = readID3v2FrameFlags(r)
			headerSize += 2
		}

		if err != nil {
			return nil, err
		}
		// if size=0, we certainly are in a padding zone. ignore the rest of
		// the tags
		if size == 0 {
			break
		}

		offset += headerSize + size

		// Check this stuff out...
		if flags != nil && flags.DataLengthIndicator {
			_, err = read7BitChunkedInt(r, 4) // read 4
			if err != nil {
				return nil, err
			}
			size -= 4
		}

		if flags != nil && flags.Unsynchronisation {
			// FIXME: Implement this.
			continue
		}

		name = strings.TrimSpace(name)
		if name == "" {
			break
		}

		b, err := readBytes(r, size)
		if err != nil {
			return nil, err
		}

		switch {
		case name[0] == 'T':
			txt, err := readTFrame(b)
			if err != nil {
				return nil, err
			}
			result[name] = txt

		case name == "APIC":
			p, err := readAPICFrame(b)
			if err != nil {
				return nil, err
			}
			result[name] = p

		case name == "PIC":
			p, err := readPICFrame(b)
			if err != nil {
				return nil, err
			}
			result[name] = p
		}

		continue
	}
	return result, nil
}

type Unsynchroniser struct {
	orig      io.Reader
	prevWasFF bool
}

// filter io.Reader which skip the Unsynchronisation bytes
func (r *Unsynchroniser) Read(p []byte) (int, error) {
	for i := 0; i < len(p); i++ {
		// there is only one byte to read.
		if i == len(p)-1 {
			if n, err := r.orig.Read(p[i : i+1]); n == 0 || err != nil {
				return i, err
			}
			// we need to read this last byte once more
			if r.prevWasFF && p[i] == 0 {
				i--
				r.prevWasFF = false
			}
			r.prevWasFF = (p[i] == 255)
			continue
		}
		if n, err := r.orig.Read(p[i : i+2]); n == 0 || err != nil {
			return i, err
		}
		if r.prevWasFF && p[i] == 0 {
			p[i] = p[i+1]
			r.prevWasFF = (p[i+1] == 255)
			continue
		}
		if p[i] == 255 && p[i+1] == 0 {
			r.prevWasFF = false
			continue
		}
		r.prevWasFF = (p[i+1] == 255)
		// these 2 bytes are fine, we skip none
		i++
	}
	return len(p), nil
}

// ReadID3v2Tags parses ID3v2.{2,3,4} tags from the io.ReadSeeker into a Metadata, returning
// non-nil error on failure.
func ReadID3v2Tags(r io.ReadSeeker) (Metadata, error) {
	_, err := r.Seek(0, os.SEEK_SET)
	if err != nil {
		return nil, err
	}

	h, err := readID3v2Header(r)
	if err != nil {
		return nil, err
	}

	var ur io.Reader

	if h.Unsynchronisation {
		ur = &Unsynchroniser{orig: r}
	} else {
		ur = r
	}

	f, err := readID3v2Frames(ur, h)
	if err != nil {
		return nil, err
	}
	return metadataID3v2{header: h, frames: f}, nil
}
