package integration

import "sync"

var clrCache = map[string]*KCluster{}
var mx sync.RWMutex

func GetCluster(configPath, clusterName string) *KCluster {
	mx.Lock()
	if len(clrCache) == 0 {
		envInit()
		if HypervisorKubeConfig != "" {
			ClusterCreate()
		} else {
			NestedSshClient = GetSshClient(NestedSshUser, NestedHost+":22", NestedSshKey)
			go NestedSshClient.NewTunnel("127.0.0.1:"+NestedDhPort, "127.0.0.1:"+NestedDhPort)
		}
	}

	k := configPath + ":" + clusterName
	if _, ok := clrCache[k]; !ok {
		clr, err := InitKCluster(configPath, clusterName)
		if err != nil {
			Critf("Kubeclient '%s' problem", k)
			panic(err)
		}
		_ = clr.CreateNs(TestNS)
		clrCache[k] = clr
	}

	mx.Unlock()
	return clrCache[k]
}
