// Package mbz extracts MusicBrainz Picard-specific tags from general tag metadata.
// See https://picard.musicbrainz.org/docs/mappings/ for more information.
package mbz

import (
	"strings"

	"github.com/dhowden/tag"
)

// Info is a structure which contains MusicBrainz identifier information.
type Info struct {
	AcoustID     string
	Album        string
	AlbumArtist  string
	Artist       string
	ReleaseGroup string
	Track        string
}

// Supported MusicBrainz tag names
const (
	TagAcoustID     = "acoustid_id"
	TagAlbum        = "musicbrainz_albumid"
	TagAlbumArtist  = "musicbrainz_albumartistid"
	TagArtist       = "musicbrainz_artistid"
	TagReleaseGroup = "musicbrainz_releasegroupid"
	TagTrack        = "musicbrainz_recordingid"
)

// UFIDProviderURL is the URL that we match inside a UFID tag.
const UFIDProviderURL = "http://musicbrainz.org"

// Mapping between the internal picard tag names and aliases.
var tags = map[string]string{
	TagAcoustID:     "Acoustid Id",
	TagAlbum:        "MusicBrainz Album Id",
	TagAlbumArtist:  "MusicBrainz Album Artist Id",
	TagArtist:       "MusicBrainz Artist Id",
	TagReleaseGroup: "MusicBrainz Release Group Id",
	TagTrack:        "MusicBrainz Track Id",
}

func (i *Info) set(t, v string) {
	switch t {
	case TagAcoustID:
		i.AcoustID = v
	case TagAlbum:
		i.Album = v
	case TagAlbumArtist:
		i.AlbumArtist = v
	case TagArtist:
		i.Artist = v
	case TagReleaseGroup:
		i.ReleaseGroup = v
	case TagTrack:
		i.Track = v
	}
}

// Set the MusicBrainz tag to the given value.
func (i *Info) Set(t, v string) {
	if _, ok := tags[t]; ok {
		i.set(t, v)
		return
	}

	for k, tt := range tags {
		if tt == t {
			i.set(k, v)
			return
		}
	}
}

// extractID3 attempts to extract MusicBrainz Picard tags from m.Raw(), where m.Format
// is assumed to be a supported version of ID3.
func extractID3(m tag.Metadata) *Info {
	var txxx, ufid string
	switch m.Format() {
	case tag.ID3v2_2:
		txxx, ufid = "TXX", "UFI"
	case tag.ID3v2_3, tag.ID3v2_4:
		txxx, ufid = "TXXX", "UFID"
	}

	i := &Info{}
	for k, v := range m.Raw() {
		switch {
		case strings.HasPrefix(k, txxx):
			if str, ok := v.(*tag.Comm); ok {
				i.Set(str.Description, str.Text)
			}
		case strings.HasPrefix(k, ufid):
			if id, ok := v.(*tag.UFID); ok {
				if id.Provider == UFIDProviderURL {
					i.Set(TagTrack, string(id.Identifier))
				}
			}
		}
	}
	return i
}

// extractMP4Vorbis attempts to extract MusicBrainz Picard tags from m.Raw(), where m.Format
// is assumed to be MP4 or VORBIS.
func extractMP4Vorbis(m tag.Metadata) *Info {
	i := &Info{}
	for t, v := range m.Raw() {
		if s, ok := v.(string); ok {
			i.Set(t, s)
		}
	}
	return i
}

// Extract tags created by MusicBrainz Picard which can be used with with the MusicBrainz and LastFM APIs.
// See https://picard.musicbrainz.org/docs/mappings/ for more information.
func Extract(m tag.Metadata) *Info {
	switch m.Format() {
	case tag.ID3v2_2, tag.ID3v2_3, tag.ID3v2_4:
		return extractID3(m)
	}
	return extractMP4Vorbis(m)
}
