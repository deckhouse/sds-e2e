package integration

import (
	"sync"
)

var clrCache = map[string]*KCluster{}
var mx sync.RWMutex

func GetCluster(configPath, clusterName string) *KCluster {
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
		clr, err := InitKCluster(configPath, clusterName)
		if err != nil {
			Fatalf("Kubeclient '%s' problem: %s", k, err.Error())
		}
		_ = clr.CreateNs(TestNS)
		clrCache[k] = clr
	}

	return clrCache[k]
}
