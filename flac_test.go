package tag

import (
	"fmt"
	"os"
	"testing"
)

type expect struct {
	file         string
	sampleRate   uint32
	totalSamples uint64
	duration     float64
}

func TestReadFLACTags(t *testing.T) {
	testFiles := []expect{
		{"./testdata/without_tags/sample.flac", 11025, 37478, 3.399365},
		{"./testdata/with_tags/sample.flac", 11025, 37478, 3.399365},
	}

	for _, testFile := range testFiles {
		file, err := os.Open(testFile.file)
		if err != nil {
			panic(err)
		}

		defer file.Close()

		metadata, err := ReadFLACTags(file)

		t.Run("ReadFLACTags", func(t *testing.T) {
			if err != nil {
				t.Errorf("ReadFLACTags(%s) returned error: %v", testFile.file, err)
			}
			if metadata == nil {
				t.Errorf("ReadFLACTags(%s) returned nil metadata", testFile.file)
			}
		})

		flacMetadata, ok := metadata.(*MetadataFLAC)

		t.Run("MetadataFLAC", func(t *testing.T) {
			if !ok {
				t.Errorf("ReadFLACTags(%s) returned wrong metadata type", testFile.file)
			}
		})

		t.Run("SampleRate", func(t *testing.T) {
			if flacMetadata.SampleRate != testFile.sampleRate {
				t.Errorf("ReadFLACTags(%s) returned wrong SampleRate: %d", testFile.file, flacMetadata.SampleRate)
			}
		})
		t.Run("TotalSamples", func(t *testing.T) {
			if flacMetadata.TotalSamples != testFile.totalSamples {
				t.Errorf("ReadFLACTags(%s) returned wrong TotalSamples: %d", testFile.file, flacMetadata.TotalSamples)
			}
		})
		t.Run("Duration", func(t *testing.T) {
			if fmt.Sprintf("%.4f", flacMetadata.Duration) != fmt.Sprintf("%.4f", testFile.duration) {
				t.Errorf("ReadFLACTags(%s) returned wrong Duration: %f", testFile.file, flacMetadata.Duration)
			}
		})
	}
}
