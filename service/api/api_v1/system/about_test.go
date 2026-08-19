package system

import "testing"

func TestLatestReleaseAPIURL(t *testing.T) {
	const expected = "https://api.github.com/repos/crossgg/sun-panel-mod/releases/latest"
	if latestReleaseAPIURL != expected {
		t.Fatalf("latest release API URL is %q, want %q", latestReleaseAPIURL, expected)
	}
}

func TestReleaseVersionFromRedirect(t *testing.T) {
	tests := []struct {
		location   string
		want       string
		recognized bool
	}{
		{
			location:   "https://github.com/crossgg/sun-panel-mod/releases/tag/v1.8.1",
			want:       "v1.8.1",
			recognized: true,
		},
		{
			location:   "https://github.com/crossgg/sun-panel-mod/releases",
			recognized: true,
		},
		{
			location: "https://github.com/crossgg/sun-panel-mod",
		},
	}
	for _, test := range tests {
		got, recognized := releaseVersionFromRedirect(test.location)
		if got != test.want || recognized != test.recognized {
			t.Errorf("releaseVersionFromRedirect(%q) = (%q, %v), want (%q, %v)", test.location, got, recognized, test.want, test.recognized)
		}
	}
}

func TestIsNewerVersion(t *testing.T) {
	tests := []struct {
		latest  string
		current string
		want    bool
	}{
		{latest: "1.8.1", current: "1.4.7", want: true},
		{latest: "1.10.0", current: "1.9.0", want: true},
		{latest: "v1.4.7", current: "1.4.7", want: false},
		{latest: "1.4.6", current: "1.4.7", want: false},
	}
	for _, test := range tests {
		if got := isNewerVersion(test.latest, test.current); got != test.want {
			t.Errorf("isNewerVersion(%q, %q) = %v, want %v", test.latest, test.current, got, test.want)
		}
	}
}
