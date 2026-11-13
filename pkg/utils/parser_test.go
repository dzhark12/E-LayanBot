package utils

import "testing"

func TestRequestCode(t *testing.T) {
	for _, tc := range []struct{ input, want string }{
		{"Kode permohonan: DM01234", "DM01234"},
		{"su 12345", "SU12345"},
		{"tidak ada kode", ""},
	} {
		if got := RequestCode(tc.input); got != tc.want {
			t.Errorf("RequestCode(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
