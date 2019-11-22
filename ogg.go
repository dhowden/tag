// Copyright 2015, David Howden
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tag

import (
	"bytes"
	"errors"
	"io"
)

const (
	idType      int = 1
	commentType int = 3
)

// ReadOGGTags reads OGG metadata from the io.ReadSeeker, returning the resulting
// metadata in a Metadata implementation, or non-nil error if there was a problem.
// See http://www.xiph.org/vorbis/doc/Vorbis_I_spec.html
// and http://www.xiph.org/ogg/doc/framing.html for details.
func ReadOGGTags(r io.ReadSeeker) (Metadata, error) {
	oggs, err := readString(r, 4)
	if err != nil {
		return nil, err
	}
	if oggs != "OggS" {
		return nil, errors.New("expected 'OggS'")
	}

	// Skip 22 bytes of Page header to read page_segments length byte at position 26
	// See http://www.xiph.org/ogg/doc/framing.html
	_, err = r.Seek(22, io.SeekCurrent)
	if err != nil {
		return nil, err
	}

	nS, err := readInt(r, 1)
	if err != nil {
		return nil, err
	}

	// Seek and discard the segments
	_, err = r.Seek(int64(nS), io.SeekCurrent)
	if err != nil {
		return nil, err
	}

	// First packet type is identification, type 1
	t, err := readInt(r, 1)
	if err != nil {
		return nil, err
	}
	if t != idType {
		return nil, errors.New("expected 'vorbis' identification type 1")
	}

	// Seek and discard 29 bytes from common and identification header
	// See http://www.xiph.org/vorbis/doc/Vorbis_I_spec.html#x1-610004.2
	_, err = r.Seek(29, io.SeekCurrent)
	if err != nil {
		return nil, err
	}

	// Read comment header packet. May include setup header packet, if it is on the
	// same page. First audio packet is guaranteed to be on the separate page.
	// See https://www.xiph.org/vorbis/doc/Vorbis_I_spec.html#x1-132000A.2
	ch, err := readPackets(r)
	if err != nil {
		return nil, err
	}
	chr := bytes.NewReader(ch)

	// First packet type is comment, type 3
	t, err = readInt(chr, 1)
	if err != nil {
		return nil, err
	}
	if t != commentType {
		return nil, errors.New("expected 'vorbis' comment type 3")
	}

	// Seek and discard 6 bytes from common header
	_, err = chr.Seek(6, io.SeekCurrent)
	if err != nil {
		return nil, err
	}

	m := &metadataOGG{
		newMetadataVorbis(),
	}

	err = m.readVorbisComment(chr)
	return m, err
}

// readPackets reads vorbis header packets from contiguous ogg pages in ReadSeeker.
// The pages are considered contiguous, if the first lacing value in second
// page's segment table continues rather than begins a packet. This is indicated
// by setting header_type_flag 0x1 (continued packet).
// See https://www.xiph.org/ogg/doc/framing.html on packets spanning pages.
func readPackets(r io.ReadSeeker) ([]byte, error) {
	buf := &bytes.Buffer{}

	firstPage := true
	for {
		// Read capture pattern
		oggs, err := readString(r, 4)
		if err != nil {
			return nil, err
		}
		if oggs != "OggS" {
			return nil, errors.New("expected 'OggS'")
		}

		// Read page header
		head, err := readBytes(r, 22)
		if err != nil {
			return nil, err
		}
		headerTypeFlag := head[1]

		continuation := headerTypeFlag&0x1 > 0
		if !(firstPage || continuation) {
			// Rewind to the beginning of the page
			_, err = r.Seek(-26, io.SeekCurrent)
			if err != nil {
				return nil, err
			}
			break
		}
		firstPage = false

		// Read the number of segments
		nS, err := readUint(r, 1)
		if err != nil {
			return nil, err
		}

		// Read segment table
		segments, err := readBytes(r, nS)
		if err != nil {
			return nil, err
		}

		// Calculate remaining page size
		pageSize := 0
		for i := uint(0); i < nS; i++ {
			pageSize += int(segments[i])
		}

		_, err = io.CopyN(buf, r, int64(pageSize))
		if err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

type metadataOGG struct {
	*metadataVorbis
}

func (m *metadataOGG) FileType() FileType {
	return OGG
}
