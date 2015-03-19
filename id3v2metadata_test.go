// Copyright 2015, David Howden
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tag

import "testing"

func TestParseXofN(t *testing.T) {
	table := []struct {
		str  string
		x, n int
	}{
		{"", 0, 0},
		{"1", 1, 0},
		{"0/2", 0, 2},
		{"1/2", 1, 2},
		{"1 / 2", 1, 2},
		{"1/", 1, 0},
		{"/2", 0, 2},
	}

	for ii, tt := range table {
		gotX, gotN := parseXofN(tt.str)
		if gotX != tt.x || gotN != tt.n {
			t.Errorf("[%d] parseXofN(%v) = %d, %d, expected: %d, %d", ii, tt.str, gotX, gotN, tt.x, tt.n)
		}
	}
}
