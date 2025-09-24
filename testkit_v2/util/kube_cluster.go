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
	"sync"
)

var clrCache = map[string]*KCluster{}
var mx = new(sync.RWMutex)

// EnsureCluster creates valid cluster if it does not exist. Check and modify existing cluster if needed. Returns cluster that can be used for tests.
func EnsureCluster(configPath, clusterName string) *KCluster {
	mx.Lock()
	defer mx.Unlock()

	if len(clrCache) == 0 {
		envInit()
		if HypervisorKubeConfig != "" {
			ClusterCreate()
		} else {
			NestedSshClient = GetSshClient(NestedSshUser, NestedHost+":22", NestedSshKey)
			go NestedSshClient.NewTunnel("127.0.0.1:"+NestedK8sPort, "127.0.0.1:"+NestedK8sPort)
		}
	}

	k := configPath + ":" + clusterName
	if _, ok := clrCache[k]; !ok {
		cluster, err := InitKCluster(configPath, clusterName)
		if err != nil {
			Fatalf("Kubeclient '%s' problem: %s", k, err.Error())
		}
		_ = cluster.CreateNs(TestNS)
		clrCache[k] = cluster
	}

	return clrCache[k]
}
