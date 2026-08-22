package pkg

import (
	"errors"
	"reflect"
	"testing"
)

type metadataFetcherStub struct {
	packages map[string]*Package
	err      error
	calls    []string
}

func (f *metadataFetcherStub) GetMetadata(name string) (*Package, error) {
	f.calls = append(f.calls, name)
	if f.err != nil {
		return nil, f.err
	}
	return f.packages[name], nil
}

func TestResolveDependenciesOrdersDependenciesAndDeduplicates(t *testing.T) {
	fetcher := &metadataFetcherStub{packages: map[string]*Package{
		"app":    {Name: "app", Dependencies: []string{"shared", "logger"}},
		"worker": {Name: "worker", Dependencies: []string{"shared"}},
		"shared": {Name: "shared"},
		"logger": {Name: "logger", Dependencies: []string{"shared"}},
	}}

	resolved, err := ResolveDependencies([]string{"app", "worker"}, fetcher)
	if err != nil {
		t.Fatalf("ResolveDependencies failed: %v", err)
	}

	got := make([]string, 0, len(resolved))
	for _, p := range resolved {
		got = append(got, p.Name)
	}
	want := []string{"shared", "logger", "app", "worker"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("resolved order = %v, want %v", got, want)
	}
	if !reflect.DeepEqual(fetcher.calls, []string{"app", "shared", "logger", "worker"}) {
		t.Fatalf("metadata calls = %v, want each package once", fetcher.calls)
	}
}

func TestResolveDependenciesRejectsCircularDependency(t *testing.T) {
	fetcher := &metadataFetcherStub{packages: map[string]*Package{
		"a": {Name: "a", Dependencies: []string{"b"}},
		"b": {Name: "b", Dependencies: []string{"a"}},
	}}

	_, err := ResolveDependencies([]string{"a"}, fetcher)
	if err == nil || err.Error() != "circular dependency detected: a" {
		t.Fatalf("error = %v, want circular dependency error", err)
	}
}

func TestResolveDependenciesPropagatesMetadataError(t *testing.T) {
	wantErr := errors.New("registry unavailable")
	fetcher := &metadataFetcherStub{err: wantErr}

	_, err := ResolveDependencies([]string{"app"}, fetcher)
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want wrapped metadata error %v", err, wantErr)
	}
}

func TestResolveDependenciesRejectsNilMetadata(t *testing.T) {
	fetcher := &metadataFetcherStub{packages: map[string]*Package{"app": nil}}

	_, err := ResolveDependencies([]string{"app"}, fetcher)
	if err == nil || err.Error() != "metadata for app is nil" {
		t.Fatalf("error = %v, want nil metadata error", err)
	}
}
