package utils

import (
	"runtime"
	"testing"
)

func TestURIToPath(t *testing.T) {
	tests := []struct {
		name    string
		uri     string
		want    string
		wantErr bool
		skipOn  string // Skip this test on specific OS (e.g., "windows", "!windows")
	}{
		{
			name:    "empty URI",
			uri:     "",
			want:    "",
			wantErr: true,
		},
		{
			name:    "non-file URI scheme",
			uri:     "http://example.com/path",
			want:    "",
			wantErr: true,
		},
		{
			name:   "Unix absolute path",
			uri:    "file:///Users/test/project/file.yaml",
			want:   "/Users/test/project/file.yaml",
			skipOn: "windows",
		},
		{
			name:   "Unix path with spaces",
			uri:    "file:///Users/test/my%20project/file.yaml",
			want:   "/Users/test/my project/file.yaml",
			skipOn: "windows",
		},
		{
			name:   "Unix path with special characters",
			uri:    "file:///Users/test/project%20%28copy%29/file.yaml",
			want:   "/Users/test/project (copy)/file.yaml",
			skipOn: "windows",
		},
		{
			name:   "Windows path with drive letter",
			uri:    "file:///C:/Users/test/project/file.yaml",
			want:   "C:\\Users\\test\\project\\file.yaml",
			skipOn: "!windows",
		},
		{
			name:   "Windows path with spaces",
			uri:    "file:///C:/Users/test/my%20project/file.yaml",
			want:   "C:\\Users\\test\\my project\\file.yaml",
			skipOn: "!windows",
		},
		{
			name:   "Windows path with lowercase drive",
			uri:    "file:///c:/Users/test/file.yaml",
			want:   "c:\\Users\\test\\file.yaml",
			skipOn: "!windows",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip test based on platform
			if tt.skipOn == "windows" && runtime.GOOS == "windows" {
				t.Skip("Skipping on Windows")
			}
			if tt.skipOn == "!windows" && runtime.GOOS != "windows" {
				t.Skip("Skipping on non-Windows")
			}

			got, err := URIToPath(tt.uri)
			if (err != nil) != tt.wantErr {
				t.Errorf("URIToPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("URIToPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPathToURI(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		want   string
		skipOn string
	}{
		{
			name: "empty path",
			path: "",
			want: "",
		},
		{
			name:   "Unix absolute path",
			path:   "/Users/test/project/file.yaml",
			want:   "file:///Users/test/project/file.yaml",
			skipOn: "windows",
		},
		{
			name:   "Unix path with spaces",
			path:   "/Users/test/my project/file.yaml",
			want:   "file:///Users/test/my%20project/file.yaml",
			skipOn: "windows",
		},
		{
			name:   "Unix path with special characters",
			path:   "/Users/test/project (copy)/file.yaml",
			want:   "file:///Users/test/project%20%28copy%29/file.yaml",
			skipOn: "windows",
		},
		{
			name:   "Windows path with backslashes",
			path:   "C:\\Users\\test\\project\\file.yaml",
			want:   "file:///C:/Users/test/project/file.yaml",
			skipOn: "!windows",
		},
		{
			name:   "Windows path with spaces",
			path:   "C:\\Users\\test\\my project\\file.yaml",
			want:   "file:///C:/Users/test/my%20project/file.yaml",
			skipOn: "!windows",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip test based on platform
			if tt.skipOn == "windows" && runtime.GOOS == "windows" {
				t.Skip("Skipping on Windows")
			}
			if tt.skipOn == "!windows" && runtime.GOOS != "windows" {
				t.Skip("Skipping on non-Windows")
			}

			got := PathToURI(tt.path)
			if got != tt.want {
				t.Errorf("PathToURI() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsFileURI(t *testing.T) {
	tests := []struct {
		name string
		uri  string
		want bool
	}{
		{
			name: "file URI",
			uri:  "file:///path/to/file",
			want: true,
		},
		{
			name: "http URI",
			uri:  "http://example.com",
			want: false,
		},
		{
			name: "https URI",
			uri:  "https://example.com",
			want: false,
		},
		{
			name: "empty string",
			uri:  "",
			want: false,
		},
		{
			name: "relative path",
			uri:  "path/to/file",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsFileURI(tt.uri); got != tt.want {
				t.Errorf("IsFileURI() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestRoundTrip tests that converting path -> URI -> path returns the original path
func TestRoundTrip(t *testing.T) {
	paths := []string{
		"/Users/test/project/file.yaml",
		"/Users/test/my project/file with spaces.yaml",
		"/tmp/test.yaml",
	}

	// On Windows, add Windows-specific paths
	if runtime.GOOS == "windows" {
		paths = []string{
			"C:\\Users\\test\\project\\file.yaml",
			"C:\\Users\\test\\my project\\file.yaml",
			"D:\\test\\file.yaml",
		}
	}

	for _, originalPath := range paths {
		t.Run(originalPath, func(t *testing.T) {
			uri := PathToURI(originalPath)
			gotPath, err := URIToPath(uri)
			if err != nil {
				t.Errorf("Round trip failed: %v", err)
				return
			}

			// Normalize paths for comparison (handle forward/backward slashes)
			if gotPath != originalPath {
				t.Errorf("Round trip failed: got %v, want %v", gotPath, originalPath)
			}
		})
	}
}
