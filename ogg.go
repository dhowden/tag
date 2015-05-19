// Copyright 2015, David Howden
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tag

import (
	"errors"
	"io"
	"os"
)

// ReadOGGTags reads OGG metadata from the io.ReadSeeker, returning the resulting
// metadata in a Metadata implementation, or non-nil error if there was a problem.
// TODO: Needs a more generic return type than "metadataFLAC" and the "FLAC" format is not as obvious as "Vorbis comment"
func ReadOGGTags(r io.ReadSeeker) (Metadata, error) {
	_, err := r.Seek(0, os.SEEK_SET)
	if err != nil {
		return nil, err
	}

	oggs, err := readString(r, 4)
	if err != nil {
		return nil, err
	}
	if oggs != "OggS" {
		return nil, errors.New("expected 'OggS'")
	}

	_, err = r.Seek(22, os.SEEK_CUR)
	if err != nil {
		return nil, err
	}

	nS, err := readInt(r, 1)
	if err != nil {
		return nil, err
	}

	_, err = r.Seek(int64(nS), os.SEEK_CUR)
	if err != nil {
		return nil, err
	}

	idComment, err := readInt(r, 1)
	if err != nil {
		return nil, err
	}
	if idComment != 1 {
		return nil, errors.New("expected 'vorbis' identification type 1")
	}

	_, err = r.Seek(29, os.SEEK_CUR)
	if err != nil {
		return nil, err
	}

	oggs, err = readString(r, 4)
	if err != nil {
		return nil, err
	}
	if oggs != "OggS" {
		return nil, errors.New("expected 'OggS'")
	}

	_, err = r.Seek(22, os.SEEK_CUR)
	if err != nil {
		return nil, err
	}

	nS, err = readInt(r, 1)
	if err != nil {
		return nil, err
	}

	_, err = r.Seek(int64(nS), os.SEEK_CUR)
	if err != nil {
		return nil, err
	}

	typeComment, err := readInt(r, 1)
	if err != nil {
		return nil, err
	}
	if typeComment != 3 {
		return nil, errors.New("expected 'vorbis' comment type 3")
	}

	_, err = r.Seek(6, os.SEEK_CUR)
	if err != nil {
		return nil, err
	}

	m := &metadataFLAC{
		c: make(map[string]string),
	}

	err = m.readVorbisComment(r)

	return m, err
}
