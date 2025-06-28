// Copyright 2015, David Howden
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tag

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
)

func getBit(b byte, n uint) bool {
	x := byte(1 << n)
	return (b & x) == x
}

func get7BitChunkedInt(b []byte) int {
	var n int
	for _, x := range b {
		n = n << 7
		n |= int(x)
	}
	return n
}

func getInt(b []byte) int {
	var n int
	for _, x := range b {
		n = n << 8
		n |= int(x)
	}
	return n
}

func readUint64LittleEndian(r io.Reader) (uint64, error) {
	b, err := readBytes(r, 8)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint64(b), nil
}

// readBytesMaxUpfront is the max up-front allocation allowed
const readBytesMaxUpfront = 10 << 20 // 10MB

func readBytes(r io.Reader, n uint) ([]byte, error) {
	if n > readBytesMaxUpfront {
		b := &bytes.Buffer{}
		if _, err := io.CopyN(b, r, int64(n)); err != nil {
			return nil, err
		}
		return b.Bytes(), nil
	}

	b := make([]byte, n)
	_, err := io.ReadFull(r, b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func readString(r io.Reader, n uint) (string, error) {
	b, err := readBytes(r, n)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func readUint(r io.Reader, n uint) (uint, error) {
	x, err := readInt(r, n)
	if err != nil {
		return 0, err
	}
	return uint(x), nil
}

func readInt(r io.Reader, n uint) (int, error) {
	b, err := readBytes(r, n)
	if err != nil {
		return 0, err
	}
	return getInt(b), nil
}

func read7BitChunkedUint(r io.Reader, n uint) (uint, error) {
	b, err := readBytes(r, n)
	if err != nil {
		return 0, err
	}
	return uint(get7BitChunkedInt(b)), nil
}

func readUint32LittleEndian(r io.Reader) (uint32, error) {
	b, err := readBytes(r, 4)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(b), nil
}

func readUint32BigEndian(r io.Reader) (uint32, error) {
	b, err := readBytes(r, 4)
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint32(b), nil
}

func cutBits(in []byte, offset, n uint) (uint64, error) {
	if n > 64 {
		return 0, errors.New("n exceeds maximum value of 64")
	}
	if len(in)*8 < int(offset+n) {
		return 0, errors.New("out of bounds read")
	}
	var res uint64
	var bitsRead uint
	if splitStart := offset % 8; splitStart > 0 {
		remaining := 8 - splitStart
		splitByte := uint64(in[int(offset/8)]) & ((1 << remaining) - 1)
		if n <= remaining {
			return uint64(splitByte) >> uint64(remaining-n), nil
		}
		bitsRead = remaining
		res |= splitByte
	}

	wholeBytes := int((n - bitsRead) / 8)
	start := int((offset + bitsRead) / 8)
	for i := range wholeBytes {
		res <<= 8
		res |= uint64(in[start+i])
		bitsRead += 8
	}

	if remaining := n - bitsRead; remaining > 0 {
		res <<= remaining
		res |= uint64(in[start+wholeBytes]) >> (8 - remaining)
	}
	return res, nil
}
