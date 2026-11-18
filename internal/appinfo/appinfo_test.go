package appinfo

import "testing"

func TestApplyUpdatesMetadata(t *testing.T) {
	before := Get()
	t.Cleanup(func() {
		current.Store(&before)
	})

	Apply(Info{
		Version:         "9.9",
		Name:            "TestApp",
		Title:           "Test Title",
		DefaultLanguage: "en-US",
	})

	got := Get()
	if got.Version != "9.9" || got.Name != "TestApp" || got.Title != "Test Title" {
		t.Fatalf("unexpected metadata: %+v", got)
	}
	if got.DefaultLanguage != "en" {
		t.Fatalf("default language = %q, want en", got.DefaultLanguage)
	}
}

func TestApplyKeepsExistingValuesForBlankFields(t *testing.T) {
	before := Get()
	t.Cleanup(func() {
		current.Store(&before)
	})

	Apply(Info{Version: "2.0", Name: "App", Title: "Title", DefaultLanguage: "ja"})
	Apply(Info{})

	got := Get()
	if got.Version != "2.0" || got.Name != "App" || got.Title != "Title" || got.DefaultLanguage != "ja" {
		t.Fatalf("blank fields should keep existing values, got %+v", got)
	}
}
