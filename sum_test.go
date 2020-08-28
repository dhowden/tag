package tag

import (
	"bytes"
	"testing"
)

func TestFuzz(t *testing.T) {
	fuzzIssue73(dataIssue73)
}

var dataIssue73 = []byte{0x49, 0x44, 0x33, 0x03, 0x00, 0x40, 0x00, 0x00, 0x00, 0x0E, 0xDB, 0xDB, 0xDB, 0xDB, 0xDB, 0xDB,
	0xDB, 0x06, 0xFF, 0x54, 0x58, 0x58, 0x00}

func fuzzIssue73(in []byte) {
	r := bytes.NewReader(in)

	Identify(r)

	m, err := ReadFrom(r)
	if err != nil {
		return
	}

	m.Format()
	m.FileType()
	m.Title()
	m.Album()
	m.Artist()
	m.AlbumArtist()
	m.Composer()
	m.Year()
	m.Genre()
	m.Track()
	m.Disc()
	m.Picture()
	m.Lyrics()

	Sum(r)
}
