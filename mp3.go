package tag

import (
	"fmt"
	"io"
	"math"
	"time"
)

type mpegVersion int

const (
	mpeg25 mpegVersion = iota
	mpegReserved
	mpeg2
	mpeg1
	mpegMax
)

type mpegLayer int

const (
	layerReserved mpegLayer = iota
	layer3
	layer2
	layer1
	layerMax
)

// Took from: https://github.com/tcolgate/mp3/blob/master/frames.go
var (
	bitrates = [mpegMax][layerMax][15]int{
		{ // MPEG 2.5
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},                       // LayerReserved
			{0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160},      // Layer3
			{0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160},      // Layer2
			{0, 32, 48, 56, 64, 80, 96, 112, 128, 144, 160, 176, 192, 224, 256}, // Layer1
		},
		{ // Reserved
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, // LayerReserved
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, // Layer3
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, // Layer2
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, // Layer1
		},
		{ // MPEG 2
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},                       // LayerReserved
			{0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160},      // Layer3
			{0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160},      // Layer2
			{0, 32, 48, 56, 64, 80, 96, 112, 128, 144, 160, 176, 192, 224, 256}, // Layer1
		},
		{ // MPEG 1
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},                          // LayerReserved
			{0, 32, 40, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320},     // Layer3
			{0, 32, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320, 384},    // Layer2
			{0, 32, 64, 96, 128, 160, 192, 224, 256, 288, 320, 352, 384, 416, 448}, // Layer1
		},
	}
	sampleRates = [int(mpegMax)][3]int{
		{11025, 12000, 8000},  //MPEG25
		{0, 0, 0},             //MPEGReserved
		{22050, 24000, 16000}, //MPEG2
		{44100, 48000, 32000}, //MPEG1
	}

	samplesPerFrame = [mpegMax][layerMax]int{
		{ // MPEG25
			0,
			576,
			1152,
			384,
		},
		{ // Reserved
			0,
			0,
			0,
			0,
		},
		{ // MPEG2
			0,
			576,
			1152,
			384,
		},
		{ // MPEG1
			0,
			1152,
			1152,
			384,
		},
	}
	slotSize = [layerMax]int{
		0, //	LayerReserved
		1, //	Layer3
		1, //	Layer2
		4, //	Layer1
	}
)

type metadataV2MP3 struct {
	*metadataID3v2
	duration time.Duration
}

type metadataV1MP3 struct {
	*metadataID3v1
	duration time.Duration
}

func getMP3Duration(header []byte, strippedSize int64) (time.Duration, error) {
	version, err := cutBits(header, 11, 2)
	if err != nil {
		return 0, fmt.Errorf("reading mpeg version: %w", err)
	}
	layer, err := cutBits(header, 13, 2)
	if err != nil {
		return 0, fmt.Errorf("reading mpeg layer: %w", err)
	}
	protection, err := cutBits(header, 15, 1)
	if err != nil {
		return 0, fmt.Errorf("reading mpeg protection: %w", err)
	}
	bitrateIndex, err := cutBits(header, 16, 4)
	if err != nil {
		return 0, fmt.Errorf("reading mpeg bitrate index: %w", err)
	}
	samplerateIndex, err := cutBits(header, 20, 2)
	if err != nil {
		return 0, fmt.Errorf("reading mpeg samplerate index: %w", err)
	}
	padding, err := cutBits(header, 21, 1)
	if err != nil {
		return 0, fmt.Errorf("reading mpeg padding bit: %w", err)
	}
	frameSampleNum := samplesPerFrame[version][layer]
	frameDuration := float64(frameSampleNum) / float64(sampleRates[version][samplerateIndex])
	frameSize := math.Floor(((frameDuration * float64(bitrates[version][layer][bitrateIndex])) * 1000) / 8)
	if padding == 1 {
		frameSize += float64(slotSize[layer])
	}
	if protection == 1 {
		frameSize += 2
	}
	// add the header length
	frameSize += 4
	duration := time.Second * time.Duration(math.Round((float64(strippedSize)/float64(frameSize))*frameDuration))

	return duration, nil
}

func ReadV2MP3Meta(r io.ReadSeeker, size int64) (Metadata, error) {
	tagMeta, err := ReadID3v2Tags(r)
	if err != nil {
		return nil, fmt.Errorf("reading id3v2 tags: %w", err)
	}

	id3Size := tagMeta.header.Size + 10
	if tagMeta.header.FooterPresent {
		id3Size += 10
	}
	_, err = r.Seek(int64(id3Size), io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("seeking to skip id3v2: %w", err)

	}

	header := make([]byte, 4)
	_, err = io.ReadFull(r, header)
	if err != nil {
		return nil, fmt.Errorf("reading first frame header: %w", err)
	}

	duration, err := getMP3Duration(header, size-int64(id3Size))
	if err != nil {
		return nil, fmt.Errorf("reading the mp3 duration: %w", err)
	}

	return &metadataV2MP3{
		metadataID3v2: tagMeta,
		duration:      duration,
	}, nil

}

func ReadV1MP3Meta(r io.ReadSeeker, size int64) (Metadata, error) {
	tagMeta, err := ReadID3v1Tags(r)
	if err != nil {
		return nil, fmt.Errorf("reading id3v2 tags: %w", err)
	}

	_, err = r.Seek(0, io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("seeking to the start: %w", err)

	}

	header := make([]byte, 4)
	_, err = io.ReadFull(r, header)
	if err != nil {
		return nil, fmt.Errorf("reading first frame header: %w", err)
	}

	duration, err := getMP3Duration(header, size-128)
	if err != nil {
		return nil, fmt.Errorf("reading the mp3 duration: %w", err)
	}

	return &metadataV1MP3{
		metadataID3v1: &tagMeta,
		duration:      duration,
	}, nil

}

func (m *metadataV2MP3) Duration() time.Duration {
	return m.duration
}

func (m *metadataV1MP3) Duration() time.Duration {
	return m.duration
}
