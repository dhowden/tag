// Copyright 2015, David Howden
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tag

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"time"
)

var (
	vorbisIdentificationPrefix = []byte("\x01vorbis")
	vorbisCommentPrefix        = []byte("\x03vorbis")
	opusTagsPrefix             = []byte("OpusTags")
)

var oggCRC32Poly04c11db7 = oggCRCTable(0x04c11db7)

type crc32Table [256]uint32

func oggCRCTable(poly uint32) *crc32Table {
	var t crc32Table

	for i := 0; i < 256; i++ {
		crc := uint32(i) << 24
		for j := 0; j < 8; j++ {
			if crc&0x80000000 != 0 {
				crc = (crc << 1) ^ poly
			} else {
				crc <<= 1
			}
		}
		t[i] = crc
	}

	return &t
}

func oggCRCUpdate(crc uint32, tab *crc32Table, p []byte) uint32 {
	for _, v := range p {
		crc = (crc << 8) ^ tab[byte(crc>>24)^v]
	}
	return crc
}

type oggPageHeader struct {
	Magic           [4]byte // "OggS"
	Version         uint8
	Flags           uint8
	GranulePosition uint64
	SerialNumber    uint32
	SequenceNumber  uint32
	CRC             uint32
	Segments        uint8
}

type oggDemuxer struct {
	packetBufs map[uint32]*bytes.Buffer
}

// Read ogg packets, can return empty slice of packets and nil err
// if more data is needed
func (o *oggDemuxer) Read(r io.Reader) ([][]byte, int, error) {
	headerBuf := &bytes.Buffer{}
	var oh oggPageHeader
	if err := binary.Read(io.TeeReader(r, headerBuf), binary.LittleEndian, &oh); err != nil {
		return nil, 0, err
	}

	if bytes.Compare(oh.Magic[:], []byte("OggS")) != 0 {
		// TODO: seek for syncword?
		return nil, 0, errors.New("expected 'OggS'")
	}

	segmentTable := make([]byte, oh.Segments)
	if _, err := io.ReadFull(r, segmentTable); err != nil {
		return nil, 0, err
	}
	var segmentsSize int64
	for _, s := range segmentTable {
		segmentsSize += int64(s)
	}
	segmentsData := make([]byte, segmentsSize)
	if _, err := io.ReadFull(r, segmentsData); err != nil {
		return nil, 0, err
	}

	headerBytes := headerBuf.Bytes()
	// reset CRC to zero in header before checksum
	headerBytes[22] = 0
	headerBytes[23] = 0
	headerBytes[24] = 0
	headerBytes[25] = 0
	crc := oggCRCUpdate(0, oggCRC32Poly04c11db7, headerBytes)
	crc = oggCRCUpdate(crc, oggCRC32Poly04c11db7, segmentTable)
	crc = oggCRCUpdate(crc, oggCRC32Poly04c11db7, segmentsData)
	if crc != oh.CRC {
		return nil, 0, fmt.Errorf("expected crc %x != %x", oh.CRC, crc)
	}

	if o.packetBufs == nil {
		o.packetBufs = map[uint32]*bytes.Buffer{}
	}

	var packetBuf *bytes.Buffer
	continued := oh.Flags&0x1 != 0
	if continued {
		if b, ok := o.packetBufs[oh.SerialNumber]; ok {
			packetBuf = b
		} else {
			return nil, 0, fmt.Errorf("could not find continued packet %d", oh.SerialNumber)
		}
	} else {
		packetBuf = &bytes.Buffer{}
	}

	var packets [][]byte
	var p int
	for _, s := range segmentTable {
		packetBuf.Write(segmentsData[p : p+int(s)])
		if s < 255 {
			packets = append(packets, packetBuf.Bytes())
			packetBuf = &bytes.Buffer{}
		}
		p += int(s)
	}

	o.packetBufs[oh.SerialNumber] = packetBuf

	return packets, int(oh.GranulePosition), nil
}

// ReadOGGMeta reads OGG metadata from the io.ReadSeeker, returning the resulting
// metadata in a Metadata implementation, or non-nil error if there was a problem.
// See http://www.xiph.org/vorbis/doc/Vorbis_I_spec.html
// and http://www.xiph.org/ogg/doc/framing.html for details.
// For Opus see https://tools.ietf.org/html/rfc7845
func ReadOGGMeta(r io.Reader) (Metadata, error) {
	od := &oggDemuxer{}
	metaExtracted := false
	m := &metadataOGG{
		metadataVorbis: newMetadataVorbis(),
	}
	prevPos := 0
	for {
		bs, pos, err := od.Read(r)
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, err
		}
		if errors.Is(err, io.EOF) {
			if !metaExtracted {
				return nil, ErrNoTagsFound
			}
			if m.sampleRate > 0 {
				m.duration = time.Second * (time.Duration(prevPos) / time.Duration(m.sampleRate))
			}
			return m, nil
		}
		prevPos = pos

		for _, b := range bs {
			switch {
			case bytes.HasPrefix(b, vorbisCommentPrefix):
				metaExtracted = true
				err = m.readVorbisComment(bytes.NewReader(b[len(vorbisCommentPrefix):]))
			case bytes.HasPrefix(b, opusTagsPrefix):
				metaExtracted = true
				err = m.readVorbisComment(bytes.NewReader(b[len(opusTagsPrefix):]))
				m.sampleRate = 48000
			case bytes.HasPrefix(b, vorbisIdentificationPrefix):
				err = m.readVorbisIdentification(bytes.NewReader(b[len(vorbisIdentificationPrefix):]))
			}
			if err != nil {
				return m, err
			}
		}
	}
}

type metadataOGG struct {
	*metadataVorbis
	sampleRate uint32
	duration   time.Duration
}

func (m *metadataOGG) FileType() FileType {
	return OGG
}

func (m *metadataOGG) Duration() time.Duration {
	return m.duration
}

func (m *metadataOGG) readVorbisIdentification(r io.ReadSeeker) error {
	_, err := r.Seek(5, io.SeekCurrent)
	if err != nil {
		return err
	}
	m.sampleRate, err = readUint32LittleEndian(r)
	if err != nil {
		return err
	}
	return nil
}
