package stacks

import "github.com/gertzgal/gh-prs/internal/model"

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
