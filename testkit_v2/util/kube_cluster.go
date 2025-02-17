package integration

var clrCache = map[string]*KCluster{}

func GetCluster(configPath, clusterName string) *KCluster {
	if len(clrCache) == 0 {
		envInit()
		if HypervisorKubeConfig != "" {
			ClusterCreate()
		} else {
			client := GetSshClient(NestedSshUser, NestedHost+":22", NestedSshKey)
			go client.NewTunnel("127.0.0.1:"+NestedDhPort, "127.0.0.1:"+NestedDhPort)
		}
	}

	k := configPath + ":" + clusterName
	if _, ok := clrCache[k]; !ok {
		clr, err := InitKCluster(configPath, clusterName)
		if err != nil {
			Critf("Kubeclient '%s' problem", k)
			panic(err)
		}
		clrCache[k] = clr
	}

	return clrCache[k]
}
