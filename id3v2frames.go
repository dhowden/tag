// Copyright 2015, David Howden
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tag

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
	"unicode/utf16"
)

func readTFrame(b []byte) (string, error) {
	txt, err := parseText(b)
	if err != nil {
		return "", err
	}
	return strings.Join(strings.Split(txt, string([]byte{0})), ""), nil
}

func parseText(b []byte) (string, error) {
	if len(b) == 0 {
		return "", nil
	}
	return decodeText(b[0], b[1:])
}

func decodeText(enc byte, b []byte) (string, error) {
	if len(b) == 0 {
		return "", nil
	}

	switch enc {
	case 0: // ISO-8859-1
		return decodeISO8859(b), nil

	case 1: // UTF-16 with byte order marker
		if len(b) == 1 {
			return "", nil
		}
		return decodeUTF16WithBOM(b)

	case 2: // UTF-16 without byte order (assuming BigEndian)
		if len(b) == 1 {
			return "", nil
		}
		return decodeUTF16(b, binary.BigEndian), nil

	case 3: // UTF-8
		return string(b), nil

	default:
		return "", fmt.Errorf("invalid encoding byte %x", enc)
	}
}

func encodingDelim(enc byte) ([]byte, error) {
	switch enc {
	case 0, 3: // see decodeText above
		return []byte{0}, nil
	case 1, 2: // see decodeText above
		return []byte{0, 0}, nil
	default:
		return nil, fmt.Errorf("invalid encoding byte %x", enc)
	}
}

func dataSplit(b []byte, enc byte) ([][]byte, error) {
	delim, err := encodingDelim(enc)
	if err != nil {
		return nil, err
	}
	result := bytes.SplitN(b, delim, 2)

	if len(result) <= 1 {
		return result, nil
	}

	if result[1][0] == 0 {
		// there was a double (or triple) 0 and we cut too early
		result[0] = append(result[0], make([]byte, 0)...)
		result[1] = result[1][1:]
	}

	return result, nil
}

func decodeISO8859(b []byte) string {
	r := make([]rune, len(b))
	for i, x := range b {
		r[i] = rune(x)
	}
	return string(r)
}

func decodeUTF16WithBOM(b []byte) (string, error) {
	var bo binary.ByteOrder
	switch {
	case b[0] == 0xFE && b[1] == 0xFF:
		bo = binary.BigEndian

	case b[0] == 0xFF && b[1] == 0xFE:
		bo = binary.LittleEndian

	default:
		return "", fmt.Errorf("invalid byte order marker %x %x", b[0], b[1])
	}
	return decodeUTF16(b[2:], bo), nil
}

func decodeUTF16(b []byte, bo binary.ByteOrder) string {
	s := make([]uint16, 0, len(b)/2)
	for i := 0; i < len(b); i += 2 {
		s = append(s, bo.Uint16(b[i:i+2]))
	}
	return string(utf16.Decode(s))
}

// Comm is a type used in COMM and USLT tag. It's a text with a description and
// a specified language
type Comm struct {
	Language    string
	Description string
	Text        string
}

// String returns a string representation of the underlying Comm instance.
func (t Comm) String() string {
	return fmt.Sprintf("Text{Lang: '%v', Description: '%v', %v lines}",
		t.Language, t.Description, strings.Count(t.Text, "\n"))
}

// IDv2.{3,4}
// -- Header
// <Header for 'Unsynchronised lyrics/text transcription', ID: "USLT">
// <Header for 'Comment', ID: "COMM">
// -- readTextWithDescrFrame
// Text encoding       $xx
// Language            $xx xx xx
// Content descriptor  <text string according to encoding> $00 (00)
// Lyrics/text         <full text string according to encoding>
func readTextWithDescrFrame(b []byte) (*Comm, error) {
	enc := b[0]

	descTextSplit, err := dataSplit(b[4:], enc)
	if err != nil {
		return nil, err
	}
	desc, err := decodeText(enc, descTextSplit[0])
	if err != nil {
		return nil, fmt.Errorf("error decoding tag description text: %v", err)
	}

	text, err := decodeText(enc, descTextSplit[1])
	if err != nil {
		return nil, fmt.Errorf("error decoding tag text: %v", err)
	}

	return &Comm{
		Language:    string(b[1:4]),
		Description: desc,
		Text:        text,
	}, nil
}

var pictureTypes = map[byte]string{
	0x00: "Other",
	0x01: "32x32 pixels 'file icon' (PNG only)",
	0x02: "Other file icon",
	0x03: "Cover (front)",
	0x04: "Cover (back)",
	0x05: "Leaflet page",
	0x06: "Media (e.g. lable side of CD)",
	0x07: "Lead artist/lead performer/soloist",
	0x08: "Artist/performer",
	0x09: "Conductor",
	0x0A: "Band/Orchestra",
	0x0B: "Composer",
	0x0C: "Lyricist/text writer",
	0x0D: "Recording Location",
	0x0E: "During recording",
	0x0F: "During performance",
	0x10: "Movie/video screen capture",
	0x11: "A bright coloured fish",
	0x12: "Illustration",
	0x13: "Band/artist logotype",
	0x14: "Publisher/Studio logotype",
}

// Picture is a type which represents an attached picture extracted from metadata.
type Picture struct {
	Ext         string // Extension of the picture file.
	MIMEType    string // MIMEType of the picture.
	Type        string // Type of the picture (see pictureTypes).
	Description string // Description.
	Data        []byte // Raw picture data.
}

// String returns a string representation of the underlying Picture instance.
func (p Picture) String() string {
	return fmt.Sprintf("Picture{Ext: %v, MIMEType: %v, Type: %v, Description: %v, Data.Size: %v}",
		p.Ext, p.MIMEType, p.Type, p.Description, len(p.Data))
}

// IDv2.2
// -- Header
// Attached picture   "PIC"
// Frame size         $xx xx xx
// -- readPICFrame
// Text encoding      $xx
// Image format       $xx xx xx
// Picture type       $xx
// Description        <textstring> $00 (00)
// Picture data       <binary data>
func readPICFrame(b []byte) (*Picture, error) {
	enc := b[0]
	ext := string(b[1:4])
	picType := b[4]

	descDataSplit, err := dataSplit(b[5:], enc)
	if err != nil {
		return nil, err
	}
	desc, err := decodeText(enc, descDataSplit[0])
	if err != nil {
		return nil, fmt.Errorf("error decoding PIC description text: %v", err)
	}

	var mimeType string
	switch ext {
	case "jpeg", "jpg":
		mimeType = "image/jpeg"
	case "png":
		mimeType = "image/png"
	}

	return &Picture{
		Ext:         ext,
		MIMEType:    mimeType,
		Type:        pictureTypes[picType],
		Description: desc,
		Data:        descDataSplit[1],
	}, nil
}

// IDv2.{3,4}
// -- Header
// <Header for 'Attached picture', ID: "APIC">
// -- readAPICFrame
// Text encoding   $xx
// MIME type       <text string> $00
// Picture type    $xx
// Description     <text string according to encoding> $00 (00)
// Picture data    <binary data>
func readAPICFrame(b []byte) (*Picture, error) {
	enc := b[0]
	mimeDataSplit := bytes.SplitN(b[1:], []byte{0}, 2)
	mimeType := string(mimeDataSplit[0])

	b = mimeDataSplit[1]
	picType := b[0]

	descDataSplit, err := dataSplit(b[1:], enc)
	if err != nil {
		return nil, err
	}
	desc, err := decodeText(enc, descDataSplit[0])
	if err != nil {
		return nil, fmt.Errorf("error decoding APIC description text: %v", err)
	}

	var ext string
	switch mimeType {
	case "image/jpeg":
		ext = "jpg"
	case "image/png":
		ext = "png"
	}

	return &Picture{
		Ext:         ext,
		MIMEType:    mimeType,
		Type:        pictureTypes[picType],
		Description: desc,
		Data:        descDataSplit[1],
	}, nil
}
