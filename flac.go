// Copyright 2015, David Howden
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tag

import (
	"errors"
	"io"
	"os"
)

// BlockType is a type which represents an enumeration of valid FLAC blocks
type BlockType byte

// FLAC block types.
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
	flac, err := readString(r, 4)
	if err != nil {
		return nil, err
	}
	if flac != "fLaC" {
		return nil, errors.New("expected 'fLaC'")
	}

	m := &metadataFLAC{
		newMetadataVorbis(),
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
	*metadataVorbis
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

func (m *metadataFLAC) FileType() FileType {
	return FLAC
}
