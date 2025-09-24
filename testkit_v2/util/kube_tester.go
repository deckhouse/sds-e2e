/*
Copyright 2025 Flant JSC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
	t.Fatal(args...)
}

func (t *T) Skipf(format string, args ...any) {
	if SkipOptional {
		Warnf(format, args...)
		t.T.Skipf(format, args...)
	}
	t.Fatalf(format, args...)
}

func (cluster *KCluster) RunTestGroupNodes(t *testing.T, label any, f func(t *T), filters ...NodeFilter) {
	if TreeMode {
		cluster.RunTestTreeGroupNodes(t, label, f, filters...)
		return
	}

	for label, nodes := range cluster.MapLabelNodes(label, filters...) {
		Infof("%d Nodes for label '%s'", len(nodes), label)
		if len(nodes) == 0 && !SkipOptional {
			t.Errorf("no Nodes for label '%s'", label)
			continue
		}

		for i, node := range nodes {
			Debugf("Run %s/%s test", label, node.Name)
			tn := TestNode{Id: i, Name: node.Name, GroupName: label, Raw: &node}
			f(&T{t, &tn})
		}
		t.Logf("'%s' tests count: %d", label, len(nodes))
	}
}

func (cluster *KCluster) RunTestTreeGroupNodes(t *testing.T, label any, f func(t *T), filters ...NodeFilter) {
	for label, nodes := range cluster.MapLabelNodes(label, filters...) {
		t.Run(label, func(t *testing.T) {
			if Parallel {
				t.Parallel()
			}
			Infof("%d Nodes for label '%s'", len(nodes), label)
			if len(nodes) == 0 {
				if SkipOptional {
					t.SkipNow()
				}
				t.Fatalf("no Nodes for label '%s'", label)
			}

			for i, node := range nodes {
				t.Run(node.Name, func(t *testing.T) {
					if Parallel {
						t.Parallel()
					}
					tn := TestNode{Id: i, Name: node.Name, GroupName: label, Raw: &node}
					f(&T{t, &tn})
				})
			}
		})
	}
}
