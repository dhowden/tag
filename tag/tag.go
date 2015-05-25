// Copyright 2015, David Howden
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
The tag tool reads metadata from media files (as supported by the tag library).
*/
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/dhowden/tag"
	"io/ioutil"
	"net/http"
	"os"
)

var raw bool
var mb bool

func init() {
	flag.BoolVar(&raw, "raw", false, "show raw tag data")
	flag.BoolVar(&mb, "mb", false, "display MusicBrainz info, if any")
}

func main() {
	flag.Parse()

	if flag.NArg() != 1 {
		fmt.Printf("usage: %v filename\n", os.Args[0])
		return
	}

	f, err := os.Open(flag.Arg(0))
	if err != nil {
		fmt.Printf("error loading file: %v", err)
		return
	}
	defer f.Close()

	m, err := tag.ReadFrom(f)
	if err != nil {
		fmt.Printf("error reading file: %v\n", err)
		return
	}

	printMetadata(m)

	if raw {
		fmt.Println()
		fmt.Println()

		tags := m.Raw()
		for k, v := range tags {
			if _, ok := v.(*tag.Picture); ok {
				fmt.Printf("%#v: %v\n", k, v)
				continue
			}
			fmt.Printf("%#v: %#v\n", k, v)
		}
	}

	if mb {
		mb := tag.MusicBrainz(&m)
		if mb.Artist != "" {
			url := fmt.Sprintf("http://musicbrainz.org/ws/2/artist/%v/?fmt=json&inc=url-rels", mb.Artist)
			geturl(url)
		} else {
			fmt.Println("Didn't find any MusicBrainz Artist Id in the tags")
		}
	}
}

func geturl(url string) {
	response, err := http.Get(url)
	var data interface{}
	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	} else {
		defer response.Body.Close()
		contents, err := ioutil.ReadAll(response.Body)
		if err != nil {
			fmt.Printf("%s", err)
			os.Exit(1)
		}
		if err = json.Unmarshal(contents, &data); err == nil {
			if text, err := json.MarshalIndent(data, "", "     "); err == nil {
				fmt.Print(string(text))
			}
		}
	}
}

func printMetadata(m tag.Metadata) {
	fmt.Printf("Metadata Format: %v\n", m.Format())
	fmt.Printf("File Type: %v\n", m.FileType())

	fmt.Printf(" Title: %v\n", m.Title())
	fmt.Printf(" Album: %v\n", m.Album())
	fmt.Printf(" Artist: %v\n", m.Artist())
	fmt.Printf(" Composer: %v\n", m.Composer())
	fmt.Printf(" Genre: %v\n", m.Genre())
	fmt.Printf(" Year: %v\n", m.Year())

	track, trackCount := m.Track()
	fmt.Printf(" Track: %v of %v\n", track, trackCount)

	disc, discCount := m.Disc()
	fmt.Printf(" Disc: %v of %v\n", disc, discCount)

	fmt.Printf(" Picture: %v\n", m.Picture())
	fmt.Printf(" Lyrics: %v\n", m.Lyrics())
}
