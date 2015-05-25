// Copyright 2015, David Howden
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tag

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
)

var atomTypes = map[int]string{
	0:  "uint8",
	1:  "text",
	13: "jpeg",
	14: "png",
	21: "uint8",
}

var atoms = atomNames(map[string]string{
	"\xa9alb": "album",
	"\xa9art": "artist",
	"\xa9ART": "artist",
	"aART":    "album_artist",
	"\xa9day": "year",
	"\xa9nam": "title",
	"\xa9gen": "genre",
	"trkn":    "track",
	"\xa9wrt": "composer",
	"\xa9too": "encoder",
	"cprt":    "copyright",
	"covr":    "picture",
	"\xa9grp": "grouping",
	"keyw":    "keyword",
	"\xa9lyr": "lyrics",
	"\xa9cmt": "comment",
	"tmpo":    "tempo",
	"cpil":    "compilation",
	"disk":    "disc",
})

type atomNames map[string]string

func (f atomNames) Name(n string) []string {
	res := make([]string, 1)
	for k, v := range f {
		if v == n {
			res = append(res, k)
		}
	}
	return res
}

// metadataMP4 is the implementation of Metadata for MP4 tag (atom) data.
type metadataMP4 map[string]interface{}

// ReadAtoms reads MP4 metadata atoms from the io.ReadSeeker into a Metadata, returning
// non-nil error if there was a problem.
func ReadAtoms(r io.ReadSeeker) (Metadata, error) {
	_, err := r.Seek(0, os.SEEK_SET)
	if err != nil {
		return nil, err
	}
	m := make(metadataMP4)
	err = m.readAtoms(r)
	return m, err
}

func readCustomAtom(r io.ReadSeeker, size uint32) (string, uint32, error) {
	var datasize uint32
	var datapos int64
	var name string
	var mean string

	for size > 8 {
		var subsize uint32
		err := binary.Read(r, binary.BigEndian, &subsize)
		if err != nil {
			return "----", size - 4, err
		}
		subname, err := readString(r, 4)
		if err != nil {
			return "----", size - 8, err
		}
		b, err := readBytes(r, int(subsize-8))
		if err != nil {
			return "----", size - subsize, err
		}
		// Remove the size of the atom from the size counter
		size -= subsize

		switch string(subname) {
		case "mean":
			mean = string(b[4:])
		case "name":
			name = string(b[4:])
		case "data":
			datapos, err = r.Seek(0, os.SEEK_CUR)
			if err != nil {
				return "----", size, err
			}
			datasize = subsize
			datapos -= int64(subsize)
		}
	}
	// there should remain only the header size
	if size != 8 {
		return "----", size, errors.New("---- atom out of bound")
	}
	if mean == "com.apple.iTunes" && datasize != 0 && name != "" {
		// we jump just before the data subatom
		_, err := r.Seek(datapos, os.SEEK_SET)
		if err != nil {
			return "----", size, err
		}
		return name, datasize + 8, nil
	}
	return "----", size, nil
}

func (m metadataMP4) readAtoms(r io.ReadSeeker) error {
	for {
		var size uint32
		ok := false
		err := binary.Read(r, binary.BigEndian, &size)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		name, err := readString(r, 4)
		if err != nil {
			return err
		}

		switch name {
		case "meta":
			// next_item_id (int32)
			_, err := readBytes(r, 4)
			if err != nil {
				return err
			}
			fallthrough
		case "moov", "udta", "ilst":
			return m.readAtoms(r)
		case "free":
			_, err := r.Seek(int64(size-8), os.SEEK_CUR)
			if err != nil {
				return err
			}
			continue
		case "mdat": // skip the data, the metadata can be at the end
			_, err := r.Seek(int64(size-8), os.SEEK_CUR)
			if err != nil {
				return err
			}
			continue
		case "----":
			// Generic atom.
			// Should have 3 sub atoms : mean, name and data.
			// We check that mean=="com.apple.iTunes" and we use the subname as
			// the name, and move to the data atom.
			// If anything goes wrong, we jump at the end of the "----" atom.
			name, size, err = readCustomAtom(r, size)
			if err != nil {
				return err
			}
			ok = (name != "----")
		default:
			_, ok = atoms[name]
		}

		b, err := readBytes(r, int(size-8))
		if err != nil {
			return err
		}

		// At this point, we allow all known atoms and the valid "----" atoms
		if !ok {
			continue
		}

		// 16: name + size + "data" + size (4 bytes each), have already read 8
		b = b[8:]
		class := getInt(b[1:4])
		contentType, ok := atomTypes[class]
		if !ok {
			return fmt.Errorf("invalid content type: %v", class)
		}

		b = b[8:]
		switch name {
		case "trkn", "disk":
			m[name] = int(b[3])
			m[name+"_count"] = int(b[5])
		default:
			var data interface{}
			// 4: atom version (1 byte) + atom flags (3 bytes)
			// 4: NULL (usually locale indicator)
			switch contentType {
			case "text":
				data = string(b)

			case "uint8":
				data = getInt(b[:1])

			case "jpeg", "png":
				data = &Picture{
					Ext:      contentType,
					MIMEType: "image/" + contentType,
					Data:     b,
				}
			}
			m[name] = data
		}
	}
}

func (metadataMP4) Format() Format     { return MP4 }
func (metadataMP4) FileType() FileType { return AAC }

func (m metadataMP4) Raw() map[string]interface{} { return m }

func (m metadataMP4) getString(n []string) string {
	for _, k := range n {
		if x, ok := m[k]; ok {
			return x.(string)
		}
	}
	return ""
}

func (m metadataMP4) getInt(n []string) int {
	for _, k := range n {
		if x, ok := m[k]; ok {
			return x.(int)
		}
	}
	return 0
}

func (m metadataMP4) Title() string {
	return m.getString(atoms.Name("title"))
}

func (m metadataMP4) Artist() string {
	return m.getString(atoms.Name("artist"))
}

func (m metadataMP4) Album() string {
	return m.getString(atoms.Name("album"))
}

func (m metadataMP4) AlbumArtist() string {
	return m.getString(atoms.Name("album_artist"))
}

func (m metadataMP4) Composer() string {
	return m.getString(atoms.Name("composer"))
}

func (m metadataMP4) Genre() string {
	return m.getString(atoms.Name("genre"))
}

func (m metadataMP4) Year() int {
	date := m.getString(atoms.Name("year"))
	if len(date) >= 4 {
		year, _ := strconv.Atoi(date[:4])
		return year
	}
	return 0
}

func (m metadataMP4) Track() (int, int) {
	x := m.getInt([]string{"trkn"})
	if n, ok := m["trkn_count"]; ok {
		return x, n.(int)
	}
	return x, 0
}

func (m metadataMP4) Disc() (int, int) {
	x := m.getInt([]string{"disk"})
	if n, ok := m["disk_count"]; ok {
		return x, n.(int)
	}
	return x, 0
}

func (m metadataMP4) Lyrics() string {
	t, ok := m["\xa9lyr"]
	if !ok {
		return ""
	}
	return t.(string)
}

func (m metadataMP4) Picture() *Picture {
	v, ok := m["covr"]
	if !ok {
		return nil
	}
	return v.(*Picture)
}
