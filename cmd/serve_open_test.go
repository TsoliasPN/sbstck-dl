package cmd

import (
	"reflect"
	"testing"
)

func TestBrowserCommandForOS(t *testing.T) {
	url := "http://127.0.0.1:8787"

	tests := []struct {
		name     string
		goos     string
		wantArgs []string
		wantErr  bool
	}{
		{
			name:     "windows",
			goos:     "windows",
			wantArgs: []string{"cmd", "/c", "start", "", url},
		},
		{
			name:     "darwin",
			goos:     "darwin",
			wantArgs: []string{"open", url},
		},
		{
			name:     "linux",
			goos:     "linux",
			wantArgs: []string{"xdg-open", url},
		},
		{
			name:    "unknown",
			goos:    "plan9",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := browserCommandForOS(tt.goos, url)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(cmd.Args, tt.wantArgs) {
				t.Fatalf("unexpected args: %v", cmd.Args)
			}
		})
	}
}
