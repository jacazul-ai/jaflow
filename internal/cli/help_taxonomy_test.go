package cli

import "testing"

func TestHelpTaxonomyHasUniqueCommandsAndValidAliases(t *testing.T) {
	groups := make(map[string]struct{})
	for _, group := range helpGroupOrder() {
		if _, exists := groups[group]; exists {
			t.Fatalf("help group %q is listed more than once", group)
		}
		groups[group] = struct{}{}
	}

	entries := make(map[string]helpEntry, len(helpEntries))
	for _, entry := range helpEntries {
		if entry.name == "" {
			t.Fatal("help entry has no name")
		}
		if _, exists := entries[entry.name]; exists {
			t.Fatalf("help command %q is duplicated", entry.name)
		}
		entries[entry.name] = entry
		if _, exists := groups[entry.group]; !exists {
			t.Fatalf("help command %q has unknown group %q", entry.name, entry.group)
		}
		if entry.canonical == "" {
			t.Fatalf("help command %q has no canonical command", entry.name)
		}
	}

	for _, entry := range helpEntries {
		canonical, exists := entries[entry.canonical]
		if !exists {
			t.Fatalf("help command %q points to missing canonical %q", entry.name, entry.canonical)
		}
		if entry.hidden && canonical.hidden {
			t.Fatalf("alias %q points to hidden canonical %q", entry.name, entry.canonical)
		}
		if !entry.hidden && entry.name != entry.canonical {
			t.Fatalf("visible command %q points to different canonical %q", entry.name, entry.canonical)
		}
		if entry.group != canonical.group {
			t.Fatalf("command %q group %q differs from canonical %q group %q", entry.name, entry.group, entry.canonical, canonical.group)
		}
	}
}

func TestHelpGroupOrderIsStable(t *testing.T) {
	want := []string{
		groupStartAndOrganize,
		groupExamineState,
		groupWorkCurrentTask,
		groupChangeReprioritize,
		groupPreserveContext,
		groupMaintainDerived,
		groupPrepareIntegrate,
		groupMigrateLegacy,
	}
	got := helpGroupOrder()
	if len(got) != len(want) {
		t.Fatalf("help groups = %v, want %v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("help group %d = %q, want %q", index, got[index], want[index])
		}
	}
}
