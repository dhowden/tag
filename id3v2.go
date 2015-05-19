// Copyright 2015, David Howden
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tag

import (
	"fmt"
	"io"
	"os"
	"strconv"
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

		b, err := readBytes(r, size)
		if err != nil {
			return nil, err
		}

		name = strings.TrimSpace(name)
		if name == "" {
			break
		}

		// There can be multiple tag with the same name. Append a number to the
		// name if there is more than one.
		rawName := name
		if _, ok := result[rawName]; ok {
			for i := 0; ok; i++ {
				rawName = name + "_" + strconv.Itoa(i)
				_, ok = result[rawName]
			}
		}

		switch {
		case name[0] == 'T':
			txt, err := readTFrame(b)
			if err != nil {
				return nil, err
			}
			result[rawName] = txt

		case name == "COMM" || name == "USLT":
			t, err := readTextWithDescrFrame(b)
			if err != nil {
				return nil, err
			}
			result[rawName] = t

		case name == "APIC":
			p, err := readAPICFrame(b)
			if err != nil {
				return nil, err
			}
			result[rawName] = p

		case name == "PIC":
			p, err := readPICFrame(b)
			if err != nil {
				return nil, err
			}
			result[rawName] = p
		}

		continue
	}
	return result, nil
}

type unsynchroniser struct {
	io.Reader
	ff bool
}

// filter io.Reader which skip the Unsynchronisation bytes
func (r *unsynchroniser) Read(p []byte) (int, error) {
	b := make([]byte, 1)
	i := 0
	for i < len(p) {
		if n, err := r.Reader.Read(b); err != nil || n == 0 {
			return i, err
		}
		if r.ff && b[0] == 0x00 {
			r.ff = false
			continue
		}
		p[i] = b[0]
		i++
		r.ff = (b[0] == 0xFF)
	}
	return i, nil
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
	ur = r
	if h.Unsynchronisation {
		ur = &unsynchroniser{Reader: r}
	}

	f, err := readID3v2Frames(ur, h)
	if err != nil {
		return nil, err
	}

	mp3, err := getMp3Infos(r, false)
	if err != nil {
		return nil, err
	}
	f["stream_type"] = fmt.Sprintf("MPEG %v Layer %v", mp3.Version, mp3.Layer)
	f["stream_bitrate"] = fmt.Sprintf("%v kbps %v", mp3.Bitrate, mp3.Type)
	f["stream_audio"] = fmt.Sprintf("%v Hz %v", mp3.Sampling, mp3.Mode)
	f["stream_size"] = mp3.Size
	f["stream_length"] = int(mp3.Length)

	return metadataID3v2{header: h, frames: f}, nil
}
