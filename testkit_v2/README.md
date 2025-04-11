## Installation
```bash
git clone https://github.com/deckhouse/sds-e2e.git
cd sds-e2e
git checkout origin/not_main_branch_if_required
cd testkit_v2
```

## Quick start for Bare Metal
Running on hypervisor require **Deckhouse server** with resources for VmCluster<br/>
Test creates virtual environment (namespace, virtual machines), install Deckhouse there and collect **virtual cluster**<br/>
All tests runs in virtual environment
```bash
# set env
export hv_ssh_dst="<user>@<host>"
export hv_ssh_key="<key path>"
export licensekey="<EE deckhouse license key>"

# init configs
mkdir ../../sds-e2e-cfg
ssh -t $hv_ssh_dst "sudo cat /root/.kube/config > kube-hypervisor.config"
scp $hv_ssh_dst:kube-hypervisor.config ../../sds-e2e-cfg/

# run tests
go test -v -timeout 99m ./tests/... -verbose -debug -hypervisorkconfig kube-hypervisor.config -sshhost $hv_ssh_dst
```

## Configs
### Check conditions
Instrument for tests configuration. You can check config fields with
- functions<br/>
&nbsp; &nbsp; `WhereIn(STRING...)` - parameter matches with any of arguments<br/>
&nbsp; &nbsp; `WhereNotIn(STRING...)` - parameter don't matches with all arguments<br/>
&nbsp; &nbsp; `WhereLike(STRING...)` - parameter contains any of substrings<br/>
&nbsp; &nbsp; `WhereNotLike(STRING...)` - parameter don't contains all substrings<br/>
&nbsp; &nbsp; `WhereReg(STRING...)` - parameter equal to any of regular expressions<br/>
&nbsp; &nbsp; `WhereNotReg(STRING...)` - parameter don't equal to all regular expressions

- string hooks<br/>
&nbsp; &nbsp; `"<condition>"` - parameter matche with string<br/>
&nbsp; &nbsp; `"!<condition>"` - parameter don't matche with string<br/>
&nbsp; &nbsp; `"%<condition>%"` - parameter contains substring<br/>
&nbsp; &nbsp; `"!%<condition>%"` - parameter don't contains substring

### Cluster required configuration
You can update test clusner configuration in **util/env.go**
- **NodeRequired** - list of test node configurations
```
"Label": {
  Name:    "!%node-skip%",
  Os:      "%Windows XP%",
  Kernel:  WhereLike{"5.15.0-122", "5.15.0-128"},
  Kubelet: WhereLike{"v1.28.15"},
},
```
> Run test on each node is required if <ins>-skipoptional</ins> option not set

- **Images** - OS image samples

- **VmCluster** - list of virtula machines in hypervisor mode
```
#name         role                 ip          cpu ram disk image
{"vm-name-1", ["master"],          "",         4,  8,  20,  "Ubuntu_22"},
{"vm-name-2", ["setup", "worker"], "10.0.0.7", 2,  6,  20,  "Ubuntu_22"},
```
> role - list of master/setup/worker (1 master and 1 setup is required)<br/>
> ip - static or empty (free)<br/>
> image - key from Images map or URL

## Run tests
`go test [FLAG]... PATH [FLAG|OPTION]...`

**Flags:**

`-v`

&nbsp; &nbsp; Use verbose mode and reat-time output for 'go test'

`-run TestOk/case1`

&nbsp; &nbsp; Select specific cases to run

`-skip TestFatal/ignore`

&nbsp; &nbsp; Select specific cases to skip

`-timeout 10m`

&nbsp; &nbsp; Set timeout for tests execution (default: 10 min)

`-parallel N`

&nbsp; &nbsp; Select specific cases to run

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

&nbsp; &nbsp; Test ssh host. Hypervisor ssh host if <ins>-hypervisorkconfig</ins> used (default: 127.0.0.1)

`-sshkey /home/user/.ssh/id_rsa`

&nbsp; &nbsp; Test ssh key (default: ~/.ssh/id_rsa)

`-kconfig kube-nested.config`

&nbsp; &nbsp; The k8s config path for test

`-hypervisorkconfig kube-hypervisor.config`

&nbsp; &nbsp; The k8s config path for hypervisor (virtualization administration)<br/>
&nbsp; &nbsp; For virtual stand generation **requaered** option <ins>-timeout 30m</ins><br/>
&nbsp; &nbsp; and **requaered** deckhouse license key <ins>export licensekey="..."</ins><br/>

`-namespace 01-01-test`

&nbsp; &nbsp; Test name space

`-keepstate`

&nbsp; &nbsp; Don`t clean up after test finished

`-logfile testlog.out`

&nbsp; &nbsp; Save detailed report to file (including verbose, debug)

> :bulb: You can prepare run command with alias<br/>
> `alias run_e2e_hv='go test -v -timeout 30m ./tests/... -debug -hypervisorkconfig kube-hypervisor.config -sshhost user@10.20.30.40 -namespace 01-01-test'`<br/>
> or script<br/>
> ```bash
> echo "hv_ssh_dst=\"$export hv_ssh_dst\"
> hv_ssh_key=\"$hv_ssh_key\"
> licensekey=\"$licensekey\"
> 
> go test -v -timeout 30m \$@ -debug -hypervisorkconfig kube-hypervisor.config -sshhost \$hv_ssh_dst -namespace 01-01-test
> " > run_e2e_hv.sh; chmod +x run_e2e_hv.sh
> ```
> and run `run_e2e_hv.sh -run TestNodeHealthCheck`

> Also you can use e2e_test.sh script with handy interface (Linux only)<br/>
> Manual and examples in `e2e_test.sh --help`

## Examples
Run all tests on localhost with medium real-time output (host preparation required)<br/>
&nbsp; &nbsp; `go test -v ./tests/... -verbose`

Create virtual cluster on hypervisor (silent mode, no <ins>-verbose</ins>, no <ins>-debug</ins>)<br/>
&nbsp; &nbsp; `go test -v ./tests/... -hypervisorkconfig kube-hypervisor.config -sshhost $hv_ssh_dst` `-timeout 30m` `-namespace 01-01-test` `-run TestNodeHealthCheck`

Run all tests with cluster generation (25+ minutes) on hypervisor cluster (one-time cluster if <ins>-namespace</ins> not set)<br/>
&nbsp; &nbsp; `go test -v` `-timeout 30m` `./tests/...` `-debug -hypervisorkconfig kube-hypervisor.config -sshhost $hv_ssh_dst`

Run test file on hypervisor with existing cluster (<ins>-namespace</ins> required)<br/>
&nbsp; &nbsp; `go test -v` `./tests/tools.go ./tests/00_healthcheck_test.go` `-debug -hypervisorkconfig kube-hypervisor.config -sshhost $hv_ssh_dst` `-namespace 01-01-test`

Debug exact/single test case (expression in <ins>-run</ins>) on hypervisor<br/>
&nbsp; &nbsp; `go test -v -timeout 30m ./tests/... -debug -hypervisorkconfig kube-hypervisor.config $hv_ssh_dst -namespace 01-01-test` `-run TestOk/case1` `-keepstate`

## Debug Hypervisor cluster
- **Get actual virtual machines**
```bash
export hv_ssh_dst="<user>@<host>"
ssh -t $hv_ssh_dst "sudo -i kubectl get vm -A"
  NAMESPACE     NAME        PHASE     NODE                    IPADDRESS    AGE
  26-03-test    vm-de11-1   Running   virtlab-storage-e2e-3   10.10.10.4   8d
  26-03-test    vm-ub22-1   Running   virtlab-storage-e2e-4   10.10.10.1   8d
  26-03-test    vm-ub22-2   Running   virtlab-storage-e2e-2   10.10.10.2   8d
```

- **Connest to virtual machines**<br/>
Pick vm actual ip address (Deckhouse master - first element in <ins>VmCluster</ins>)
```bash
export hv_ssh_dst="<user>@<host>"
ssh -J $hv_ssh_dst -i ../../sds-e2e-cfg/id_rsa_test user@10.10.10.1
```
