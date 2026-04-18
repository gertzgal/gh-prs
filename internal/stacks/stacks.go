package stacks

import (
	"fmt"

	"github.com/gertzgal/gh-prs/internal/model"
)

type Node struct {
	PR    model.PR
	Child *Node
}

type Grouped struct {
	Stacks     []*Node
	Standalone []model.PR
}

func Group(prs []model.PR, defaultBranch string) Grouped {
	childrenOf := make(map[string][]model.PR)
	for _, pr := range prs {
		childrenOf[pr.BaseRefName] = append(childrenOf[pr.BaseRefName], pr)
	}

	var walk func(pr model.PR) *Node
	walk = func(pr model.PR) *Node {
		bucket := childrenOf[pr.HeadRefName]
		if len(bucket) == 0 {
			return &Node{PR: pr, Child: nil}
		}
		return &Node{PR: pr, Child: walk(bucket[0])}
	}

	var stacks []*Node
	var standalone []model.PR
	for _, pr := range prs {
		if pr.BaseRefName != defaultBranch {
			continue
		}
		if len(childrenOf[pr.HeadRefName]) > 0 {
			stacks = append(stacks, walk(pr))
		} else {
			standalone = append(standalone, pr)
		}
	}
	return Grouped{Stacks: stacks, Standalone: standalone}
}

// Annotate returns a new slice of PRs with StackID and StackPos populated for
// every PR that belongs to a stack. Standalone PRs (and orphans excluded from
// Group) keep nil pointers. The input slice is not mutated.
//
// Stack IDs are 1-based, assigned in the order Group returns stacks (stable).
// StackPos is the 1-based "i/N" position within the stack (e.g. "2/3").
func Annotate(prs []model.PR, defaultBranch string) []model.PR {
	grouped := Group(prs, defaultBranch)

	type ann struct {
		id  int
		pos string
	}
	byNumber := make(map[int]ann)
	for i, root := range grouped.Stacks {
		stackID := i + 1
		members := flatten(root)
		n := len(members)
		for idx, m := range members {
			byNumber[m.Number] = ann{
				id:  stackID,
				pos: fmt.Sprintf("%d/%d", idx+1, n),
			}
		}
	}

	out := make([]model.PR, len(prs))
	for i, p := range prs {
		out[i] = p
		if a, ok := byNumber[p.Number]; ok {
			id := a.id
			pos := a.pos
			out[i].StackID = &id
			out[i].StackPos = &pos
		}
	}
	return out
}

func flatten(n *Node) []model.PR {
	var out []model.PR
	for cur := n; cur != nil; cur = cur.Child {
		out = append(out, cur.PR)
	}
	return out
}
