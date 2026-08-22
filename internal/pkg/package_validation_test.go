package pkg

import "testing"

func TestPackageValidate(t *testing.T) {
	tests := []struct {
		name    string
		pkg     Package
		wantErr bool
	}{
		{
			name: "valid package",
			pkg: Package{
				Name: "owner/repo", Version: "v1.0.0", Source: "https://example.com/release.tar.gz",
			},
		},
		{name: "missing name", pkg: Package{Version: "v1.0.0", Source: "source"}, wantErr: true},
		{name: "missing version", pkg: Package{Name: "owner/repo", Source: "source"}, wantErr: true},
		{name: "missing source", pkg: Package{Name: "owner/repo", Version: "v1.0.0"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.pkg.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
