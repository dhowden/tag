// Package mbz extracts MusicBrainz Picard-specific tags from general tag metadata.
// See https://picard.musicbrainz.org/docs/mappings/ for more information.
package mbz

import (
	"strings"

	"github.com/dhowden/tag"
)

// Supported MusicBrainz tag names.
const (
	AcoustID          = "acoustid_id"
	AcoustFingerprint = "acoustid_fingerprint"
	Album             = "musicbrainz_albumid"
	AlbumArtist       = "musicbrainz_albumartistid"
	Artist            = "musicbrainz_artistid"
	Disc              = "musicbrainz_discid"
	Recording         = "musicbrainz_recordingid"
	ReleaseGroup      = "musicbrainz_releasegroupid"
	Track             = "musicbrainz_trackid"
	TRM               = "musicbrainz_trmid"
)

// UFIDProviderURL is the URL that we match inside a UFID tag.
const UFIDProviderURL = "http://musicbrainz.org"

// Mapping between the internal picard tag names and aliases.
var tags = map[string]string{
	AcoustID:          "Acoustid Id",
	AcoustFingerprint: "Acoustid Fingerprint",
	Album:             "MusicBrainz Album Id",
	AlbumArtist:       "MusicBrainz Album Artist Id",
	Artist:            "MusicBrainz Artist Id",
	Disc:              "MusicBrainz Disc Id",
	Recording:         "MusicBrainz Track Id",
	ReleaseGroup:      "MusicBrainz Release Group Id",
	Track:             "MusicBrainz Release Track Id",
	TRM:               "MusicBrainz TRM Id",
}

// Info is a structure which contains MusicBrainz identifier information.
type Info map[string]string

// Get returns the value for the given MusicBrainz tag.
func (i Info) Get(tag string) string {
	return i[tag]
}

// set the MusicBrainz tag to the given value.
func (i Info) set(t, v string) {
	if _, ok := tags[t]; ok {
		i[t] = v
		return
	}

	for k, tt := range tags {
		if tt == t {
			i[k] = v
			return
		}
	}
}

// extractID3 attempts to extract MusicBrainz Picard tags from m.Raw(), where m.Format
// is assumed to be a supported version of ID3.
func extractID3(m tag.Metadata) Info {
	var txxx, ufid string
	switch m.Format() {
	case tag.ID3v2_2:
		txxx, ufid = "TXX", "UFI"
	case tag.ID3v2_3, tag.ID3v2_4:
		txxx, ufid = "TXXX", "UFID"
	}

	i := Info{}
	for k, v := range m.Raw() {
		switch {
		case strings.HasPrefix(k, txxx):
			if str, ok := v.(*tag.Comm); ok {
				i.set(str.Description, str.Text)
			}
		case strings.HasPrefix(k, ufid):
			if id, ok := v.(*tag.UFID); ok {
				if id.Provider == UFIDProviderURL {
					i.set(Recording, string(id.Identifier))
				}
			}
		}
	}
	return i
}

// extractMP4Vorbis attempts to extract MusicBrainz Picard tags from m.Raw(), where m.Format
// is assumed to be MP4 or VORBIS.
func extractMP4Vorbis(m tag.Metadata) Info {
	i := Info{}
	for t, v := range m.Raw() {
		if s, ok := v.(string); ok {
			i.set(t, s)
		}
	}
	return i
}

// Extract tags created by MusicBrainz Picard which can be used with with the MusicBrainz and LastFM APIs.
// See https://picard.musicbrainz.org/docs/mappings/ for more information.
func Extract(m tag.Metadata) Info {
	switch m.Format() {
	case tag.ID3v2_2, tag.ID3v2_3, tag.ID3v2_4:
		return extractID3(m)
	}
	return extractMP4Vorbis(m)
}
