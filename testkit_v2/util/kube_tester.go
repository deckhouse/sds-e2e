package integration

import (
	"testing"

	coreapi "k8s.io/api/core/v1"
)

type TestNode struct {
	Id        int
	Name      string
	GroupName string
	Raw       *coreapi.Node
}

type T struct {
	*testing.T
	Node *TestNode
}

func (t *T) Skip(args ...any) {
	if SkipOptional {
		Warn(args...)
		t.T.Skip(args...)
	}
	t.T.Fatal(args...)
}

func (t *T) Skipf(format string, args ...any) {
	if SkipOptional {
		Warnf(format, args...)
		t.T.Skipf(format, args...)
	}
	t.T.Fatalf(format, args...)
}

func (clr *KCluster) RunTestGroupNodes(t *testing.T, label any, f func(t *T), filters ...NodeFilter) {
	if TreeMode {
		clr.RunTestTreeGroupNodes(t, label, f, filters...)
		return
	}

	for label, nodes := range clr.MapLabelNodes(label, filters...) {
		Infof("%d Nodes for label '%s'", len(nodes), label)
		if len(nodes) == 0 && !SkipOptional {
			t.Errorf("no Nodes for label '%s'", label)
			continue
		}

		for i, node := range nodes {
			Debugf("Run %s/%s test", label, node.ObjectMeta.Name)
			tn := TestNode{Id: i, Name: node.ObjectMeta.Name, GroupName: label, Raw: &node}
			f(&T{t, &tn})
		}
		t.Logf("'%s' tests count: %d", label, len(nodes))
	}
}

func (clr *KCluster) RunTestTreeGroupNodes(t *testing.T, label any, f func(t *T), filters ...NodeFilter) {
	for label, nodes := range clr.MapLabelNodes(label, filters...) {
		t.Run(label, func(t *testing.T) {
			if Parallel {
				t.Parallel()
			}
			Infof("%d Nodes for label '%s'", len(nodes), label)
			if len(nodes) == 0 {
				if SkipOptional {
					t.Skipf("no Nodes for label '%s'", label)
				}
				t.Fatalf("no Nodes for label '%s'", label)
			}

			for i, node := range nodes {
				t.Run(node.ObjectMeta.Name, func(t *testing.T) {
					if Parallel {
						t.Parallel()
					}
					tn := TestNode{Id: i, Name: node.ObjectMeta.Name, GroupName: label, Raw: &node}
					f(&T{t, &tn})
				})
			}
		})
	}
}
