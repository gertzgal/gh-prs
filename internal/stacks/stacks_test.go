package stacks_test

import (
	"testing"

	"github.com/gertzgal/gh-prs/internal/model"
	"github.com/gertzgal/gh-prs/internal/stacks"
)

func pr(overrides model.PR) model.PR {
	base := model.PR{
		Number:           1,
		Title:            "t",
		URL:              "u",
		IsDraft:          false,
		HeadRefName:      "h",
		BaseRefName:      "main",
		Additions:        0,
		Deletions:        0,
		ChangedFiles:     0,
		MergeStateStatus: "CLEAN",
	}
	if overrides.Number != 0 {
		base.Number = overrides.Number
	}
	if overrides.Title != "" {
		base.Title = overrides.Title
	}
	if overrides.URL != "" {
		base.URL = overrides.URL
	}
	if overrides.HeadRefName != "" {
		base.HeadRefName = overrides.HeadRefName
	}
	if overrides.BaseRefName != "" {
		base.BaseRefName = overrides.BaseRefName
	}
	base.IsDraft = overrides.IsDraft
	base.Additions = overrides.Additions
	base.Deletions = overrides.Deletions
	base.ChangedFiles = overrides.ChangedFiles
	base.ReviewDecision = overrides.ReviewDecision
	base.CiState = overrides.CiState
	if overrides.MergeStateStatus != "" {
		base.MergeStateStatus = overrides.MergeStateStatus
	}
	return base
}

func TestGroup_Empty(t *testing.T) {
	got := stacks.Group(nil, "main")
	if len(got.Stacks) != 0 {
		t.Errorf("want no stacks, got %d", len(got.Stacks))
	}
	if len(got.Standalone) != 0 {
		t.Errorf("want no standalone, got %d", len(got.Standalone))
	}
}

func TestGroup_SingleStandalone(t *testing.T) {
	solo := pr(model.PR{Number: 1, HeadRefName: "feat-a", BaseRefName: "main"})
	got := stacks.Group([]model.PR{solo}, "main")

	if len(got.Stacks) != 0 {
		t.Errorf("want 0 stacks, got %d", len(got.Stacks))
	}
	if len(got.Standalone) != 1 {
		t.Fatalf("want 1 standalone, got %d", len(got.Standalone))
	}
	if got.Standalone[0].Number != solo.Number {
		t.Errorf("want standalone #%d, got #%d", solo.Number, got.Standalone[0].Number)
	}
}

func TestGroup_Linear4Stack(t *testing.T) {
	a := pr(model.PR{Number: 1, HeadRefName: "a", BaseRefName: "main"})
	b := pr(model.PR{Number: 2, HeadRefName: "b", BaseRefName: "a"})
	c := pr(model.PR{Number: 3, HeadRefName: "c", BaseRefName: "b"})
	d := pr(model.PR{Number: 4, HeadRefName: "d", BaseRefName: "c"})

	got := stacks.Group([]model.PR{a, b, c, d}, "main")

	if len(got.Standalone) != 0 {
		t.Errorf("want 0 standalone, got %d", len(got.Standalone))
	}
	if len(got.Stacks) != 1 {
		t.Fatalf("want 1 stack, got %d", len(got.Stacks))
	}

	root := got.Stacks[0]
	expect := []int{1, 2, 3, 4}
	node := root
	for i, want := range expect {
		if node == nil {
			t.Fatalf("node %d is nil", i)
		}
		if node.PR.Number != want {
			t.Errorf("node %d: want #%d, got #%d", i, want, node.PR.Number)
		}
		node = node.Child
	}
	if node != nil {
		t.Errorf("want nil tip, got #%d", node.PR.Number)
	}
}

func TestGroup_TwoIndependent2Stacks(t *testing.T) {
	a1 := pr(model.PR{Number: 1, HeadRefName: "a1", BaseRefName: "main"})
	a2 := pr(model.PR{Number: 2, HeadRefName: "a2", BaseRefName: "a1"})
	b1 := pr(model.PR{Number: 3, HeadRefName: "b1", BaseRefName: "main"})
	b2 := pr(model.PR{Number: 4, HeadRefName: "b2", BaseRefName: "b1"})

	got := stacks.Group([]model.PR{a1, a2, b1, b2}, "main")

	if len(got.Standalone) != 0 {
		t.Errorf("want 0 standalone, got %d", len(got.Standalone))
	}
	if len(got.Stacks) != 2 {
		t.Fatalf("want 2 stacks, got %d", len(got.Stacks))
	}

	first := got.Stacks[0]
	if first.PR.Number != 1 {
		t.Errorf("first stack root: want #1, got #%d", first.PR.Number)
	}
	if first.Child == nil || first.Child.PR.Number != 2 {
		t.Errorf("first stack child: want #2, got %v", first.Child)
	}
	if first.Child != nil && first.Child.Child != nil {
		t.Errorf("first stack should terminate at depth 2")
	}

	second := got.Stacks[1]
	if second.PR.Number != 3 {
		t.Errorf("second stack root: want #3, got #%d", second.PR.Number)
	}
	if second.Child == nil || second.Child.PR.Number != 4 {
		t.Errorf("second stack child: want #4, got %v", second.Child)
	}
	if second.Child != nil && second.Child.Child != nil {
		t.Errorf("second stack should terminate at depth 2")
	}
}

func TestGroup_MultiChildPick_FirstOnly(t *testing.T) {
	root := pr(model.PR{Number: 1, HeadRefName: "root", BaseRefName: "main"})
	firstChild := pr(model.PR{Number: 2, HeadRefName: "first", BaseRefName: "root"})
	secondChild := pr(model.PR{Number: 3, HeadRefName: "second", BaseRefName: "root"})

	got := stacks.Group([]model.PR{root, firstChild, secondChild}, "main")

	if len(got.Stacks) != 1 {
		t.Fatalf("want 1 stack, got %d", len(got.Stacks))
	}
	rootNode := got.Stacks[0]
	if rootNode.PR.Number != 1 {
		t.Errorf("want root #1, got #%d", rootNode.PR.Number)
	}
	if rootNode.Child == nil || rootNode.Child.PR.Number != firstChild.Number {
		t.Errorf("want first-child #%d, got %v", firstChild.Number, rootNode.Child)
	}

	var seen []int
	var collect func(n *stacks.Node)
	collect = func(n *stacks.Node) {
		if n == nil {
			return
		}
		seen = append(seen, n.PR.Number)
		collect(n.Child)
	}
	for _, s := range got.Stacks {
		collect(s)
	}
	for _, n := range seen {
		if n == secondChild.Number {
			t.Errorf("secondChild #%d should be absent from stacks", secondChild.Number)
		}
	}
	for _, p := range got.Standalone {
		if p.Number == secondChild.Number {
			t.Errorf("secondChild #%d should be absent from standalone", secondChild.Number)
		}
	}
}

func TestGroup_OrphanExcluded(t *testing.T) {
	orphan := pr(model.PR{Number: 1, HeadRefName: "orphan", BaseRefName: "feature-x"})
	got := stacks.Group([]model.PR{orphan}, "main")

	if len(got.Stacks) != 0 {
		t.Errorf("want 0 stacks, got %d", len(got.Stacks))
	}
	if len(got.Standalone) != 0 {
		t.Errorf("want 0 standalone, got %d", len(got.Standalone))
	}
}
