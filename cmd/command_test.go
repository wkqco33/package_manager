package cmd

import (
	"errors"
	"testing"

	"ppm/internal/config"
	"ppm/internal/pkg"
	"ppm/internal/platform"
)

func TestInstallCommandRejectsMissingPackage(t *testing.T) {
	err := ExecuteArgs([]string{"install"})
	if err == nil {
		t.Fatal("expected install to reject missing package arguments")
	}
}

func TestInstallCommandInjectsConfigLoader(t *testing.T) {
	wantErr := errors.New("config unavailable")
	command := newInstallCommand(installDependencies{
		LoadConfig: func() (*config.Config, error) { return nil, wantErr },
		NewFetcher: func(*config.Config) pkg.RegistryFetcher { return nil },
	})
	if err := command.Execute([]string{"owner/repo"}); !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want injected config error", err)
	}
}

func TestUpdateCommandInjectsConfigLoader(t *testing.T) {
	wantErr := errors.New("config unavailable")
	command := newUpdateCommand(updateDependencies{
		LoadConfig: func() (*config.Config, error) { return nil, wantErr },
	})
	if err := command.Execute(nil); !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want injected config error", err)
	}
}

func TestUninstallCommandInjectsConfigLoader(t *testing.T) {
	wantErr := errors.New("config unavailable")
	command := newUninstallCommand(uninstallDependencies{
		LoadConfig: func() (*config.Config, error) { return nil, wantErr },
	})
	if err := command.Execute([]string{"owner/repo"}); !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want injected config error", err)
	}
}

func TestCleanCommandInjectsConfigLoader(t *testing.T) {
	wantErr := errors.New("config unavailable")
	command := newCleanCommand(cleanDependencies{
		LoadConfig: func() (*config.Config, error) { return nil, wantErr },
	})
	if err := command.Execute(nil); !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want injected config error", err)
	}
}

func TestInfoCommandInjectsConfigLoader(t *testing.T) {
	wantErr := errors.New("config unavailable")
	command := newInfoCommand(infoDependencies{
		LoadConfig: func() (*config.Config, error) { return nil, wantErr },
	})
	if err := command.Execute([]string{"owner/repo"}); !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want injected config error", err)
	}
}

func TestListCommandInjectsPackageDirectoryLookup(t *testing.T) {
	wantErr := errors.New("paths unavailable")
	command := newListCommand(listDependencies{
		GetPackagesDir: func() (string, error) { return "", wantErr },
	})
	if err := command.Execute(nil); !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want injected path error", err)
	}
}

func TestInitCommandInjectsConfigGeneration(t *testing.T) {
	wantErr := errors.New("write failed")
	command := newInitCommand(initDependencies{
		GenerateConfig: func() error { return wantErr },
		GetPaths: func() (*platform.Paths, error) {
			return &platform.Paths{}, nil
		},
	})
	if err := command.Execute(nil); !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want injected generation error", err)
	}
}

func TestInfoCommandRequiresExactlyOnePackage(t *testing.T) {
	for _, args := range [][]string{{"info"}, {"info", "owner/repo", "extra"}} {
		if err := ExecuteArgs(args); err == nil {
			t.Fatalf("expected info %v to reject invalid argument count", args)
		}
	}
}
