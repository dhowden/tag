// Copyright 2015, David Howden
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tag

import (
	"errors"
	"io"
)

// blockType is a type which represents an enumeration of valid FLAC blocks
type blockType byte

// FLAC block types.
const (
	StreamInfoBlock blockType = 0
	// Padding Block               1
	// Application Block           2
	// Seektable Block             3
	// Cue Sheet Block             5
	vorbisCommentBlock blockType = 4
	pictureBlock       blockType = 6
)

// ReadFLACTags reads FLAC metadata from the io.ReadSeeker, returning the resulting
// metadata in a Metadata implementation, or non-nil error if there was a problem.
func ReadFLACTags(r io.ReadSeeker) (Metadata, error) {
	flac, err := readString(r, 4)
	if err != nil {
		return nil, err
	}
	if flac != "fLaC" {
		return nil, errors.New("expected 'fLaC'")
	}

	m := &MetadataFLAC{
		metadataVorbis: newMetadataVorbis(),
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

type MetadataFLAC struct {
	*metadataVorbis

	MiniBlockSize uint16
	MaxBlockSize  uint16
	SampleRate    uint32
	TotalSamples  uint64
	Duration      float64
}

func (m *MetadataFLAC) readFLACMetadataBlock(r io.ReadSeeker) (last bool, err error) {
	blockHeader, err := readBytes(r, 1)
	if err != nil {
		return
	}

	if getBit(blockHeader[0], 7) {
		blockHeader[0] ^= 1 << 7
		last = true
	}

	blockLen, err := readInt(r, 3)
	if err != nil {
		return
	}

	switch blockType(blockHeader[0]) {
	case StreamInfoBlock:
		err = m.readStreamInfo(r, blockLen)
	case vorbisCommentBlock:
		err = m.readVorbisComment(r)

	case pictureBlock:
		err = m.readPictureBlock(r)

	default:
		_, err = r.Seek(int64(blockLen), io.SeekCurrent)
	}
	return
}

func (m *MetadataFLAC) readStreamInfo(r io.ReadSeeker, len int) error {
	data := make([]byte, len)

	if _, err := r.Read(data); err != nil {
		return err
	}

	m.MiniBlockSize = uint16(data[0])<<8 | uint16(data[1])
	m.MaxBlockSize = uint16(data[2])<<8 | uint16(data[3])

	m.SampleRate = (uint32(data[10])<<16 | uint32(data[11])<<8 | uint32(data[12])) >> 4

	m.TotalSamples = uint64(data[13])<<32 | uint64(data[14])<<24 | uint64(data[15])<<16 | uint64(data[16])<<8 | uint64(data[17])

	m.TotalSamples ^= m.TotalSamples >> 36 << 36

	m.Duration = float64(m.TotalSamples) / float64(m.SampleRate)

	return nil
}

func (m *MetadataFLAC) FileType() FileType {
	return FLAC
}
