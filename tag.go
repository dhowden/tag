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

	case string(b[0:4]) == "OggS":
		return ReadOGGTags(r)

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

// Extract the tag created with MusicBrainz Picard.
// You can use them with the MusicBrainz and LastFM API
// See https://picard.musicbrainz.org/docs/mappings/ for the mappings
func MusicBrainz(m *Metadata) (mb *MBInfo) {
	txxx := "TXXX"
	ufid := "UFID"
	raw := (*m).Raw()
	mb = new(MBInfo)

	for k, v := range raw {
		var frame, value string
		switch (*m).Format() {
		case ID3v2_2:
			txxx = "TXX"
			ufid = "UFI"
			fallthrough
		case ID3v2_3, ID3v2_4:
			switch k[0:len(txxx)] {
			case txxx:
				if str, ok := v.(*Comm); ok {
					frame = str.Description
					value = str.Text
				}
			case ufid:
				if str, ok := v.(*UFID); ok {
					if str.Provider == "http://musicbrainz.org" {
						value = string(str.Identifier)
						frame = "MusicBrainz Track Id"
					}
				}
			}
		case MP4, VORBIS, FLAC:
			if str, ok := v.(string); ok {
				frame = k
				value = str
			}
		}

		switch frame {
		case "Acoustid Id", "acoustid_id":
			mb.Acoustid = value
		case "MusicBrainz Album Artist Id", "musicbrainz_albumartistid":
			mb.AlbumArtist = value
		case "MusicBrainz Artist Id", "musicbrainz_artistid":
			mb.Artist = value
		case "MusicBrainz Release Group Id", "musicbrainz_releasegroupid":
			mb.ReleaseGroup = value
		case "MusicBrainz Album Id", "musicbrainz_albumid":
			mb.Album = value
		case "MusicBrainz Track Id", "musicbrainz_trackid":
			mb.Track = value
		}
	}
	return
}

type MBInfo struct {
	AlbumArtist  string `musicbrainz:"musicbrainz_albumartistid"`
	Album        string `musicbrainz:"musicbrainz_albumid"`
	Artist       string `musicbrainz:"musicbrainz_artistid"`
	ReleaseGroup string `musicbrainz:"musicbrainz_releasegroupid"`
	Track        string `musicbrainz:"musicbrainz_recordingid"`
	Acoustid     string `musicbrainz:"acoustid_id"`
}

// Format is an enumeration of metadata types supported by this package.
type Format string

const (
	ID3v1   Format = "ID3v1"   // ID3v1 tag format.
	ID3v2_2        = "ID3v2.2" // ID3v2.2 tag format.
	ID3v2_3        = "ID3v2.3" // ID3v2.3 tag format (most common).
	ID3v2_4        = "ID3v2.4" // ID3v2.4 tag format.
	MP4            = "MP4"     // MP4 tag (atom) format.
	VORBIS         = "VORBIS"  // Vorbis Comment tag format.
)

// FileType is an enumeration of the audio file types supported by this package, in particular
// there are audio file types which share metadata formats, and this type is used to distinguish
// between them.
type FileType string

const (
	MP3  FileType = "MP3"  // MP3 file
	AAC           = "AAC"  // M4A file (MP4)
	ALAC          = "ALAC" // Apple Lossless file FIXME: actually detect this
	FLAC          = "FLAC" // FLAC file
	OGG           = "OGG"  // OGG file
)

// Metadata is an interface which is used to describe metadata retrieved by this package.
type Metadata interface {
	// Format returns the metadata Format used to encode the data.
	Format() Format

	// FileType returns the file type of the audio file.
	FileType() FileType

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

	// Genre returns the genre of the track.
	Genre() string

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
