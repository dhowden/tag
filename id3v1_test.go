// This file is subject to the CC0 1.0 Universal (CC0 1.0) Public Domain Dedication
// license.  Its contents can be found at:
// http://creativecommons.org/publicdomain/zero/1.0

package tag

import (
	"bytes"
	"testing"
)

//go:generate go-bindata -o id3v1_testdata.go -pkg tag -ignore .txt id3v1_test

func TestReadID3v1Tags(t *testing.T) {
	for _, name := range []string{
		"id3v1_test/sample_usascii_v1.mp3",
		"id3v1_test/sample_ms932_v1.mp3",
		"id3v1_test/sample_utf8_v1.mp3"} {
		doTest(name, 0, 30, t)
	}
	for _, name := range []string{
		"id3v1_test/sample_usascii_v1.1.mp3",
		"id3v1_test/sample_ms932_v1.1.mp3",
		"id3v1_test/sample_utf8_v1.1.mp3"} {
		doTest(name, 1, 28, t)
	}
}

func doTest(name string, track int, length int, t *testing.T) {
	mp3 := MustAsset(name)
	metadata, _ := ReadID3v1Tags(bytes.NewReader(mp3))
	if actual, total := metadata.Track(); actual != track || total != 0 {
		t.Errorf("Track number for %s is (%d, %d) where (%d, 0) is expected.", name, actual, total, track)
	}
	comment := metadata.Raw()["comment"].(string)
	if actual := len(comment); actual != length {
		t.Errorf("Comment length for %s is %d where %d is expected", name, actual, length)
	}
}
