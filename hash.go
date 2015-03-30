package tag

import (
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"os"
)

// Hash creates a hash of the audio file data provided by the io.ReadSeeker which metadata
// (ID3, MP4) invariant.
func Hash(r io.ReadSeeker) (string, error) {
	b, err := readBytes(r, 11)
	if err != nil {
		return "", err
	}

	if string(b[4:11]) == "ftypM4A" {
		return HashAtoms(r)
	}

	if string(b[0:3]) == "ID3" {
		return HashID3v2(r)
	}

	h, err := HashID3v1(r)
	if err != nil {
		if err == ErrNotID3v1 {
			return HashAll(r)
		}
		return "", err
	}
	return h, nil
}

// HashAll returns a hash of the entire content.
func HashAll(r io.ReadSeeker) (string, error) {
	_, err := r.Seek(0, os.SEEK_SET)
	if err != nil {
		return "", fmt.Errorf("error seeking to 0: %v", err)
	}

	b, err := ioutil.ReadAll(r)
	if err != nil {
		return "", nil
	}
	return hash(b), nil
}

// HashAtoms constructs a hash of MP4 audio file data provided by the io.ReadSeeker which is metadata invariant.
func HashAtoms(r io.ReadSeeker) (string, error) {
	_, err := r.Seek(0, os.SEEK_SET)
	if err != nil {
		return "", fmt.Errorf("error seeking to 0: %v", err)
	}

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
			return HashAtoms(r)

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
			return hash(b), nil
		}

		_, err = r.Seek(int64(size-8), os.SEEK_CUR)
		if err != nil {
			return "", fmt.Errorf("error reading '%v' tag: %v", name, err)
		}
	}
}

// HashID3v1 constructs a hash of MP3 audio file data (assumed to have ID3v1 tags) provided by the
// io.ReadSeeker which is metadata invariant.
func HashID3v1(r io.ReadSeeker) (string, error) {
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
	return hash(b[:len(b)-128]), nil
}

// HashID3v2 constructs a hash of MP3 audio file data (assumed to have ID3v2 tags) provided by the
// io.ReadSeeker which is metadata invariant.
func HashID3v2(r io.ReadSeeker) (string, error) {
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
	return hash(b[:len(b)-128]), nil
}

func hash(b []byte) string {
	return fmt.Sprintf("%x", sha1.Sum(b))
}
