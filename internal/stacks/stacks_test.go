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

func TestAnnotate_MixedStackAndStandalone(t *testing.T) {
	a1 := pr(model.PR{Number: 1, HeadRefName: "a1", BaseRefName: "main"})
	a2 := pr(model.PR{Number: 2, HeadRefName: "a2", BaseRefName: "a1"})
	a3 := pr(model.PR{Number: 3, HeadRefName: "a3", BaseRefName: "a2"})
	b1 := pr(model.PR{Number: 4, HeadRefName: "b1", BaseRefName: "main"})
	b2 := pr(model.PR{Number: 5, HeadRefName: "b2", BaseRefName: "b1"})
	solo := pr(model.PR{Number: 6, HeadRefName: "solo", BaseRefName: "main"})

	in := []model.PR{a1, a2, a3, b1, b2, solo}
	got := stacks.Annotate(in, "main")

	if len(got) != len(in) {
		t.Fatalf("length: got %d, want %d", len(got), len(in))
	}

	byNum := make(map[int]model.PR, len(got))
	for _, p := range got {
		byNum[p.Number] = p
	}

	cases := []struct {
		num       int
		wantStack *int
		wantPos   *string
	}{
		{1, intPtr(1), strPtr("1/3")},
		{2, intPtr(1), strPtr("2/3")},
		{3, intPtr(1), strPtr("3/3")},
		{4, intPtr(2), strPtr("1/2")},
		{5, intPtr(2), strPtr("2/2")},
		{6, nil, nil},
	}
	for _, c := range cases {
		p := byNum[c.num]
		if !equalIntPtr(p.StackID, c.wantStack) {
			t.Errorf("#%d StackID: got %v, want %v", c.num, derefInt(p.StackID), derefInt(c.wantStack))
		}
		if !equalStrPtr(p.StackPos, c.wantPos) {
			t.Errorf("#%d StackPos: got %q, want %q", c.num, derefStr(p.StackPos), derefStr(c.wantPos))
		}
	}
}

func TestAnnotate_DoesNotMutateInput(t *testing.T) {
	a1 := pr(model.PR{Number: 1, HeadRefName: "a1", BaseRefName: "main"})
	a2 := pr(model.PR{Number: 2, HeadRefName: "a2", BaseRefName: "a1"})
	in := []model.PR{a1, a2}

	_ = stacks.Annotate(in, "main")

	if in[0].StackID != nil || in[0].StackPos != nil {
		t.Errorf("input[0] mutated: StackID=%v StackPos=%v", in[0].StackID, in[0].StackPos)
	}
	if in[1].StackID != nil || in[1].StackPos != nil {
		t.Errorf("input[1] mutated: StackID=%v StackPos=%v", in[1].StackID, in[1].StackPos)
	}
}

func TestAnnotate_PreservesOrder(t *testing.T) {
	in := []model.PR{
		pr(model.PR{Number: 10, HeadRefName: "x", BaseRefName: "main"}),
		pr(model.PR{Number: 20, HeadRefName: "y", BaseRefName: "x"}),
		pr(model.PR{Number: 30, HeadRefName: "z", BaseRefName: "main"}),
	}
	got := stacks.Annotate(in, "main")
	for i := range in {
		if got[i].Number != in[i].Number {
			t.Errorf("order changed at %d: got #%d want #%d", i, got[i].Number, in[i].Number)
		}
	}
}

func TestAnnotate_Empty(t *testing.T) {
	got := stacks.Annotate(nil, "main")
	if len(got) != 0 {
		t.Errorf("want empty slice, got %d entries", len(got))
	}
}

func intPtr(i int) *int       { return &i }
func strPtr(s string) *string { return &s }
func derefInt(p *int) any {
	if p == nil {
		return nil
	}
	return *p
}
func derefStr(p *string) any {
	if p == nil {
		return nil
	}
	return *p
}
func equalIntPtr(a, b *int) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}
func equalStrPtr(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}
