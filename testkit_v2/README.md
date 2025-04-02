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

  `-v`

  `-run TestOk/case1`

  `-skip TestFatal/ignore`

  `-timeout 30m`

  `-parallel N`

  **Options:**

  `-verbose`

  &nbsp; &nbsp; Output with Info messages

  `-debug`

  &nbsp; &nbsp; Output with Debug messages

  `-skipoptional`

  &nbsp; &nbsp; Skip optional tests (no required resources)

  `-notparallel`

  &nbsp; &nbsp; Run test groups in single mode

  `-tree`

  &nbsp; &nbsp; Run tests in tree mode. Can be turned on in <ins>-notparallel</ins> mode

  `-stand (lockal|dev|metal|stage|ci)`

  &nbsp; &nbsp; Test stand class

  `-sshhost user@127.0.0.1`

  &nbsp; &nbsp; Test ssh host. Hypervisor ssh host if <ins>-hypervisorkconfig</ins> used

  `-sshkey /home/user/.ssh/id_rsa`

  &nbsp; &nbsp; Test ssh key

  `-kconfig kube-nested.config`

  &nbsp; &nbsp; The k8s config path for test

  `-hypervisorkconfig kube-hypervisor.config`

  &nbsp; &nbsp; The k8s config path for hypervisor (virtualization administration)<br/>
  &nbsp; &nbsp; For virtual stand generation requaered option <ins>-timeout 30m</ins><br/>
  &nbsp; &nbsp; and deckhouse license key <ins>export licensekey="..."</ins><br/>

  `-namespace 01-01-test`

  &nbsp; &nbsp; Test name space

  **Examples:**

  `go test -v ./tests/... -verbose`

  &nbsp; &nbsp; Run all tests on localhost with medium real-time output

  `go test -v -timeout 30m ./tests/01_first_test.go -verbose -debug -hypervisorkconfig kube-hypervisor.config -sshhost 10.20.30.40 -namespace 01-01-test`

  &nbsp; &nbsp; Run test unit with cluster generation on hypervisor

  `go test -v -timeout 30m ./tests/... -verbose -debug -hypervisorkconfig kube-hypervisor.config -sshhost user@10.20.30.40 -namespace 01-01-test -run TestOk/case1`

  &nbsp; &nbsp; Run test case with cluster generation on hypervisor


  > :bulb: You can prepare run command with alias<br/>
  > `alias e2e_test_hv='go test -v -timeout 30m ./tests/... -verbose -debug -hypervisorkconfig kube-hypervisor.config -sshhost user@10.20.30.40 -namespace 01-01-test'`<br/>
  > or script<br/>
  > `echo "go test -v -timeout 30m ./tests/... -verbose -debug -hypervisorkconfig kube-hypervisor.config -sshhost user@10.20.30.40 -namespace 01-01-test" > e2e_test_hv.sh`
