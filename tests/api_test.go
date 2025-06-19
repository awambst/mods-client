package tests

import (
  "testing"
	api "mod-installer/api"
)

func TestFetchAllModMeta(t *testing.T) {
	mods, err := api.FetchAllModMeta()
	if err != nil {
		t.Fatal(err)
	}

	for game, modsByGame := range mods {
		t.Logf("Game: %s", game)
		for mod, versions := range modsByGame {
			t.Logf("  Mod: %s (%d versions)", mod, len(versions))
		}
	}
}

