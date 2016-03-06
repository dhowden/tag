// Copyright 2015, David Howden
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
The check tool performs tag lookups on full music collections (iTunes or directory tree of files).
*/
package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/dhowden/itl"
	"github.com/dhowden/tag"
)

var itlXML, path string
var sum bool

func init() {
	flag.StringVar(&itlXML, "itlXML", "", "iTunes Library Path")
	flag.StringVar(&path, "path", "", "path to directory containing audio files")
	flag.BoolVar(&sum, "sum", false, "compute the checksum of the audio file (doesn't work for .flac or .ogg yet)")
}

func decodeLocation(l string) (string, error) {
	u, err := url.ParseRequestURI(l)
	if err != nil {
		return "", err
	}
	// Annoyingly this doesn't replace &#38; (&)
	path := strings.Replace(u.Path, "&#38;", "&", -1)
	return path, nil
}

func main() {
	flag.Parse()

	if itlXML == "" && path == "" || itlXML != "" && path != "" {
		fmt.Println("you must specify one of -itlXML or -path")
		flag.Usage()
		os.Exit(1)
	}

	var paths <-chan string
	if itlXML != "" {
		var err error
		paths, err = walkLibrary(itlXML)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	if path != "" {
		paths = walkPath(path)
	}

	p := &processor{
		decodingErrors: make(map[string]int),
		hashErrors:     make(map[string]int),
		hashes:         make(map[string]int),
	}

	done := make(chan bool)
	go func() {
		p.do(paths)
		fmt.Println(p)
		close(done)
	}()
	<-done
}

func walkPath(root string) <-chan string {
	ch := make(chan string)
	fn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		ch <- path
		return nil
	}

	go func() {
		err := filepath.Walk(root, fn)
		if err != nil {
			fmt.Println(err)
		}
		close(ch)
	}()
	return ch
}

func walkLibrary(path string) (<-chan string, error) {
	f, err := os.Open(itlXML)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	l, err := itl.ReadFromXML(f)
	if err != nil {
		return nil, err
	}

	paths := make(chan string)
	go func() {
		for _, t := range l.Tracks {
			loc, err := decodeLocation(t.Location)
			if err != nil {
				fmt.Println(err)
				continue
			}
			paths <- loc
		}
		close(paths)
	}()
	return paths, nil
}

type processor struct {
	decodingErrors map[string]int
	hashErrors     map[string]int
	hashes         map[string]int
}

func (p *processor) String() string {
	result := ""
	for k, v := range p.decodingErrors {
		result += fmt.Sprintf("%v : %v\n", k, v)
	}

	for k, v := range p.hashErrors {
		result += fmt.Sprintf("%v : %v\n", k, v)
	}

	for k, v := range p.hashErrors {
		if v > 1 {
			result += fmt.Sprintf("%v : %v\n", k, v)
		}
	}
	return result
}

func (p *processor) do(ch <-chan string) {
	for path := range ch {
		func() {
			defer func() {
				if p := recover(); p != nil {
					fmt.Printf("Panicing at: %v", path)
					panic(p)
				}
			}()
			tf, err := os.Open(path)
			if err != nil {
				p.decodingErrors["error opening file"]++
				return
			}
			defer tf.Close()

			_, _, err = tag.Identify(tf)
			if err != nil {
				fmt.Println("IDENTIFY:", path, err.Error())
			}

			_, err = tag.ReadFrom(tf)
			if err != nil {
				fmt.Println("READFROM:", path, err.Error())
				p.decodingErrors[err.Error()]++
			}

			if sum {
				_, err = tf.Seek(0, os.SEEK_SET)
				if err != nil {
					fmt.Println("DIED:", path, "error seeking back to 0:", err)
					return
				}

				h, err := tag.Sum(tf)
				if err != nil {
					fmt.Println("SUM:", path, err.Error())
					p.hashErrors[err.Error()]++
				}
				p.hashes[h]++
			}
		}()
	}
}
