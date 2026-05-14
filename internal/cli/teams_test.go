package cli

import "testing"

func TestUnionLogins_PreservesAuthorOrderThenAppendsTeamOnly(t *testing.T) {
	got := unionLogins([]string{"carol", "alice"}, []string{"alice", "bob", "dave"})
	want := []string{"carol", "alice", "bob", "dave"}
	if !equalStrings(got, want) {
		t.Errorf("unionLogins: want %v, got %v", want, got)
	}
}

func TestUnionLogins_DedupesCaseInsensitive(t *testing.T) {
	got := unionLogins([]string{"Alice"}, []string{"alice", "BOB"})
	want := []string{"Alice", "BOB"}
	if !equalStrings(got, want) {
		t.Errorf("unionLogins: want %v, got %v", want, got)
	}
}

func TestUnionLogins_NoAuthorsKeepsTeamOrder(t *testing.T) {
	got := unionLogins(nil, []string{"alice", "bob"})
	want := []string{"alice", "bob"}
	if !equalStrings(got, want) {
		t.Errorf("unionLogins: want %v, got %v", want, got)
	}
}

func TestUnionLogins_NoTeamsKeepsAuthors(t *testing.T) {
	got := unionLogins([]string{"@me"}, nil)
	want := []string{"@me"}
	if !equalStrings(got, want) {
		t.Errorf("unionLogins: want %v, got %v", want, got)
	}
}

func TestUnionLogins_EmptyEmpty(t *testing.T) {
	got := unionLogins(nil, nil)
	if len(got) != 0 {
		t.Errorf("unionLogins(nil,nil): want empty, got %v", got)
	}
}

// ---------------------------------------------------------------------------
// --team flag composition (mirrors --author tests)
// ---------------------------------------------------------------------------

func TestComposeFlags_TeamDefault_NilWhenNotSet(t *testing.T) {
	got := composeFlags("", false, false, "", false, nil, nil, nil, map[string]string{})
	if len(got.Teams) != 0 {
		t.Errorf("Teams: want empty, got %v", got.Teams)
	}
}

func TestComposeFlags_TeamFromFlag(t *testing.T) {
	got := composeFlags("", false, false, "", false, nil, []string{"widgets", "Gadgets"}, nil, map[string]string{})
	if len(got.Teams) != 2 || got.Teams[0] != "widgets" || got.Teams[1] != "Gadgets" {
		t.Errorf("Teams: want [widgets Gadgets], got %v", got.Teams)
	}
}

func TestComposeFlags_TeamFromEnv_CommaSeparated(t *testing.T) {
	got := composeFlags("", false, false, "", false, nil, nil, nil, map[string]string{"GH_PRS_TEAM": "widgets, gadgets"})
	if len(got.Teams) != 2 || got.Teams[0] != "widgets" || got.Teams[1] != "gadgets" {
		t.Errorf("Teams: want [widgets gadgets] from env, got %v", got.Teams)
	}
}

func TestComposeFlags_TeamFlagWinsOverEnv(t *testing.T) {
	got := composeFlags("", false, false, "", false, nil, []string{"sprockets"}, nil, map[string]string{"GH_PRS_TEAM": "widgets,gadgets"})
	if len(got.Teams) != 1 || got.Teams[0] != "sprockets" {
		t.Errorf("Teams: flag should win, want [sprockets], got %v", got.Teams)
	}
}

// ---------------------------------------------------------------------------
// pruneExcluded
// ---------------------------------------------------------------------------

func TestPruneExcluded_Empty_Passthrough(t *testing.T) {
	got := pruneExcluded([]string{"alice", "bob"}, nil)
	if !equalStrings(got, []string{"alice", "bob"}) {
		t.Errorf("want passthrough, got %v", got)
	}
}

func TestPruneExcluded_DropsMatching_PreservesOrder(t *testing.T) {
	got := pruneExcluded([]string{"alice", "bob", "carol"}, []string{"bob"})
	if !equalStrings(got, []string{"alice", "carol"}) {
		t.Errorf("want [alice carol], got %v", got)
	}
}

func TestPruneExcluded_CaseInsensitive(t *testing.T) {
	got := pruneExcluded([]string{"Alice", "BOB"}, []string{"alice"})
	if !equalStrings(got, []string{"BOB"}) {
		t.Errorf("want [BOB], got %v", got)
	}
}

func TestPruneExcluded_NoOverlap_Unchanged(t *testing.T) {
	got := pruneExcluded([]string{"alice"}, []string{"bob"})
	if !equalStrings(got, []string{"alice"}) {
		t.Errorf("want [alice], got %v", got)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
