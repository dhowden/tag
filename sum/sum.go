// Copyright 2015, David Howden
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
The sum tool constructs a checksum of a media file exluding any metadata
(as recognised by the tag library).
*/
package main

import (
	"fmt"
	"os"

	"github.com/dhowden/tag"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Printf("usage: %v filename\n", os.Args[0])
		return
	}

	f, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Printf("error loading file: %v", err)
		os.Exit(1)
	}
	defer f.Close()

	h, err := tag.Sum(f)
	if err != nil {
		fmt.Printf("error constructing checksum: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(h)
}
