package main

import "testing"

func TestQuickEntryTargetUpdatesCategory(t *testing.T) {
	if got, want := quickEntryTarget("spring"), "/quick-entry?category=spring"; got != want {
		t.Fatalf("quickEntryTarget(spring) = %q, want %q", got, want)
	}
	if got, want := quickEntryTarget("all"), "/quick-entry?category=all"; got != want {
		t.Fatalf("quickEntryTarget(all) = %q, want %q", got, want)
	}
}
