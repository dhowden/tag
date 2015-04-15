package tag

import (
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"os"
)

// Sum creates a checksum of the audio file data provided by the io.ReadSeeker which is metadata
// (ID3, MP4) invariant.
func Sum(r io.ReadSeeker) (string, error) {
	b, err := readBytes(r, 11)
	if err != nil {
		return "", err
	}

	if string(b[4:11]) == "ftypM4A" {
		return SumAtoms(r)
	}

	if string(b[0:3]) == "ID3" {
		return SumID3v2(r)
	}

	h, err := SumID3v1(r)
	if err != nil {
		if err == ErrNotID3v1 {
			return SumAll(r)
		}
		return "", err
	}
	return h, nil
}

// SumAll returns a checksum of the entire content.
func SumAll(r io.ReadSeeker) (string, error) {
	_, err := r.Seek(0, os.SEEK_SET)
	if err != nil {
		return "", fmt.Errorf("error seeking to 0: %v", err)
	}

	b, err := ioutil.ReadAll(r)
	if err != nil {
		return "", nil
	}
	return sum(b), nil
}

// SumAtoms constructs a checksum of MP4 audio file data provided by the io.ReadSeeker which is
// metadata invariant.
func SumAtoms(r io.ReadSeeker) (string, error) {
	_, err := r.Seek(0, os.SEEK_SET)
	if err != nil {
		return "", fmt.Errorf("error seeking to 0: %v", err)
	}
	return sumAtoms(r)
}

func sumAtoms(r io.ReadSeeker) (string, error) {
	for {
		var size uint32
		err := binary.Read(r, binary.BigEndian, &size)
		if err != nil {
			if err == io.EOF {
				return "", fmt.Errorf("reached EOF before audio data")
			}
			return "", err
		}

		name, err := readString(r, 4)
		if err != nil {
			return "", err
		}

		switch name {
		case "meta":
			// next_item_id (int32)
			_, err := readBytes(r, 4)
			if err != nil {
				return "", err
			}
			fallthrough

		case "moov", "udta", "ilst":
			return sumAtoms(r)

		case "free":
			_, err = r.Seek(int64(size-8), os.SEEK_CUR)
			if err != nil {
				return "", fmt.Errorf("error reading 'free' space: %v", err)
			}
			continue

		case "mdat": // stop when we get to the data
			b, err := readBytes(r, int(size-8))
			if err != nil {
				return "", fmt.Errorf("error reading audio data: %v", err)
			}
			return sum(b), nil
		}

		_, err = r.Seek(int64(size-8), os.SEEK_CUR)
		if err != nil {
			return "", fmt.Errorf("error reading '%v' tag: %v", name, err)
		}
	}
}

// SumID3v1 constructs a checksum of MP3 audio file data (assumed to have ID3v1 tags) provided
// by the io.ReadSeeker which is metadata invariant.
func SumID3v1(r io.ReadSeeker) (string, error) {
	_, err := r.Seek(0, os.SEEK_SET)
	if err != nil {
		return "", fmt.Errorf("error seeking to 0: %v", err)
	}

	b, err := ioutil.ReadAll(r)
	if err != nil {
		return "", err
	}

	if len(b) < 128 {
		return "", fmt.Errorf("file size must be greater than 128 bytes for ID3v1 metadata (size: %v)", len(b))
	}
	return sum(b[:len(b)-128]), nil
}

// SumID3v2 constructs a checksum of MP3 audio file data (assumed to have ID3v2 tags) provided by the
// io.ReadSeeker which is metadata invariant.
func SumID3v2(r io.ReadSeeker) (string, error) {
	_, err := r.Seek(0, os.SEEK_SET)
	if err != nil {
		return "", fmt.Errorf("error seeking to 0: %v", err)
	}

	h, err := readID3v2Header(r)
	if err != nil {
		return "", fmt.Errorf("error reading ID3v2 header: %v", err)
	}

	_, err = r.Seek(int64(h.Size), os.SEEK_SET)
	if err != nil {
		return "", fmt.Errorf("error seeking to end of ID3V2 header: %v", err)
	}

	b, err := ioutil.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("error reading audio data: %v", err)
	}

	if len(b) < 128 {
		return "", fmt.Errorf("file size must be greater than 128 bytes for MP3 (ID3v2 header size: %d, remaining: %d)", h.Size, len(b))
	}
	return sum(b[:len(b)-128]), nil
}

func sum(b []byte) string {
	return fmt.Sprintf("%x", sha1.Sum(b))
}
