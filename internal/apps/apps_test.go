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

func TestFilterUninstalled(t *testing.T) {
	sampleApps := []DefaultApp{
		{Name: "user/app1", BinName: "app1"},
		{Name: "user/app2", BinName: "app2"},
		{Name: "user/app3", BinName: "app3"},
	}

	installed := []string{"user/app1", "app2"} // full name or base name
	uninstalled := FilterUninstalled(sampleApps, installed)

	if len(uninstalled) != 1 || uninstalled[0].Name != "user/app3" {
		t.Fatalf("expected [user/app3], got %v", uninstalled)
	}

	// All installed
	allInstalled := FilterUninstalled(sampleApps, []string{"user/app1", "user/app2", "user/app3"})
	if len(allInstalled) != 0 {
		t.Fatalf("expected empty slice, got %v", allInstalled)
	}
}

func TestFilterByName(t *testing.T) {
	sampleApps := []DefaultApp{
		{Name: "user/app1", BinName: "app1"},
		{Name: "user/app2", BinName: "app2"},
	}

	// Match by full name
	matched, err := FilterByName(sampleApps, []string{"user/app1"})
	if err != nil || len(matched) != 1 || matched[0].Name != "user/app1" {
		t.Fatalf("unexpected match for full name: %v, %v", matched, err)
	}

	// Match by short name or bin name
	matched, err = FilterByName(sampleApps, []string{"app2"})
	if err != nil || len(matched) != 1 || matched[0].Name != "user/app2" {
		t.Fatalf("unexpected match for short/bin name: %v, %v", matched, err)
	}

	// Unknown name returns error
	_, err = FilterByName(sampleApps, []string{"nonexistent"})
	if err == nil {
		t.Fatalf("expected error for nonexistent app, got nil")
	}
}
