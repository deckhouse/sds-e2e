### Run tests on Bare Metal Server

1. Clone git repository from https://github.com/deckhouse/sds-e2e
    - `git clone <url>`
    - `cd sds-e2e/testkit_v2`
2. Init config data
    - `mkdir ../../sds-e2e-cfg`
    - `export hv_ssh_dst="<user>@<host>"`
    - `ssh -t $hv_ssh_dst "sudo cat /root/.kube/config > kube-hypervisor.config"`
    - `scp $hv_ssh_dst:kube-hypervisor.config ../../sds-e2e-cfg/`
3. Run test
    - `export licensekey="<deckhouse license key>"`
    - `export hv_ssh_dst="<user>@<host>"`
    - `export hv_ssh_key=<key path>`
    - `./e2e_test.sh Metal --ssh-host $hv_ssh_dst --ssh-key $hv_ssh_key --hypervisor-kconfig kube-hypervisor.config -v -d --ns te2est-2025 tests/01_sds_nc_test.go`
