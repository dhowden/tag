// Copyright 2015, David Howden
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tag

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// BlockType is a type which represents an enumeration of valid FLAC blocks
type BlockType byte

const (
	StreamInfoBlock    BlockType = 0
	PaddingBlock                 = 1
	ApplicationBlock             = 2
	SeektableBlock               = 3
	VorbisCommentBlock           = 4 // Supported
	CueSheetBlock                = 5
	PictureBlock                 = 6 // Supported
)

// ReadFLACTags reads FLAC metadata from the io.ReadSeeker, returning the resulting
// metadata in a Metadata implementation, or non-nil error if there was a problem.
func ReadFLACTags(r io.ReadSeeker) (Metadata, error) {
	_, err := r.Seek(0, os.SEEK_SET)
	if err != nil {
		return nil, err
	}

	flac, err := readString(r, 4)
	if err != nil {
		return nil, err
	}
	if flac != "fLaC" {
		return nil, errors.New("expected 'fLaC'")
	}

	m := &metadataFLAC{
		c: make(map[string]string),
	}

	for {
		last, err := m.readFLACMetadataBlock(r)
		if err != nil {
			return nil, err
		}

		if last {
			break
		}
	}
	return m, nil
}

type metadataFLAC struct {
	c map[string]string // the vorbis comments
	p *Picture
}

func (m *metadataFLAC) readFLACMetadataBlock(r io.ReadSeeker) (last bool, err error) {
	blockHeader, err := readBytes(r, 1)
	if err != nil {
		return
	}

	if getBit(blockHeader[0], 7) {
		blockHeader[0] ^= (1 << 7)
		last = true
	}

	blockLen, err := readInt(r, 3)
	if err != nil {
		return
	}

	switch BlockType(blockHeader[0]) {
	case VorbisCommentBlock:
		err = m.readVorbisComment(r)

	case PictureBlock:
		err = m.readPictureBlock(r)

	default:
		_, err = r.Seek(int64(blockLen), os.SEEK_CUR)
	}
	return
}

func (m *metadataFLAC) readVorbisComment(r io.Reader) error {
	vendorLen, err := readInt32LittleEndian(r)
	if err != nil {
		return err
	}

	vendor, err := readString(r, vendorLen)
	if err != nil {
		return err
	}
	m.c["vendor"] = vendor

	commentsLen, err := readInt32LittleEndian(r)
	if err != nil {
		return err
	}

	for i := 0; i < commentsLen; i++ {
		l, err := readInt32LittleEndian(r)
		if err != nil {
			return err
		}
		s, err := readString(r, l)
		if err != nil {
			return err
		}
		k, v, err := parseComment(s)
		if err != nil {
			return err
		}
		m.c[strings.ToLower(k)] = v
	}
	return nil
}

func (m *metadataFLAC) readPictureBlock(r io.Reader) error {
	b, err := readInt(r, 4)
	if err != nil {
		return err
	}
	pictureType, ok := pictureTypes[byte(b)]
	if !ok {
		return fmt.Errorf("invalid picture type: %v", b)
	}
	mimeLen, err := readInt(r, 4)
	if err != nil {
		return err
	}
	mime, err := readString(r, mimeLen)
	if err != nil {
		return err
	}

	ext := ""
	switch mime {
	case "image/jpeg":
		ext = "jpg"
	case "image/png":
		ext = "png"
	case "image/gif":
		ext = "gif"
	}

	descLen, err := readInt(r, 4)
	if err != nil {
		return err
	}
	desc, err := readString(r, descLen)
	if err != nil {
		return err
	}

	// We skip width <32>, height <32>, colorDepth <32>, coloresUsed <32>
	_, err = readInt(r, 4) // width
	if err != nil {
		return err
	}
	_, err = readInt(r, 4) // height
	if err != nil {
		return err
	}
	_, err = readInt(r, 4) // color depth
	if err != nil {
		return err
	}
	_, err = readInt(r, 4) // colors used
	if err != nil {
		return err
	}

	dataLen, err := readInt(r, 4)
	if err != nil {
		return err
	}
	data := make([]byte, dataLen)
	_, err = io.ReadFull(r, data)
	if err != nil {
		return err
	}

	m.p = &Picture{
		Ext:         ext,
		MIMEType:    mime,
		Type:        pictureType,
		Description: desc,
		Data:        data,
	}
	return nil
}

func parseComment(c string) (k, v string, err error) {
	kv := strings.SplitN(c, "=", 2)
	if len(kv) != 2 {
		err = errors.New("vorbis comment must contain '='")
		return
	}
	k = kv[0]
	v = kv[1]
	return
}

func (m *metadataFLAC) Format() Format {
	return FLAC
}

func (m *metadataFLAC) Raw() map[string]interface{} {
	raw := make(map[string]interface{}, len(m.c))
	for k, v := range m.c {
		raw[k] = v
	}
	return raw
}

func (m *metadataFLAC) Title() string {
	return m.c["title"]
}

func (m *metadataFLAC) Artist() string {
	// PERFORMER
	// The artist(s) who performed the work. In classical music this would be the
	// conductor, orchestra, soloists. In an audio book it would be the actor who
	// did the reading. In popular music this is typically the same as the ARTIST
	// and is omitted.
	if m.c["performer"] != "" {
		return m.c["performer"]
	}
	return m.c["artist"]
}

func (m *metadataFLAC) Album() string {
	return m.c["album"]
}

func (m *metadataFLAC) AlbumArtist() string {
	// This field isn't included in the standard.
	return ""
}

func (m *metadataFLAC) Composer() string {
	// ARTIST
	// The artist generally considered responsible for the work. In popular music
	// this is usually the performing band or singer. For classical music it would
	// be the composer. For an audio book it would be the author of the original text.
	if m.c["composer"] != "" {
		return m.c["composer"]
	}
	if m.c["performer"] == "" {
		return ""
	}
	return m.c["artist"]
}

func (m *metadataFLAC) Genre() string {
	return m.c["genre"]
}

func (m *metadataFLAC) Year() int {
	// FIXME: try to parse the date in m.c["date"] to extract this
	return 0
}

func (m *metadataFLAC) Track() (int, int) {
	x, _ := strconv.Atoi(m.c["tracknumber"])
	// https://wiki.xiph.org/Field_names
	n, _ := strconv.Atoi(m.c["tracktotal"])
	return x, n
}

func (m *metadataFLAC) Disc() (int, int) {
	// https://wiki.xiph.org/Field_names
	x, _ := strconv.Atoi(m.c["discnumber"])
	n, _ := strconv.Atoi(m.c["disctotal"])
	return x, n
}

func (m *metadataFLAC) Picture() *Picture {
	return m.p
}
