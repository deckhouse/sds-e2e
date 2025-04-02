## Setup test
  ```bash
  git clone <url>
  cd sds-e2e/testkit_v2
  ```
#### Init config data for Bare Metal Server
  ```bash
  export hv_ssh_dst="<user>@<host>"
  export hv_ssh_key="<key path>"
  export licensekey="<deckhouse license key>"

  mkdir ../../sds-e2e-cfg
  ssh -t $hv_ssh_dst "sudo cat /root/.kube/config > kube-hypervisor.config"
  scp $hv_ssh_dst:kube-hypervisor.config ../../sds-e2e-cfg/
  ```

## Run tests with e2e_test.sh
  Manual and examples in `e2e_test.sh --help`

## Run tests with go test
  `go test [FLAG]... PATH [FLAG|OPTION]...`

  **Flags:**
  &nbsp; `-v`
  &nbsp; `-run TestOk/case1`
  &nbsp; `-skip TestFatal/ignore`
  &nbsp; `-timeout 30m`
  &nbsp; `-parallel N`

  **Options:**
  &nbsp; `-verbose`
  &nbsp; &nbsp; &nbsp; Output with Info messages
  &nbsp; `-debug`
  &nbsp; &nbsp; &nbsp; Output with Debug messages
  &nbsp; `-skipoptional`
  &nbsp; &nbsp; &nbsp; Skip optional tests (no required resources)
  &nbsp; `-notparallel`
  &nbsp; &nbsp; &nbsp; Run test groups in single mode
  &nbsp; `-tree`
  &nbsp; &nbsp; &nbsp; Run tests in tree mode. Can be turned on in <u>-notparallel</u> mode

  &nbsp; `-stand (lockal|dev|metal|stage|ci)`
  &nbsp; &nbsp; &nbsp; Test stand class
  &nbsp; `-sshhost user@127.0.0.1`
  &nbsp; &nbsp; &nbsp; Test ssh host. Hypervisor ssh host if <u>-hypervisorkconfig</u> used
  &nbsp; `-sshkey /home/user/.ssh/id_rsa`
  &nbsp; &nbsp; &nbsp; Test ssh key
  &nbsp; `-kconfig kube-nested.config`
  &nbsp; &nbsp; &nbsp; The k8s config path for test
  &nbsp; `-hypervisorkconfig kube-hypervisor.config`
  &nbsp; &nbsp; &nbsp; The k8s config path for hypervisor (virtualization administration)
  &nbsp; &nbsp; &nbsp; For virtual stand generation requaered option <u>-timeout 30m</u>
  &nbsp; &nbsp; &nbsp; and deckhouse license key <u>export licensekey="..."</u>
  &nbsp; `-namespace 01-01-test`
  &nbsp; &nbsp; &nbsp; Test name space

  **Examples:**
  &nbsp; `go test -v ./tests/... -verbose`
  &nbsp; &nbsp; &nbsp; Run all tests on localhost with medium real-time output
  &nbsp; `go test -v -timeout 30m ./tests/01_first_test.go -verbose -debug -hypervisorkconfig kube-hypervisor.config -sshhost 10.20.30.40 -namespace 01-01-test`
  &nbsp; &nbsp; &nbsp; Run test unit with cluster generation on hypervisor
  &nbsp; `go test -v -timeout 30m ./tests/... -verbose -debug -hypervisorkconfig kube-hypervisor.config -sshhost user@10.20.30.40 -namespace 01-01-test -run TestOk/case1`
  &nbsp; &nbsp; &nbsp; Run test case with cluster generation on hypervisor

  > :bulb: You can prepare run command with alias
  > `alias e2e_test_hv='go test -v -timeout 30m ./tests/... -verbose -debug -hypervisorkconfig kube-hypervisor.config -sshhost user@10.20.30.40 -namespace 01-01-test'`
  > or script
  > `echo "go test -v -timeout 30m ./tests/... -verbose -debug -hypervisorkconfig kube-hypervisor.config -sshhost user@10.20.30.40 -namespace 01-01-test" > e2e_test_hv.sh`
