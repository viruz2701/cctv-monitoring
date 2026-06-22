package protocols

import (
	"os"
	"testing"
)

func TestFTPDriverSafePath(t *testing.T) {
	driver := &FTPDriver{
		rootPath: "/tmp/ftp-root",
	}

	tests := []struct {
		name       string
		userPath   string
		wantErr    error
		wantSuffix string
	}{
		{
			name:       "normal subdir",
			userPath:   "subdir/file.txt",
			wantErr:    nil,
			wantSuffix: "/tmp/ftp-root/subdir/file.txt",
		},
		{
			name:       "root path",
			userPath:   "/",
			wantErr:    nil,
			wantSuffix: "/tmp/ftp-root",
		},
		{
			name:     "path traversal ../../etc/passwd",
			userPath: "../../etc/passwd",
			wantErr:  os.ErrPermission,
		},
		{
			name:     "path traversal with absolute",
			userPath: "/etc/passwd",
			wantErr:  os.ErrPermission,
		},
		{
			name:     "path traversal with double dots mid",
			userPath: "subdir/../../../etc/passwd",
			wantErr:  os.ErrPermission,
		},
		{
			name:       "normal with dots in filename",
			userPath:   "subdir/file.with.dots.txt",
			wantErr:    nil,
			wantSuffix: "/tmp/ftp-root/subdir/file.with.dots.txt",
		},
		{
			name:       "empty path",
			userPath:   "",
			wantErr:    nil,
			wantSuffix: "/tmp/ftp-root",
		},
		{
			name:     "symlink-like traversal",
			userPath: "subdir/../../etc/shadow",
			wantErr:  os.ErrPermission,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPath, gotErr := driver.safePath(tt.userPath)

			if tt.wantErr != nil {
				if gotErr == nil {
					t.Errorf("safePath(%q) wanted error, got nil (path=%s)", tt.userPath, gotPath)
				}
				return
			}

			if gotErr != nil {
				t.Errorf("safePath(%q) unexpected error: %v", tt.userPath, gotErr)
				return
			}

			if gotPath != tt.wantSuffix {
				t.Errorf("safePath(%q) = %q, want suffix %q", tt.userPath, gotPath, tt.wantSuffix)
			}
		})
	}
}
