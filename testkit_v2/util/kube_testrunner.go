package integration

import (
	"testing"
)


type TestNode struct {
	Id int
	Name string
	GroupName string
}

type tRunner struct {
	clr *KCluster
	t *testing.T
}

func (r tRunner) PerGroupNode(f func(t *testing.T, node TestNode)) {
	for group, nodes := range r.clr.GetGroupNodes() {
		r.t.Run(group, func(t *testing.T) {
			if len(nodes) == 0 && !SkipOptional {
				t.Fatal("no Nodes for group", group)
			}
			for i, nName := range nodes {
				t.Run(nName, func(t *testing.T) {
					node := TestNode{Id: i, GroupName: group, Name: nName}
					f(t, node)
				})
			}
		})
	}
}

func (r tRunner) GroupNodeParallel(f func(node TestNode)) {
				//t.Parallel()
}

func (r tRunner) GroupParallel(f func(node TestNode)) {
				//t.Parallel()
}
