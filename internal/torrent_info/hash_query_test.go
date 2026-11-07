package torrent_info

import "testing"

func TestIsTorrentInfoHash(t *testing.T) {
	hash := "0123456789abcdef0123456789abcdef01234567"
	tests := []struct {
		input string
		want  bool
	}{
		{hash, true},
		{"ABCDEF0123456789ABCDEF0123456789ABCDEF01", true},
		{hash[:39], false},
		{hash + "a", false},
		{"tt1234567", false},
		{"Example.Movie.2026", false},
		{hash[:39] + "x", false},
		{" " + hash, false},
	}
	for _, tc := range tests {
		if got := IsTorrentInfoHash(tc.input); got != tc.want {
			t.Errorf("IsTorrentInfoHash(%q) = %t, want %t", tc.input, got, tc.want)
		}
	}
}
