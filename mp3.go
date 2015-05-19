package tag

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
)

// Some documentation :
// http://id3.org/mp3Frame
// http://www.codeproject.com/Articles/8295/MPEG-Audio-Frame-Header

// the number of frames to scan in fast mode

type mp3Infos struct {
	Version  string
	Layer    string
	Type     string
	Mode     string
	Bitrate  int
	Sampling int
	Size     int64
	Length   float64
	vbr      int
}

func getMp3Infos(r io.ReadSeeker, slow bool) (*mp3Infos, error) {
	h := new(mp3Infos)
	var err error
	var nbscan, bitrateSum, frameCount int
	var pos, start int64
	var buf [8]byte

	nbscan = 50

	// skip the padding at the start
	for ; buf[0] == 0; _, err = r.Read(buf[0:1]) {
		if err != nil {
			return nil, err
		}
	}

	// no more padding, we are now at the start of the actual data
	start, err = r.Seek(-1, 1)
	if err != nil {
		return nil, err
	}

	// we read the first frame. Maybe a xing header
	j, err := r.Read(buf[:4])
	if j < 4 || err != nil {
		return nil, errors.New("not a MP3 file")
	}
	offset := h.readHeader(buf)
	if offset == 5 {
		return nil, errors.New("not a MP3 file")
	}
	if !(buf[0] == 255 && buf[1] >= 224) {
		return nil, errors.New("not a MP3 file")
	}

	_, err = r.Seek(xingoffset(h.Version, h.Mode), 1)
	if err != nil {
		return nil, err
	}
	_, err = r.Read(buf[:8])
	if err != nil {
		return nil, err
	}
	if !slow && (string(buf[:4]) == "Xing" || string(buf[:4]) == "Info") {
		flags := buf[7]
		if (1&flags != 0) && (2&flags != 0) {
			var frames, size uint32
			binary.Read(r, binary.BigEndian, &frames)
			binary.Read(r, binary.BigEndian, &size)
			h.Length = float64(frames) * samplePerFrame(h.Version, h.Layer) / float64(h.Sampling)
			h.Size = int64(size)
			bitrate := getNearestBitrate(float64(h.Size/125)/h.Length, h.Version, h.Layer)
			if bitrate != h.Bitrate {
				h.Bitrate = bitrate
				h.Type = "VBR"
			}
			return h, nil
		}
	}

	//TODO support VBRI Header and LAME extension

	// go to the next frame
	_, err = r.Seek(start+offset, 0)

	for i := 0; err != io.EOF && (slow || frameCount < nbscan); {
		i, err = r.Read(buf[:4])
		if i < 4 {
			break
		}
		pos += int64(i)
		// looking for the synchronization bits
		switch {
		case (buf[0] == 255) && (buf[1] >= 224):
			// found a valid mp3 frame. we read the header to know where the
			// next one is
			pos, _ = r.Seek(h.readHeader(buf)-4, 1)

			bitrateSum += h.Bitrate
			frameCount++
			if h.vbr > 2 {
				nbscan = 100
			}
			break
		case string(buf[:3]) == "TAG":
			pos, _ = r.Seek(128-4, 1) // id3v1 tag, bypass it
			break
		default:
			r.Seek(-3, 1) // looking for the next header
		}
	}

	// Extrapolate the total length base on the nbscan readHeaders
	if err == io.EOF {
		h.Size = pos
	} else {
		end, err := r.Seek(0, 2)
		if err != nil {
			return h, err
		}
		h.Length = h.Length * float64(end-int64(start)) / float64(pos-int64(start))
		h.Size = end
	}

	// For VBR, choose the closest match
	if frameCount > 1 || h.Type == "VBR" {
		h.Bitrate = getNearestBitrate(float64(bitrateSum/frameCount), h.Version, h.Layer)
	}
	return h, nil
}

func getNearestBitrate(s float64, v string, l string) int {
	diff := s
	result := int(s)
	for _, v := range mp3Bitrate[v+l] {
		if math.Abs(float64(v)-s) < diff {
			result = v
			diff = math.Abs(float64(v) - s)
		}
	}
	return result
}

func (h *mp3Infos) readHeader(buf [8]byte) int64 {
	v := buf[1] & 24 >> 3
	l := buf[1] & 6 >> 1

	b := buf[2] & 240 >> 4
	s := buf[2] & 12 >> 2
	c := buf[3] & 192 >> 6

	// if the values are off, try 1 byte after
	if l == 0 || b == 15 || v == 1 || b == 0 || s == 3 {
		return 11
	}

	if h.Version == "" {
		h.Version = mp3Version[v]
		h.Layer = mp3Layer[l]
		h.Sampling = mp3Sampling[mp3Version[v]][s]
		h.Mode = mp3Channel[c]
		h.Type = "CBR"
	}

	bitrate := mp3Bitrate[mp3Version[v]+mp3Layer[l]][b]
	mult := frameLengthMult[mp3Version[v]+mp3Layer[l]]

	switch {
	case h.vbr > 2:
		h.Type = "VBR"

	case bitrate != h.Bitrate:
		h.vbr++
	}

	h.Bitrate = bitrate

	samples := samplePerFrame(mp3Version[v], mp3Layer[l])

	h.Length += samples / float64(h.Sampling)

	return int64(mult * bitrate * 1000 / h.Sampling)
}

func xingoffset(v string, m string) int64 {
	switch {
	case v == "2" && m == "mono":
		return 9
	case v == "1" && m != "mono":
		return 32
	default:
		return 17
	}
}

func samplePerFrame(v string, l string) float64 {
	switch {
	case v == "1" && l == "I":
		return 384
	case (v == "2" || v == "2.5") && l == "III":
		return 576
	}
	return 1152
}

// constants for deconding frames
var (
	mp3Version = [4]string{"2.5", "x", "2", "1"}
	mp3Layer   = [4]string{"r", "III", "II", "I"}
	mp3Bitrate = map[string][16]int{
		"1I":     {0, 32, 64, 96, 128, 160, 192, 224, 256, 288, 320, 352, 384, 416, 448},
		"1II":    {0, 32, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320, 384},
		"1III":   {0, 32, 40, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320},
		"2I":     {0, 32, 48, 56, 64, 80, 96, 112, 128, 144, 160, 176, 192, 224, 256},
		"2II":    {0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160},
		"2III":   {0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160},
		"2.5I":   {0, 32, 48, 56, 64, 80, 96, 112, 128, 144, 160, 176, 192, 224, 256},
		"2.5II":  {0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160},
		"2.5III": {0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160},
	}
	mp3Sampling = map[string][4]int{
		"1":   {44100, 48000, 32000, 0},
		"2":   {22050, 24000, 16000, 0},
		"2.5": {11025, 12000, 8000, 0},
	}
	mp3Channel      = [4]string{"Stereo", "Join Stereo", "Dual", "Mono"}
	frameLengthMult = map[string]int{
		"1I":     48,
		"1II":    144,
		"1III":   144,
		"2I":     24,
		"2II":    144,
		"2III":   72,
		"2.5I":   24,
		"2.5II":  72,
		"2.5III": 144,
	}
)
