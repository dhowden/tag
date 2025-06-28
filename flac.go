// Copyright 2015, David Howden
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tag

import (
	"errors"
	"fmt"
	"io"
	"time"
)

// blockType is a type which represents an enumeration of valid FLAC blocks
type blockType byte

// FLAC block types.
const (
	// Padding Block               1
	// Application Block           2
	// Seektable Block             3
	// Cue Sheet Block             5
	streamInfoBlock    blockType = 0
	vorbisCommentBlock blockType = 4
	pictureBlock       blockType = 6
)

// ReadFLACMeta reads FLAC metadata from the io.ReadSeeker, returning the resulting
// metadata in a Metadata implementation, or non-nil error if there was a problem.
func ReadFLACMeta(r io.ReadSeeker) (Metadata, error) {
	flac, err := readString(r, 4)
	if err != nil {
		return nil, err
	}
	if flac != "fLaC" {
		return nil, errors.New("expected 'fLaC'")
	}

	m := &metadataFLAC{
		metadataVorbis: newMetadataVorbis(),
	}

	for {
		last, err := m.readFLACBlock(r)
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
	*metadataVorbis
	duration time.Duration
}

func (m *metadataFLAC) readFLACBlock(r io.ReadSeeker) (last bool, err error) {
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

	switch blockType(blockHeader[0]) {
	case vorbisCommentBlock:
		err = m.readVorbisComment(r)

	case pictureBlock:
		err = m.readPictureBlock(r)

	case streamInfoBlock:
		err = m.readStreamingInfoBlock(r, blockLen)

	default:
		_, err = r.Seek(int64(blockLen), io.SeekCurrent)
	}
	return
}

func (m *metadataFLAC) readStreamingInfoBlock(r io.Reader, len int) error {
	data := make([]byte, len)
	if _, err := r.Read(data); err != nil {
		return err
	}

	sampleRate, err := cutBits(data, 80, 20)
	if err != nil {
		return fmt.Errorf("reading sample rate: %w", err)
	}

	sampleNum, err := cutBits(data, 108, 36)
	if err != nil {
		return fmt.Errorf("reading sample number: %w", err)
	}

	m.duration = time.Second * (time.Duration(sampleNum) / time.Duration(sampleRate))

	return nil
}

func (m *metadataFLAC) FileType() FileType {
	return FLAC
}

func (m *metadataFLAC) Duration() time.Duration {
	return m.duration
}
