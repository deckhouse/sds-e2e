#!/usr/bin/env bash
shopt -s extglob

bold=$(tput bold)
normal=$(tput sgr0)
red="\033[0;31m"
nc="\033[0m"

DIR="$(cd "$(dirname "$0")" && pwd)"
OPTIONS="hi:v"
LONGOPTS="help,ssh-key:,ssh-host:,kconfig:,verbose,debug,run:,skip:,ns:,namespace:"

function usage() {
  >&2 cat <<EOF
  ${bold}Info${normal}
    Run E2E tests

  ${bold}Usage:${normal}
    $0 [stand] [options] [tests_path]

  ${bold}Stand:${normal}
    Local:
        Run on local device (not implemented)
    Dev:
        Run on developer cluster, without virtualization (default)
    Stage:
        (not implemented)
    Ci:
        (not implemented)
    Metal:
        Run on bare meal cluster, with virtualization

  ${bold}Options:${normal}
    -h, --help:
        Display this help message

    -v, --verbose:
        Make tests run more verbosely

    -d, --debug:
        Display debug messages, skip unavailable tests

    --run '<expression>':
        Run tests that match the regular expression

    --skip '<expression>':
        Run tests that do not match the regular expression

    --ssh-host 'user@1.2.3.4':
        Set server IP and User for ssh forwarding
        
    --ssh-key '~/.ssh/id_rsa':
        Customized path to ssh key file

    --kconfig '~/.kube/config':
        Customized path to kubernetes config file

    --ns, --namespace 'te2est-1234':
        Set test name space

    --fakepubsub-node-port <fakepubsub_node_port>:
        (not implemented)

  ${bold}Env:${normal}
    export licensekey=s6Cr6T
    export registryDockerCfg=Ba6S4e==

  ${bold}Examples:${normal}
    # Run all tests on default (Dev) stand
    $0
    $0 -v --ssh-key ~/.ssh/id_rsa_cloud --ssh-host ubuntu@158.160.36.69

    # Run specific tests files on local stand
    $0 Local -v tests/01_sds_nc_test.go tests/02_sds_lv_test.go
    $0 Local tests/*_nc_test.go

    # Run exact test on bare metal stand
    $0 Metal ./tests/... -run=TestNode/LVG_resizing*
    $0 Metal -v --ssh-key='~/.ssh/id_rsa_metal' --ssh-host='v.pupkin@94.26.231.181' -run=TestNode/LVG_resizing
    ./e2e_test.sh Metal tests/ -run=TestLVG
    ./e2e_test.sh Metal tests/03_sds_lv_test.go -v -run TestPVC/PVC_clean
EOF
}

function ssh_fwd() {
  if [[ -z "$ssh_host" ]]; then return 1; fi

  # kill old port forwarding
  for id in $(ps aux | grep "ssh -fN -x .*-L [^\s]*\s" | awk '{print $2}'); do kill -3 $id; done
  #for id in $(ps aux | grep "ssh -fN -x .*-L [^\s]*\s${ssh_host}" | awk '{print $2}'); do kill -3 $id; done

  local ssh_flags=(-fN -x)
  if [[ -n "$ssh_key" ]]; then
    ssh_flags+=("-i $ssh_key")
  fi

  for address in "$@"; do
    ssh_flags+=("-L $address")
  done

  shcmd="ssh ${ssh_flags[*]} ${ssh_host}"
  if $verbose; then echo "RUN: ${shcmd}"; fi
  ${shcmd}
}

function run_local() {
  echo >&2 "Not implemented"
  exit 1
}

function run_dev() {
  # kubernetes api forwarding
  ssh_fwd 6445:127.0.0.1:6445

  # run tests
  shcmd="go test ${test_flags[*]} ${tests_path} ${test_args[*]}"
  if $verbose; then echo "RUN: ${shcmd}"; fi
  ${shcmd}
}

function run_stage() {
  echo >&2 "Not implemented"
  exit 1
}

function run_ci() {
  echo >&2 "Not implemented"
  exit 1
}

function run_bare_metal() {
  # kubernetes api forwarding (Bare Metal server, Master node) + VirtualMachines ssh forwarding
  ssh_fwd 6445:127.0.0.1:6445 6443:10.10.10.180:6443 \
          2220:10.10.10.180:22 2221:10.10.10.181:22 2222:10.10.10.182:22 2223:10.10.10.183:22 2224:10.10.10.184:22

  # run tests
  shcmd="go test ${test_flags[*]} ${tests_path} ${test_args[*]}"
  if $verbose; then echo "RUN: ${shcmd}"; fi
  ${shcmd}
}

function main() {
  local run_stand="dev"
  local ssh_host
  local ssh_key
  local tests_path="${DIR}/tests/"
  local test_flags=()
  local test_args=()
  local verbose=false
  local debug=false

  case "$1" in
    Local) run_stand="lockal"; shift ;;
    Dev)   run_stand="dev"; shift ;;
    Stage) run_stand="stage"; shift ;;
    Ci)    run_stand="ci"; shift ;;
    Metal) run_stand="metal"; shift ;;
  esac

  OPTS=$(getopt -a -q --options=$OPTIONS --long=$LONGOPTS -n "$progname" -- "$@")
  if [ $? != 0 ] ; then echo -e "  ${red}Error in command line arguments${nc}\n" >&2 ; usage; exit 1 ; fi
  eval set -- "$OPTS"

  # parse options
  while true; do
    case "$1" in
      -h|--help) usage; exit; ;;
      -v|--verbose) verbose=true; test_flags+=(-v); test_args+=(-verbose); shift ;;  #summary_format=standard-verbose
      -d|--debug) debug=true; test_args+=(-debug); shift ;;
      -i|--ssh-key) ssh_key="$2"; shift 2 ;;
      --ssh-host) ssh_host="$2"; shift 2 ;;
      --kconfig) echo >&2 "Not implemented"; shift 2 ;;
      --run) test_flags+=(-run="$2"); shift 2 ;;
      --skip) test_flags+=(-skip="$2"); shift 2 ;;
      --ns|--namespace) test_args+=(-namespace "$2"); shift 2 ;;

      # TODO add options

      -- ) shift; break ;;
      * ) break ;;
    esac
  done

  # parse args
  for arg in "$@"; do
    case "${arg}" in
      ?(.)/*([/a-zA-Z0-9_\-\.\+\(\)])) tests_path="${arg}" ;;
      [a-zA-Z0-9_]+([/a-zA-Z0-9_\-\.\+\(\)])) tests_path="${DIR}/${arg}" ;;

      *) echo >&2 "${red}invalid argument '${arg}'${nc}"; return 1 ;;
    esac
  done

  test_args+=(-stand "${run_stand}")

  case "${run_stand}" in
    local)
      run_local
      ;;
    dev)
      # Optionally some params can be hardcoded
      #ssh_host="ubuntu@158.160.36.69"
      #ssh_key="~/.ssh/id_rsa_T14"

      test_args+=(-kconfig "${DIR}/data/kube-dev.config")
      run_dev
      ;;
    stage)
      run_stage
      ;;
    ci)
      run_ci
      ;;
    metal)
      # Optionally some params can be hardcoded
      #ssh_host="d.shipkov@94.26.231.181"
      #ssh_key="~/.ssh/id_rsa_T14"

      # [OLD SCHOOL] init NS test1 (VMs, ...)
      # TODO run on master `sudo /sds-e2e/testkit/run.sh` if no NS

      test_flags+=(-skip="TestFatal/ignore") # Fake example
      test_args+=(-kconfig "${DIR}/data/kube-metal.config")
      test_flags+=(-timeout "30m")
      run_bare_metal
      ;;
  esac
}

main "$@"
