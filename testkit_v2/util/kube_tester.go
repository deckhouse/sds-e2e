package integration

import "testing"

type TestNode struct {
	Id        int
	GroupName string
	Name      string
}

func (clr *KCluster) RunTestGroupNodes(t *testing.T, f func(t *testing.T, tNode TestNode)) {
	if *treeFlag {
		clr.RunTestTreeGroupNodes(t, f)
		return
	}

	for label, nodes := range clr.MapLabelNodes(nil) {
		if len(nodes) == 0 && !SkipOptional {
			Errf("0 Nodes for label '%s'", label)
			t.Errorf("no Nodes for label '%s'", label)
			continue
		}

		for i, nName := range nodes {
			Debugf("Run %s/%s test", label, nName)
			tn := TestNode{Id: i, GroupName: label, Name: nName}
			f(t, tn)
		}
		t.Logf("'%s' tests count: %d", label, len(nodes))
	}
}

func (clr *KCluster) RunTestTreeGroupNodes(t *testing.T, f func(t *testing.T, tNode TestNode)) {
	for label, nodes := range clr.MapLabelNodes(nil) {
		t.Run(label, func(t *testing.T) {
			if len(nodes) == 0 && !SkipOptional {
				Errf("0 Nodes for label '%s'", label)
				t.Fatalf("no Nodes for label '%s'", label)
			}

			for i, nName := range nodes {
				t.Run(nName, func(t *testing.T) {
					node := TestNode{Id: i, GroupName: label, Name: nName}
					f(t, node)
				})
			}
		})
	}
}

func (clr *KCluster) RunTestParallelGroupNodes(t *testing.T, f func(t *testing.T, tNode TestNode)) {
	for label, nodes := range clr.MapLabelNodes(nil) {
		t.Run(label, func(t *testing.T) {
			if len(nodes) == 0 && !SkipOptional {
				Errf("0 Nodes for label '%s'", label)
				t.Fatalf("no Nodes for label '%s'", label)
			}

			for i, nName := range nodes {
				t.Run(nName, func(t *testing.T) {
					t.Parallel()
					node := TestNode{Id: i, GroupName: label, Name: nName}
					f(t, node)
				})
			}
		})
	}
}
