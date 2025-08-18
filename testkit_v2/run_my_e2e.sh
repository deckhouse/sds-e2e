export hv_ssh_dst="ubuntu@94.26.231.181"
export licensekey="sFk1BZvFobocvhQVhLcYHUmNUggH7ou2"
go test -v -timeout 30m ./tests/tools.go $@ -debug -sshhost $hv_ssh_dst -hypervisorkconfig kube-hypervisor.config -skipoptional -keepstate -clustertype "Ubuntu 22 mini" -namespace test-e2e-pupkin
