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
}
var mp3id3v11Metadata = testMetadata{
	Album:  "Test Album",
	Artist: "Test Artist",
	Genre:  "Jazz",
	Lyrics: "",
	Title:  "Test Title",
	Track:  3,
	Year:   2000,
}

type testData struct {
	testMetadata
	FileType
	Format
}

func TestReadFrom(t *testing.T) {
	testdata := map[string]testData{
		"with_tags/sample.flac":       {fullMetadata, FLAC, VORBIS},
		"with_tags/sample.id3v11.mp3": {mp3id3v11Metadata, MP3, ID3v1},
		// TODO: Convert sample.id3v22.mp3 file to ID3v2.2 tag format
		"with_tags/sample.id3v22.mp3": {fullMetadata, MP3, ID3v2_3},
		"with_tags/sample.id3v23.mp3": {fullMetadata, MP3, ID3v2_3},
		"with_tags/sample.id3v24.mp3": {fullMetadata, MP3, ID3v2_4},
		// TODO: Detect correct file type
		"with_tags/sample.m4a": {fullMetadata, UnknownFileType, MP4},
		// TODO: Detect correct file type
		"with_tags/sample.mp4": {fullMetadata, UnknownFileType, MP4},
		"with_tags/sample.ogg": {fullMetadata, OGG, VORBIS},

		"without_tags/sample.flac": {emptyMetadata, FLAC, VORBIS},
		// TODO: Detect correct file type
		"without_tags/sample.m4a": {emptyMetadata, UnknownFileType, MP4},
		"without_tags/sample.mp3": {emptyMetadata, MP3, UnknownFormat},
		// TODO: Detect correct file type
		"without_tags/sample.mp4": {emptyMetadata, UnknownFileType, MP4},
		"without_tags/sample.ogg": {emptyMetadata, OGG, VORBIS},
	}

	for path, data := range testdata {
		if err := test(t, path, data); err != nil {

			// mp3 id3v11 returns an err if it doesn't find any tags
			if err != ErrNoTagsFound && path != "without_tags/sample.mp3" {
				t.Error(err)
			}

		}
	}
}

func test(t *testing.T, path string, data testData) error {
	t.Log("testing '" + path + "'")
	f, err := os.Open("testdata/" + path)
	if err != nil {
		return err
	}
	defer f.Close()

	m, err := ReadFrom(f)
	if err != nil {
		return err
	}
	compareMetadata(t, m, data)
	return nil
}

func compareMetadata(t *testing.T, m Metadata, tt testData) {
	testValue(t, tt.Album, m.Album())
	testValue(t, tt.AlbumArtist, m.AlbumArtist())
	testValue(t, tt.Artist, m.Artist())
	testValue(t, tt.Composer, m.Composer())
	testValue(t, tt.Genre, m.Genre())
	testValue(t, tt.Lyrics, m.Lyrics())
	testValue(t, tt.Title, m.Title())
	testValue(t, tt.Year, m.Year())

	disc, discTotal := m.Disc()
	testValue(t, tt.Disc, disc)
	testValue(t, tt.DiscTotal, discTotal)

	track, trackTotal := m.Track()
	testValue(t, tt.Track, track)
	testValue(t, tt.TrackTotal, trackTotal)

	testValue(t, tt.Format, m.Format())
	testValue(t, tt.FileType, m.FileType())
}

func testValue(t *testing.T, expected interface{}, found interface{}) {
	if expected != found {
		t.Errorf("expected '%v', found '%v'", expected, found)
	}
}
