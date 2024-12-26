#!/usr/bin/env bash
shopt -s extglob

# TODO возможно добавить запуск тестов с пмошью gotestsum
#      https://github.com/kubernetes-sigs/prow/blob/main/test/integration/integration-test.sh
#      https://github.com/gotestyourself/gotestsum


## TODO extra options
##set -o errexit
##set -o nounset
##set -o pipefail
##
##SCRIPT_ROOT="$(cd "$(dirname "$0")" && pwd)"
### shellcheck disable=SC1091
##source "${SCRIPT_ROOT}"/lib.sh
##
### shellcheck disable=SC1091
##source "${REPO_ROOT}/hack/build/setup-go.sh"


function usage() {
  >&2 cat <<EOF
Run E2E tests.

Usage: $0 [stand] [options] [tests_path]

Stand:
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

Options:
    -cluster='<cluster_name>':
        <add description here>

    -run='<tests_to_run>':
        <add description here>

    -fakepubsub-node-port=<fakepubsub_node_port>:
        <add description here>

    -ssh-host='user@1.2.3.4':
        
    -ssh-key='~/.ssh/id_rsa':
        Customized path to ssh key file

    -kube-conf='~/.kube/config':
        Customized path to kubernetes config file

    -kube-context='ctx-name':
        Select not default context in config

    -v, -verbose:
        Make tests run more verbosely.

    -h, -help:
        Display this help message.

Examples:
    # Run tests on local stand
    $0 Local

    # Run tests on dev stand
    $0 Dev -run=TestNode/LVG_resizing

    # Run tests on stage stand
    $0 Stage

    # Run tests on ci stand
    $0 Ci

    # Run tests on bare metal stand
    $0 Metal
EOF

##Examples:
##  # Bring up the KIND cluster and Prow components, but only run the
##  # "TestClonerefs/postsubmit" test.
##  $0 -run=Clonerefs/post
##
##  # Only run the "TestClonerefs/postsubmit" test, with increased verbosity.
##  $0 -verbose -no-setup -run=Clonerefs/post
##
##  # Recompile and redeploy the Prow components that use the "fakegitserver" and
##  # "fakegerritserver" images, then only run the "TestClonerefs/postsubmit"
##  # test, but also
##  $0 -verbose -build=fakegitserver,fakegerritserver -run=Clonerefs/post
##
##  # Recompile "deck" image, redeploy "deck" and "deck-tenanted" Prow components,
##  # then only run the "TestDeck" tests. The test knows that "deck" and
##  # "deck-tenanted" components both depend on the "deck" image in lib.sh (grep
##  # for PROW_IMAGES_TO_COMPONENTS).
##  $0 -verbose -build=deck -run=Clonerefs/post
##
##  # Recompile all Prow components, redeploy them, and then only run the
##  # "TestClonerefs/postsubmit" test.
##  $0 -verbose -no-setup-kind-cluster -run=Clonerefs/post
##
##  # Before running the "TestClonerefs/postsubmit" test, delete all ProwJob
##  # Custom Resources and test pods from test-pods namespace.
##  $0 -verbose -no-setup-kind-cluster -run=Clonerefs/post -clear=ALL
##
##Options:
##    -run-integration-test='<tests_to_run>':
##    -no-setup:
##        Skip setup of the KIND cluster and Prow installation. That is, only run
##        gotestsum. This is useful if you already have the cluster and components
##        set up, and only want to run some tests without setting up the cluster
##        or recompiling Prow images.
##
##    -no-setup-kind-cluster:
##        Skip setup of the KIND cluster, but still (re-)install Prow to the
##        cluster. Flag "-build=..." implies this flag. This is useful if you want
##        to skip KIND setup. Most of the time, you will want to use this flag
##        when rerunning tests after initially setting up the cluster (because
##        most of the time your changes will not impact the KIND cluster itself).
##
##    -build='':
##        Build only the comma-separated list of Prow components with
##        "${REPO_ROOT}"/hack/prowimagebuilder. Useful when developing a fake
##        service that needs frequent recompilation. The images are a
##        comma-separated string. Also results in only redeploying certain entries
##        in PROW_COMPONENTS, by way of PROW_IMAGES_TO_COMPONENTS in lib.sh.
##
##        The value "ALL" for this flag is an alias for all images (PROW_IMAGES in
##        lib.sh).
##
##        By default, "-build=ALL" is assumed, so that users do not have to
##        provide any arguments to this script to run all tests.
##
##        Implies -no-setup-kind-cluster.
##
##    -clear='':
##        Delete the comma-separated list of Kubernetes resources from the KIND
##        cluster before running the test. Possible values: "ALL", "prowjobs",
##        "test-pods". ALL is an alias for prowjobs and test-pods.
##
##        This makes it easier to see the exact ProwJob Custom Resource ("watch
##        kubectl get prowjobs.prow.k8s.io") or associated test pod ("watch
##        kubectl get pods -n test-pods") that is created by the test being run.
##
##    -run='':
##        Run only those tests that match the given pattern. The format is
##        "TestName/testcasename". E.g., "TestClonerefs/postsubmit" will only run
##        1 test. Due to fuzzy matching, "Clonerefs/post" is equivalent.
##
##    -save-logs='':
##        Export all cluster logs to the given directory (directory will be
##        created if it doesn't exist).
##
##    -teardown:
##        Delete the KIND cluster and also the local Docker registry used by the
##        cluster.
}


DIR="$(cd "$(dirname "$0")" && pwd)"

#export kubeconfig=/app/tmp/kube.config
#export licensekey=GLfZvuUmrjFHs7wBPxmETnfCzC62j8uE
#export registryDockerCfg=eyJhdXRocyI6eyJkZXYtcmVnaXN0cnkuZGVja2hvdXNlLmlvIjp7ImF1dGgiOiJiR2xqWlc1elpTMTBiMnRsYmpwSFRHWmFkblZWYlhKcVJraHpOM2RDVUhodFJWUnVaa042UXpZeWFqaDFSUT09In19fQ==

function rm_sshfwd() {
  for id in $(ps aux | grep "ssh -fN .*-L 644.:.*:644.\s" | awk '{print $2}'); do kill -3 $id; done
}

function run_local() {
  echo >&2 "Not implemented"
  exit 1
}

function run_dev() {
  # kubernetes api forwarding
  rm_sshfwd
  ssh -fN ${ssh_key} -L 6445:127.0.0.1:6445 ${ssh_host}

  # run tests
  echo "RUN: go test -v ${tests_path} ${kube_context} ${kube_conf} ${tests_to_run}"
  go test -v ${tests_path} ${kube_context} ${kube_conf} ${tests_to_run}
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
  # kubernetes api forwarding
  rm_sshfwd
  ssh -fN ${ssh_key} -L 6444:127.0.0.1:6445 ${ssh_host}  # Bare Metal server
  ssh -fN ${ssh_key} -L 6445:10.10.10.180:6443 ${ssh_host}  # Master node

  # configure virtualmachines, virtualdisks, ...
  # TODO clusterManagement/cluster.go:InitClusterCreate()

  # run tests
  echo "RUN: go test -v ${tests_path} ${kube_context} ${kube_conf} ${tests_to_run}"
  go test -v ${tests_path} ${kube_context} ${kube_conf} ${tests_to_run}
}

###function build_gotestsum() {
###  log "Building gotestsum"
###  set -x
###  pushd "${REPO_ROOT}/hack/tools"
###  go build -o "${REPO_ROOT}/_bin/gotestsum" gotest.tools/gotestsum
###  popd
###  set +x
###}

function main() {
  declare -a tests_to_run
  #declare -a setup_args
  #setup_args=(-setup-kind-cluster -setup-prow-components -build=ALL)
  #declare -a clear_args
  #declare -a teardown_args

  local run_stand="dev"
  local ssh_host
  local ssh_key
  local kube_conf  #"${DIR}/data/kube.config"
  local kube_context
  local tests_path="${DIR}/tests/"

  #local summary_format=pkgname
  #local setup_kind_cluster=0
  #local setup_prow_components=0
  #local build_images
  #local resource
  #local resources_val
  #local fakepubsub_node_port

  for arg in "$@"; do
    case "${arg}" in
      Local)
        run_stand=lockal
        ;;
      Dev)
        run_stand=dev
        ;;
      Stage)
        run_stand=stage
        ;;
      Ci)
        run_stand=ci
        ;;
      Metal)
        run_stand=metal
        ;;
      -i=*)
        #ssh_key="-i ~/.ssh/id_rsa_T14"
        ssh_key=("${arg}")
        ;;
      -ssh-host)
        echo >&2 "Not implemented"
        ;;
      -ssh-key)
        echo >&2 "Not implemented"
        ;;
      -kube-conf)
        echo >&2 "Not implemented"
        ;;
      -kube-context)
        echo >&2 "Not implemented"
        ;;

      #-no-setup)
      #  unset 'setup_args[0]'
      #  unset 'setup_args[1]'
      #  unset 'setup_args[2]'
      #  ;;
      #-no-setup-kind-cluster)
      #  unset 'setup_args[0]'
      #  ;;
      #-build=*)
      #  # Imply -no-setup-kind-cluster.
      #  unset 'setup_args[0]'
      #  # Because we specified a "-build=..." flag explicitly, drop the default
      #  # "-build=ALL" option.
      #  unset 'setup_args[2]'
      #  setup_args+=("${arg}")
      #  ;;
      #-clear=*)
      #  resources_val="${arg#-clear=}"
      #  for resource in ${resources_val//,/ }; do
      #    case "${resource}" in
      #      ALL)
      #        clear_args=(-prowjobs -test-pods)
      #      ;;
      #      prowjobs|test-pods)
      #        clear_args+=("${resource}")
      #      ;;
      #      *)
      #        echo >&2 "unrecognized argument to -clear: ${resource}"
      #        return 1
      #      ;;
      #    esac
      #  done
      #  ;;
      -run=*)
        tests_to_run+=("${arg}")
        ;;
      #-save-logs=*)
      #  teardown_args+=("${arg}")
      #  ;;
      #-teardown)
      #  teardown_args+=(-all)
      #  ;;
      -v|-verbose)
        summary_format=standard-verbose
        ;;
      -help)
        usage
        return
        ;;
      --*)
        echo >&2 "cannot use flags with two leading dashes ('--...'), use single dashes instead ('-...')"
        return 1
        ;;
      ?(.)/*([/a-zA-Z0-9_\-\.\+\(\)]))
        tests_path="${arg}"
        ;;
      [a-zA-Z0-9_]+([/a-zA-Z0-9_\-\.\+\(\)]))
        tests_path="${DIR}/${arg}"
        ;;
      *)
        echo >&2 "invalid argument '${arg}'"
        return 1
        ;;
    esac
  done

###  # By default use 30303 for fakepubsub.
###  fakepubsub_node_port="30303"
###
###  # If in CI (pull-test-infra-integration presubmit job), do some things slightly differently.
###  if [[ -n "${ARTIFACTS:-}" ]]; then
###    # Use the ARTIFACTS variable to save log output.
###    teardown_args+=(-save-logs="${ARTIFACTS}/kind_logs")
###    # Randomize the node port used for the fakepubsub service.
###    fakepubsub_node_port="$(get_random_node_port)"
###    log "Using randomized port ${fakepubsub_node_port} for fakepubsub"
###  fi
###
###  if [[ -n "${teardown_args[*]}" ]]; then
###    # shellcheck disable=SC2064
###    trap "${SCRIPT_ROOT}/teardown.sh ${teardown_args[*]}" EXIT
###  fi
###
###  for arg in "${setup_args[@]}"; do
###    case "${arg}" in
###      -setup-kind-cluster) setup_kind_cluster=1 ;;
###      -setup-prow-components) setup_prow_components=1 ;;
###      -build=*)
###        build_images="${arg#-build=}"
###        ;;
###    esac
###  done
###
###  if ((setup_kind_cluster)); then
###    source "${REPO_ROOT}"/hack/tools/ensure-kind.sh
###    "${SCRIPT_ROOT}"/setup-kind-cluster.sh \
###      -fakepubsub-node-port="${fakepubsub_node_port}"
###  fi
###
###  if ((setup_prow_components)); then
###    "${SCRIPT_ROOT}"/setup-prow-components.sh \
###      ${build_images:+"-build=${build_images}"} \
###      -fakepubsub-node-port="${fakepubsub_node_port}"
###  fi
###
###  build_gotestsum
###
###  if [[ -n "${clear_args[*]}" ]]; then
###    "${SCRIPT_ROOT}/clear.sh" "${clear_args[@]}"
###  fi
###
###  log "Finished preparing environment; running integration test"
###
###  JUNIT_RESULT_DIR="${REPO_ROOT}/_output"
###  # If we are in CI, copy to the artifact upload location.
###  if [[ -n "${ARTIFACTS:-}" ]]; then
###    JUNIT_RESULT_DIR="${ARTIFACTS}"
###  fi
###
###  # Run integration tests with junit output.
###  mkdir -p "${JUNIT_RESULT_DIR}"
###  "${REPO_ROOT}/_bin/gotestsum" \
###    --format "${summary_format}" \
###    --junitfile="${JUNIT_RESULT_DIR}/junit-integration.xml" \
###    -- "${SCRIPT_ROOT}/test" \
###    --run-integration-test ${tests_to_run[@]:+"${tests_to_run[@]}"} \
###    --fakepubsub-node-port "${fakepubsub_node_port}"

  case "${run_stand}" in
    local)
      run_local
      ;;
    dev)
      ssh_host="ubuntu@158.160.36.69"
      ssh_key="-i ~/.ssh/id_rsa_T14"
      kube_conf="-kubeconf ${DIR}/data/kube-dev.config"
      kube_context="-kubecontext dev"
      run_dev
      ;;
    stage)
      run_stage
      ;;
    ci)
      run_ci
      ;;
    metal)
      ssh_host="d.shipkov@94.26.231.181"
      ssh_key="-i ~/.ssh/id_rsa_T14"
      kube_conf="-kubeconf ${DIR}/data/kube-metal.config"
      run_bare_metal
      ;;
  esac
}

main "$@"
