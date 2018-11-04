// Copyright 2015, David Howden
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tag

import (
	"os"
	"testing"
)

type testMetadata struct {
	Album       string
	AlbumArtist string
	Artist      string
	Comment     string
	Composer    string
	Disc        int
	DiscTotal   int
	Genre       string
	Lyrics      string
	Title       string
	Track       int
	TrackTotal  int
	Year        int
}

var emptyMetadata = testMetadata{}
var fullMetadata = testMetadata{
	Album:       "Test Album",
	AlbumArtist: "Test AlbumArtist",
	Artist:      "Test Artist",
	Composer:    "Test Composer",
	Disc:        2,
	DiscTotal:   0,
	Genre:       "Jazz",
	Lyrics:      "",
	Title:       "Test Title",
	Track:       3,
	TrackTotal:  6,
	Year:        2000,
	Comment:     "Test Comment",
}
var mp3id3v11Metadata = testMetadata{
	Album:   "Test Album",
	Artist:  "Test Artist",
	Genre:   "Jazz",
	Lyrics:  "",
	Title:   "Test Title",
	Track:   3,
	Year:    2000,
	Comment: "Test Comment",
}

func TestReadFrom(t *testing.T) {
	testdata := map[string]testMetadata{
		"with_tags/sample.flac":       fullMetadata,
		"with_tags/sample.id3v11.mp3": mp3id3v11Metadata,
		"with_tags/sample.id3v22.mp3": fullMetadata,
		"with_tags/sample.id3v23.mp3": fullMetadata,
		"with_tags/sample.id3v24.mp3": fullMetadata,
		"with_tags/sample.m4a":        fullMetadata,
		"with_tags/sample.mp4":        fullMetadata,
		"with_tags/sample.ogg":        fullMetadata,
		"with_tags/sample.dsf":        fullMetadata,
		"without_tags/sample.flac":    emptyMetadata,
		"without_tags/sample.m4a":     emptyMetadata,
		"without_tags/sample.mp3":     emptyMetadata,
		"without_tags/sample.mp4":     emptyMetadata,
		"without_tags/sample.ogg":     emptyMetadata,
	}

	for path, metadata := range testdata {
		if err := test(t, path, metadata); err != nil {

			// mp3 id3v11 returns an err if it doesn't find any tags
			if err != ErrNoTagsFound && path != "without_tags/sample.mp3" {
				t.Error(err)
			}

		}
	}
}

func test(t *testing.T, path string, metadata testMetadata) error {
	t.Log("testing " + path)
	f, err := os.Open("testdata/" + path)
	if err != nil {
		return err
	}
	defer f.Close()

	m, err := ReadFrom(f)
	if err != nil {
		return err
	}
	compareMetadata(t, m, metadata)
	return nil
}

func compareMetadata(t *testing.T, m Metadata, tt testMetadata) {
	testValue(t, tt.Album, m.Album())
	testValue(t, tt.AlbumArtist, m.AlbumArtist())
	testValue(t, tt.Artist, m.Artist())
	testValue(t, tt.Composer, m.Composer())
	testValue(t, tt.Genre, m.Genre())
	testValue(t, tt.Lyrics, m.Lyrics())
	testValue(t, tt.Title, m.Title())
	testValue(t, tt.Year, m.Year())
	testValue(t, tt.Comment, m.Comment())

	disc, discTotal := m.Disc()
	testValue(t, tt.Disc, disc)
	testValue(t, tt.DiscTotal, discTotal)

	track, trackTotal := m.Track()
	testValue(t, tt.Track, track)
	testValue(t, tt.TrackTotal, trackTotal)
}

func testValue(t *testing.T, expected interface{}, found interface{}) {
	if expected != found {
		t.Errorf("expected '%v', found '%v'", expected, found)
	}
}
