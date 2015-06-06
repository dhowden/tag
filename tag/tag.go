// Copyright 2015, David Howden
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
The tag tool reads metadata from media files (as supported by the tag library).
*/
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/dhowden/tag"
)

var raw bool

var usage = func() {
	fmt.Fprintf(os.Stderr, "usage: %s [optional flags] filename\n", os.Args[0])
	flag.PrintDefaults()
}

func init() {
	flag.BoolVar(&raw, "raw", false, "show raw tag data")

	flag.Usage = usage
}

func main() {
	flag.Parse()

	if flag.NArg() != 1 {
		usage()
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
