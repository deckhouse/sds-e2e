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

import "testing"

type TestNode struct {
	Id        int
	Name      string
	GroupName string
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

		for i, nName := range nodes {
			Debugf("Run %s/%s test", label, nName)
			tn := TestNode{Id: i, Name: nName, GroupName: label}
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

			for i, nName := range nodes {
				t.Run(nName, func(t *testing.T) {
					if Parallel {
						t.Parallel()
					}
					tn := TestNode{Id: i, Name: nName, GroupName: label}
					f(&T{t, &tn})
				})
			}
		})
	}
}
