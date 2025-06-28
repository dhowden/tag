// Copyright 2015, David Howden
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tag

import (
	"errors"
	"io"
	"time"
)

// ReadDSFMeta reads DSF metadata from the io.ReadSeeker, returning the resulting
// metadata in a Metadata implementation, or non-nil error if there was a problem.
// samples: http://www.2l.no/hires/index.html
func ReadDSFMeta(r io.ReadSeeker) (Metadata, error) {
	dsd, err := readString(r, 4)
	if err != nil {
		return nil, err
	}
	if dsd != "DSD " {
		return nil, errors.New("expected 'DSD '")
	}

	_, err = r.Seek(int64(16), io.SeekCurrent)
	if err != nil {
		return nil, err
	}

	id3Pointer, err := readUint64LittleEndian(r)
	if err != nil {
		return nil, err
	}

	_, err = r.Seek(int64(28), io.SeekCurrent)
	if err != nil {
		return nil, err
	}

	sampleRate, err := readUint32LittleEndian(r)
	if err != nil {
		return nil, err
	}

	_, err = r.Seek(int64(4), io.SeekCurrent)
	if err != nil {
		return nil, err
	}

	sampleNum, err := readUint64LittleEndian(r)
	if err != nil {
		return nil, err
	}

	duration := time.Second * (time.Duration(sampleNum) / time.Duration(sampleRate))

	_, err = r.Seek(int64(id3Pointer), io.SeekStart)
	if err != nil {
		return nil, err
	}

	id3, err := ReadID3v2Tags(r)
	if err != nil {
		return nil, err
	}

	return metadataDSF{
		metadataID3v2: id3,
		duration:      duration,
	}, nil
}

type metadataDSF struct {
	*metadataID3v2
	duration time.Duration
}

func (m metadataDSF) FileType() FileType {
	return DSF
}

func (m metadataDSF) Duration() time.Duration {
	return m.duration
}
