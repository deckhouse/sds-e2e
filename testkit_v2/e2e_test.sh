#!/usr/bin/env bash
shopt -s extglob

bold=$(tput bold)
normal=$(tput sgr0)
red="\033[0;31m"
nc="\033[0m"

DIR="$(cd "$(dirname "$0")" && pwd)"
OPTIONS="hi:v"
LONGOPTS="help,ssh-key:,ssh-host:,kconfig:,verbose,debug,run:,skip:,ns:,namespace:,hypervisor-kconfig:"

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

    --hypervisor-kconfig '~/.kube/config':
        Customized path to kubernetes config file for virtual hypervisor

    --ns, --namespace 'te2est-1234':
        Set test name space

    --fakepubsub-node-port <fakepubsub_node_port>:
        (not implemented)

  ${bold}Env:${normal}
    export licensekey=s6Cr6T

  ${bold}Examples:${normal}
    # Run all tests on default (Dev) stand
    $0
    $0 -v --kconfig \$KUBECONFIG --ssh-key ~/.ssh/id_rsa_cloud --ssh-host ubuntu@158.160.36.69

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

  local ssh_flags=(-fN -x)
  if [[ -n "$ssh_key" ]]; then
    ssh_flags+=("-i $ssh_key")
  fi

  for address in "$@"; do
    ssh_flags+=("-L $address")
  done

  shcmd="ssh ${ssh_flags[*]} ${ssh_host}"
  if $debug; then echo "RUN: ${shcmd}"; fi
  ${shcmd}
}

function run_local() {
  echo >&2 "Not implemented"
  exit 1
}

function run_dev() {
  shcmd="go test ${test_flags[*]} ${tests_path} ${test_args[*]}"
  if $debug; then echo "RUN: ${shcmd}"; fi
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
  shcmd="go test ${test_flags[*]} ${tests_path} ${test_args[*]}"
  if $debug; then echo "RUN: ${shcmd}"; fi
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
      --kconfig) test_args+=(-kconfig "$2"); shift 2 ;;
      --hypervisor-kconfig) test_args+=(-hypervisorkconfig "$2"); shift 2 ;;
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

  if [[ -z "$ssh_host" ]]; then echo -e "  ${red}No '--ssh-host' command line argument${nc}\n"; usage; exit 1; fi

  test_args+=(-stand "${run_stand}")

  case "${run_stand}" in
    local)
      run_local
      ;;
    dev)
      # Optionally some params can be hardcoded
      #ssh_host="ubuntu@55.44.33.22"
      #ssh_key="~/.ssh/id_rsa"

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
      #ssh_host="user@99.88.77.66"
      #ssh_key="~/.ssh/id_rsa"

      test_flags+=(-skip="TestFatal/ignore") # Fake example
      test_flags+=(-timeout "30m")
      test_args+=(-sshhost "${ssh_host}")
      test_args+=(-sshkey "${ssh_key}")
      run_bare_metal
      ;;
  esac
}

main "$@"
