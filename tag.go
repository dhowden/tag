// Copyright 2015, David Howden
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package tag provides basic MP3 (ID3v1,2.{2,3,4}) and MP4 metadata parsing.
package tag

import (
	"errors"
	"io"
)

// ErrNoTagsFound is the error returned by ReadFrom when the metadata format
// cannot be identified.
var ErrNoTagsFound = errors.New("no tags found")

// ReadFrom parses audio file metadata tags (currently supports ID3v1,2.{2,3,4} and MP4).
// This method attempts to determine the format of the data provided by the io.ReadSeeker,
// and then chooses ReadAtoms (MP4), ReadID3v2Tags (ID3v2.{2,3,4}) or ReadID3v1Tags as
// appropriate.  Returns non-nil error if the format of the given data could not be determined,
// or if there was a problem parsing the data.
func ReadFrom(r io.ReadSeeker) (Metadata, error) {
	b, err := readBytes(r, 11)
	if err != nil {
		return nil, err
	}

	switch {
	case string(b[0:4]) == "fLaC":
		return ReadFLACTags(r)

	case string(b[4:11]) == "ftypM4A":
		return ReadAtoms(r)

	case string(b[0:3]) == "ID3":
		return ReadID3v2Tags(r)
	}

	m, err := ReadID3v1Tags(r)
	if err != nil {
		if err == ErrNotID3v1 {
			err = ErrNoTagsFound
		}
		return nil, err
	}
	return m, nil
}

// Format is an enumeration of metadata types supported by this package.
type Format string

const (
	ID3v1   Format = "ID3v1"   // ID3v1 tag format.
	ID3v2_2        = "ID3v2.2" // ID3v2.2 tag format.
	ID3v2_3        = "ID3v2.3" // ID3v2.3 tag format (most common).
	ID3v2_4        = "ID3v2.4" // ID3v2.4 tag format.
	MP4            = "MP4"     // MP4 tag (atom) format.
	FLAC           = "FLAC"    // FLAC (Vorbis Comment) tag format.
)

// Metadata is an interface which is used to describe metadata retrieved by this package.
type Metadata interface {
	// Format returns the metadata Format used to encode the data.
	Format() Format

	// Title returns the title of the track.
	Title() string

	// Album returns the album name of the track.
	Album() string

	// Artist returns the artist name of the track.
	Artist() string

	// AlbumArtist returns the album artist name of the track.
	AlbumArtist() string

	// Composer returns the composer of the track.
	Composer() string

	// Year returns the year of the track.
	Year() int

	// Track returns the track number and total tracks, or zero values if unavailable.
	Track() (int, int)

	// Disc returns the disc number and total discs, or zero values if unavailable.
	Disc() (int, int)

	// Picture returns a picture, or nil if not available.
	Picture() *Picture

	// Lyrics returns the lyrics, or an empty string if unavailable.
	Lyrics() string

	// Raw returns the raw mapping of retrieved tag names and associated values.
	// NB: tag/atom names are not standardised between formats.
	Raw() map[string]interface{}
}
