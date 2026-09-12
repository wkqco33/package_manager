package cmd

import (
	"errors"
	"strings"
	"testing"

	"github.com/wkqco33/package_manager/internal/config"
	"github.com/wkqco33/package_manager/internal/pkg"
	"github.com/wkqco33/package_manager/internal/platform"
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

func TestConfigSetCommandUpdatesAndSavesValue(t *testing.T) {
	cfg := &config.Config{RegistryURL: "old", InstallPath: "/bin"}
	var saved *config.Config
	command := newConfigCommand(configDependencies{
		LoadConfig: func() (*config.Config, error) { return cfg, nil },
		SaveConfig: func(value *config.Config) error {
			saved = value
			return nil
		},
		SetValue: config.SetValue,
	})
	if err := command.Execute([]string{"set", "registry_url", "https://registry.example.com"}); err != nil {
		t.Fatalf("config set error = %v", err)
	}
	if saved == nil || saved.RegistryURL != "https://registry.example.com" {
		t.Fatalf("saved config = %#v, want updated registry URL", saved)
	}
	if saved.InstallPath != "/bin" {
		t.Errorf("InstallPath = %q, want existing value preserved", saved.InstallPath)
	}
}

func TestConfigSetCreatesDefaultWhenConfigIsMissing(t *testing.T) {
	want := &config.Config{RegistryURL: "default", InstallPath: "/bin"}
	var saved *config.Config
	command := newConfigCommand(configDependencies{
		LoadConfig:    func() (*config.Config, error) { return nil, config.ErrConfigNotFound },
		DefaultConfig: func() (*config.Config, error) { return want, nil },
		SaveConfig: func(value *config.Config) error {
			saved = value
			return nil
		},
		SetValue: config.SetValue,
	})
	if err := command.Execute([]string{"set", "auth_token", "new-token"}); err != nil {
		t.Fatalf("config set error = %v", err)
	}
	if saved == nil || saved.AuthToken != "new-token" {
		t.Fatalf("saved config = %#v, want token in default config", saved)
	}
}

func TestConfigSetRequiresKeyAndValue(t *testing.T) {
	command := newConfigCommand(configDependencies{})
	if err := command.Execute([]string{"set", "auth_token"}); err == nil {
		t.Fatal("expected config set to reject missing value")
	}
}

func TestInfoCommandRequiresExactlyOnePackage(t *testing.T) {
	for _, args := range [][]string{{"info"}, {"info", "owner/repo", "extra"}} {
		if err := ExecuteArgs(args); err == nil {
			t.Fatalf("expected info %v to reject invalid argument count", args)
		}
	}
}

func TestUninstallRequiresConfirmationBeforeRemoval(t *testing.T) {
	confirmed := false
	removed := false
	command := newUninstallCommand(uninstallDependencies{
		LoadConfig: func() (*config.Config, error) {
			return &config.Config{InstallPath: t.TempDir()}, nil
		},
		Confirm: func() error {
			confirmed = true
			return errors.New("declined")
		},
		Remove: func(string, string) error {
			removed = true
			return nil
		},
	})

	if err := command.Execute([]string{"owner/repo"}); err == nil {
		t.Fatal("expected declined confirmation error")
	}
	if !confirmed {
		t.Fatal("expected confirmation to be requested")
	}
	if removed {
		t.Fatal("remove operation must not run after declined confirmation")
	}
}

func TestCleanAllRequiresConfirmation(t *testing.T) {
	oldAll, oldConfirm := cleanAll, confirmDestructiveAction
	t.Cleanup(func() {
		cleanAll = oldAll
		confirmDestructiveAction = oldConfirm
	})
	confirmDestructiveAction = func(string) error { return errors.New("declined") }

	command := newCleanCommand(cleanDependencies{
		LoadConfig: func() (*config.Config, error) {
			return &config.Config{InstallPath: t.TempDir()}, nil
		},
		GetPackagesDir: func() (string, error) { return t.TempDir(), nil },
	})
	cleanAll = true
	if err := command.Execute(nil); err == nil {
		t.Fatal("expected declined confirmation error")
	}
}

func TestDestructiveConfirmationRequiresExplicitBypassWithoutInput(t *testing.T) {
	oldNoInput := noInputMode
	t.Cleanup(func() { noInputMode = oldNoInput })
	noInputMode = true

	if err := requireDestructiveConfirmation("clean --all", false); err == nil {
		t.Fatal("expected non-interactive confirmation error")
	}
	if err := requireDestructiveConfirmation("clean --all", true); err != nil {
		t.Fatalf("--yes/--force bypass should succeed: %v", err)
	}
}

func TestSelfUpdateRequiresConfirmationWithoutInput(t *testing.T) {
	if err := ExecuteArgs([]string{"--no-input", "self-update"}); err == nil {
		t.Fatal("self-update must require explicit confirmation")
	}
}

func TestConfigSetPasswordStdin(t *testing.T) {
	cfg := &config.Config{}
	var saved *config.Config
	command := newConfigCommand(configDependencies{
		LoadConfig:    func() (*config.Config, error) { return cfg, nil },
		SaveConfig:    func(value *config.Config) error { saved = value; return nil },
		SetValue:      config.SetValue,
		PasswordStdin: strings.NewReader("token-from-stdin\n"),
	})
	if err := command.Execute([]string{"set", "auth_token", "--password-stdin"}); err != nil {
		t.Fatalf("config set --password-stdin error = %v", err)
	}
	if saved == nil || saved.AuthToken != "token-from-stdin" {
		t.Fatalf("saved token = %#v, want token-from-stdin", saved)
	}
}

func TestConfigSetPasswordFile(t *testing.T) {
	cfg := &config.Config{}
	var saved *config.Config
	command := newConfigCommand(configDependencies{
		LoadConfig: func() (*config.Config, error) { return cfg, nil },
		SaveConfig: func(value *config.Config) error { saved = value; return nil },
		SetValue:   config.SetValue,
		ReadFile: func(string) ([]byte, error) {
			return []byte("token-from-file\n"), nil
		},
	})
	if err := command.Execute([]string{"set", "auth_token", "--password-file", "token.txt"}); err != nil {
		t.Fatalf("config set --password-file error = %v", err)
	}
	if saved == nil || saved.AuthToken != "token-from-file" {
		t.Fatalf("saved token = %#v, want token-from-file", saved)
	}
}

func TestExecuteArgsRejectsUnknownCommand(t *testing.T) {
	if err := ExecuteArgs([]string{"does-not-exist"}); err == nil {
		t.Fatal("unknown command must return a non-zero error")
	}
	if err := ExecuteArgs([]string{"config", "does-not-exist"}); err == nil {
		t.Fatal("unknown subcommand must return a non-zero error")
	}
}
