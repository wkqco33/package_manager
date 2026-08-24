package apps

import "testing"

func TestDefaultAppsAreWellFormed(t *testing.T) {
	if len(DefaultApps) == 0 {
		t.Fatal("DefaultApps must not be empty")
	}
	seen := make(map[string]bool, len(DefaultApps))
	for _, app := range DefaultApps {
		if app.Name == "" {
			t.Errorf("DefaultApp has empty Name")
		}
		if app.BinName == "" {
			t.Errorf("DefaultApp %q has empty BinName", app.Name)
		}
		if app.Description == "" {
			t.Errorf("DefaultApp %q has empty Description", app.Name)
		}
		if app.Homepage == "" {
			t.Errorf("DefaultApp %q has empty Homepage", app.Name)
		}
		if seen[app.Name] {
			t.Errorf("duplicate DefaultApp name: %q", app.Name)
		}
		seen[app.Name] = true
	}
}

func TestDefaultAppsUseOwnerRepoFormat(t *testing.T) {
	for _, app := range DefaultApps {
		parts := splitOwnerRepo(app.Name)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			t.Errorf("DefaultApp name %q must be in owner/repo format", app.Name)
		}
	}
}

func splitOwnerRepo(name string) []string {
	for i := 0; i < len(name); i++ {
		if name[i] == '/' {
			return []string{name[:i], name[i+1:]}
		}
	}
	return nil
}
