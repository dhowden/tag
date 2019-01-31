// Copyright 2015, David Howden
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tag

import (
	"strconv"
	"strings"
)

type frameNames map[string][2]string

func (f frameNames) Name(s string, fm Format) string {
	l, ok := f[s]
	if !ok {
		return ""
	}

	switch fm {
	case ID3v2_2:
		return l[0]
	case ID3v2_3:
		return l[1]
	case ID3v2_4:
		if s == "year" || s == "date" {
			return "TDRC"
		}
		return l[1]
	}
	return ""
}

var frames = frameNames(map[string][2]string{
	"title":        {"TT2", "TIT2"},
	"artist":       {"TP1", "TPE1"},
	"album":        {"TAL", "TALB"},
	"album_artist": {"TP2", "TPE2"},
	"composer":     {"TCM", "TCOM"},
	"date":         {"TDA", "TDAT"},
	"year":         {"TYE", "TYER"},
	"track":        {"TRK", "TRCK"},
	"disc":         {"TPA", "TPOS"},
	"genre":        {"TCO", "TCON"},
	"picture":      {"PIC", "APIC"},
	"lyrics":       {"", "USLT"},
	"comment":      {"COM", "COMM"},
})

// metadataID3v2 is the implementation of Metadata used for ID3v2 tags.
type metadataID3v2 struct {
	header *id3v2Header
	frames map[string]interface{}
}

func (m metadataID3v2) getString(k string) string {
	v, ok := m.frames[k]
	if !ok {
		return ""
	}
	return v.(string)
}

func (m metadataID3v2) Format() Format              { return m.header.Version }
func (m metadataID3v2) FileType() FileType          { return MP3 }
func (m metadataID3v2) Raw() map[string]interface{} { return m.frames }

func (m metadataID3v2) Title() string {
	return m.getString(frames.Name("title", m.Format()))
}

func (m metadataID3v2) Artist() string {
	return m.getString(frames.Name("artist", m.Format()))
}

func (m metadataID3v2) Album() string {
	return m.getString(frames.Name("album", m.Format()))
}

func (m metadataID3v2) AlbumArtist() string {
	return m.getString(frames.Name("album_artist", m.Format()))
}

func (m metadataID3v2) Composer() string {
	return m.getString(frames.Name("composer", m.Format()))
}

func (m metadataID3v2) Genre() string {
	return id3v2genre(m.getString(frames.Name("genre", m.Format())))
}

func (m metadataID3v2) Date() string {
	date := m.getString(frames.Name("date", m.Format()))
	if "" == date {
		if year := m.Year(); year != 0 {
			return strconv.Itoa(year)
		}
	}
	return date
}

func (m metadataID3v2) Year() int {
	year, _ := strconv.Atoi(m.getString(frames.Name("year", m.Format())))
	return year
}

func parseXofN(s string) (x, n int) {
	xn := strings.Split(s, "/")
	if len(xn) != 2 {
		x, _ = strconv.Atoi(s)
		return x, 0
	}
	x, _ = strconv.Atoi(strings.TrimSpace(xn[0]))
	n, _ = strconv.Atoi(strings.TrimSpace(xn[1]))
	return x, n
}

func (m metadataID3v2) Track() (int, int) {
	return parseXofN(m.getString(frames.Name("track", m.Format())))
}

func (m metadataID3v2) Disc() (int, int) {
	return parseXofN(m.getString(frames.Name("disc", m.Format())))
}

func (m metadataID3v2) Lyrics() string {
	t, ok := m.frames[frames.Name("lyrics", m.Format())]
	if !ok {
		return ""
	}
	return t.(*Comm).Text
}

func (m metadataID3v2) Comment() string {
	t, ok := m.frames[frames.Name("comment", m.Format())]
	if !ok {
		return ""
	}
	// id3v23 has Text, id3v24 has Description
	if t.(*Comm).Description == "" {
		return trimString(t.(*Comm).Text)
	}
	return trimString(t.(*Comm).Description)
}

func (m metadataID3v2) Picture() *Picture {
	v, ok := m.frames[frames.Name("picture", m.Format())]
	if !ok {
		return nil
	}
	return v.(*Picture)
}
