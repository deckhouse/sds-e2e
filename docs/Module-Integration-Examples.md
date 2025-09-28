# Примеры интеграции e2e тестов в пайплайны модулей

## Обзор

Данный документ содержит примеры интеграции e2e тестов в CI/CD пайплайны различных модулей Deckhouse, демонстрируя реализацию двух основных сценариев:

1. **Сценарий 1**: Встраивание тестов в пайплайны модулей
2. **Сценарий 2**: Отдельный пайплайн тестирования

## Сценарий 1: Интеграция в пайплайны модулей

### sds-replicated-volume

#### GitHub Actions

```yaml
# .github/workflows/ci.yml в репозитории sds-replicated-volume
name: SDS Replicated Volume CI

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]
  schedule:
    - cron: '0 2 * * 1'  # Еженедельно в понедельник в 2:00 UTC

env:
  MODULE_NAME: "sds-replicated-volume"
  E2E_REPO: "deckhouse/sds-e2e"

jobs:
  # Парсинг лейблов из PR
  parse-labels:
    runs-on: ubuntu-latest
    outputs:
      skip-e2e: ${{ steps.parse.outputs.skip-e2e }}
      skip-slow: ${{ steps.parse.outputs.skip-slow }}
      force-full: ${{ steps.parse.outputs.force-full }}
      force-stress: ${{ steps.parse.outputs.force-stress }}
      env-filter: ${{ steps.parse.outputs.env-filter }}
      priority: ${{ steps.parse.outputs.priority }}
      test-filter: ${{ steps.parse.outputs.test-filter }}
      module-branch: ${{ steps.parse.outputs.module-branch }}
      module-tag: ${{ steps.parse.outputs.module-tag }}
    steps:
      - name: Parse PR description for labels
        id: parse
        run: |
          PR_BODY="${{ github.event.pull_request.body }}"
          
          # Проверяем лейблы
          if echo "$PR_BODY" | grep -q "\[skip-e2e\]"; then
            echo "skip-e2e=true" >> $GITHUB_OUTPUT
          fi
          
          if echo "$PR_BODY" | grep -q "\[skip-slow-tests\]"; then
            echo "skip-slow=true" >> $GITHUB_OUTPUT
          fi
          
          if echo "$PR_BODY" | grep -q "\[force-full-e2e\]"; then
            echo "force-full=true" >> $GITHUB_OUTPUT
          fi
          
          if echo "$PR_BODY" | grep -q "\[force-stress-tests\]"; then
            echo "force-stress=true" >> $GITHUB_OUTPUT
          fi
          
          # Извлекаем среду
          if echo "$PR_BODY" | grep -q "\[env:bare-metal\]"; then
            echo "env-filter=bare-metal" >> $GITHUB_OUTPUT
          elif echo "$PR_BODY" | grep -q "\[env:hypervisor\]"; then
            echo "env-filter=hypervisor" >> $GITHUB_OUTPUT
          elif echo "$PR_BODY" | grep -q "\[env:all\]"; then
            echo "env-filter=all" >> $GITHUB_OUTPUT
          fi
          
          # Извлекаем приоритет
          if echo "$PR_BODY" | grep -q "\[priority:high\]"; then
            echo "priority=high" >> $GITHUB_OUTPUT
          elif echo "$PR_BODY" | grep -q "\[priority:low\]"; then
            echo "priority=low" >> $GITHUB_OUTPUT
          fi
          
          # Извлекаем конкретные тесты
          if echo "$PR_BODY" | grep -q "\[test:data-export\]"; then
            echo "test-filter=data-export" >> $GITHUB_OUTPUT
          elif echo "$PR_BODY" | grep -q "\[test:sds-node-configurator\]"; then
            echo "test-filter=sds-node-configurator" >> $GITHUB_OUTPUT
          elif echo "$PR_BODY" | grep -q "\[test:healthcheck\]"; then
            echo "test-filter=healthcheck" >> $GITHUB_OUTPUT
          elif echo "$PR_BODY" | grep -q "\[test:lvg\]"; then
            echo "test-filter=lvg" >> $GITHUB_OUTPUT
          elif echo "$PR_BODY" | grep -q "\[test:pvc\]"; then
            echo "test-filter=pvc" >> $GITHUB_OUTPUT
          fi
          
          # Извлекаем ветку модуля
          MODULE_BRANCH=$(echo "$PR_BODY" | grep -o "\[module-branch:[^]]*\]" | sed 's/\[module-branch://;s/\]//')
          if [ -n "$MODULE_BRANCH" ]; then
            echo "module-branch=$MODULE_BRANCH" >> $GITHUB_OUTPUT
          fi
          
          # Извлекаем тег модуля
          MODULE_TAG=$(echo "$PR_BODY" | grep -o "\[module-tag:[^]]*\]" | sed 's/\[module-tag://;s/\]//')
          if [ -n "$MODULE_TAG" ]; then
            echo "module-tag=$MODULE_TAG" >> $GITHUB_OUTPUT
          fi

  # Этап сборки
  build:
    needs: parse-labels
    runs-on: ubuntu-latest
    outputs:
      image-tag: ${{ steps.meta.outputs.tags }}
    steps:
      - name: Checkout
        uses: actions/checkout@v4
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.22'
      
      - name: Build Go binary
        run: |
          go build -o bin/sds-replicated-volume ./cmd/
      
      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3
      
      - name: Extract metadata
        id: meta
        uses: docker/metadata-action@v5
        with:
          images: ghcr.io/deckhouse/sds-replicated-volume
          tags: |
            type=ref,event=branch
            type=ref,event=pr
            type=sha,prefix={{branch}}-
      
      - name: Build and push Docker image
        uses: docker/build-push-action@v5
        with:
          context: .
          push: true
          tags: ${{ steps.meta.outputs.tags }}
          labels: ${{ steps.meta.outputs.labels }}

  # Быстрые тесты - встраиваются в пайплайн
  smoke-tests:
    needs: build
    runs-on: ubuntu-latest
    steps:
      - name: Checkout e2e tests
        uses: actions/checkout@v4
        with:
          repository: ${{ env.E2E_REPO }}
          path: sds-e2e
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.22'
      
      - name: Run smoke tests
        run: |
          cd sds-e2e/testkit_v2
          go test -v -timeout 10m ./tests/00_healthcheck_test.go \
            -stand local -verbose -module ${{ env.MODULE_NAME }}

  fast-e2e:
    needs: build
    runs-on: [self-hosted, bare-metal]
    steps:
      - name: Checkout e2e tests
        uses: actions/checkout@v4
        with:
          repository: ${{ env.E2E_REPO }}
          path: sds-e2e
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.22'
      
      - name: Run fast e2e tests
        run: |
          cd sds-e2e/testkit_v2
          go test -v -timeout 30m ./tests/... \
            -stand metal \
            -run "Test.*LVG.*|Test.*PVC.*|Test.*Storage.*" \
            -kconfig ${{ secrets.KUBECONFIG_BARE_METAL }} \
            -sshhost ${{ secrets.SSH_HOST }} \
            -sshkey ${{ secrets.SSH_PRIVATE_KEY }} \
            -namespace test-e2e-${{ github.run_id }}-${{ env.MODULE_NAME }}

  # Медленные тесты - отдельный workflow с расписанием
  scheduled-tests:
    if: github.event_name == 'schedule' || github.event_name == 'workflow_dispatch'
    needs: build
    runs-on: [self-hosted, hypervisor]
    steps:
      - name: Checkout e2e tests
        uses: actions/checkout@v4
        with:
          repository: ${{ env.E2E_REPO }}
          path: sds-e2e
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.22'
      
      - name: Run full e2e tests
        run: |
          cd sds-e2e/testkit_v2
          go test -v -timeout 120m ./tests/... \
            -stand metal \
            -run "Test.*" \
            -hypervisorkconfig ${{ secrets.KUBECONFIG_HYPERVISOR }} \
            -sshhost ${{ secrets.SSH_HOST }} \
            -sshkey ${{ secrets.SSH_PRIVATE_KEY }} \
            -namespace test-e2e-${{ github.run_id }}-${{ env.MODULE_NAME }}

  # Уведомление о готовности к тестированию
  notify-e2e-pipeline:
    needs: [build, smoke-tests, fast-e2e]
    if: always() && (needs.smoke-tests.result == 'success' && needs.fast-e2e.result == 'success')
    runs-on: ubuntu-latest
    steps:
      - name: Trigger e2e tests
        run: |
          curl -X POST \
            -H "Authorization: token ${{ secrets.GITHUB_TOKEN }}" \
            -H "Accept: application/vnd.github.v3+json" \
            -d '{
              "event_type": "e2e-test-request", 
              "client_payload": {
                "module": "${{ env.MODULE_NAME }}", 
                "version": "${{ github.sha }}",
                "image_tag": "${{ needs.build.outputs.image-tag }}"
              }
            }' \
            https://api.github.com/repos/${{ env.E2E_REPO }}/dispatches
```

#### GitLab CI

```yaml
# .gitlab-ci.yml в репозитории sds-replicated-volume
stages:
  - build
  - test-smoke
  - test-fast
  - test-scheduled
  - notify

variables:
  MODULE_NAME: "sds-replicated-volume"
  E2E_REPO: "deckhouse/sds-e2e"
  DOCKER_DRIVER: overlay2
  DOCKER_TLS_CERTDIR: "/certs"

services:
  - docker:24-dind

build:
  stage: build
  image: golang:1.22
  before_script:
    - go mod download
  script:
    - go build -o bin/sds-replicated-volume ./cmd/
    - docker build -t $CI_REGISTRY_IMAGE:$CI_COMMIT_SHA .
    - docker push $CI_REGISTRY_IMAGE:$CI_COMMIT_SHA
  artifacts:
    paths:
      - bin/
    expire_in: 1 hour

# Быстрые тесты
test-smoke:
  stage: test-smoke
  image: golang:1.22
  script:
    - git clone https://github.com/$E2E_REPO.git sds-e2e
    - cd sds-e2e/testkit_v2
    - go test -v -timeout 10m ./tests/00_healthcheck_test.go \
        -stand local -verbose -module $MODULE_NAME
  rules:
    - if: $CI_PIPELINE_SOURCE == "push"
    - if: $CI_PIPELINE_SOURCE == "merge_request_event"

test-fast:
  stage: test-fast
  image: golang:1.22
  tags:
    - bare-metal
    - k8s-cluster
    - lvm-support
  script:
    - git clone https://github.com/$E2E_REPO.git sds-e2e
    - cd sds-e2e/testkit_v2
    - go test -v -timeout 30m ./tests/... \
        -stand metal \
        -run "Test.*LVG.*|Test.*PVC.*|Test.*Storage.*" \
        -kconfig $KUBECONFIG_BARE_METAL \
        -sshhost $SSH_HOST \
        -sshkey $SSH_PRIVATE_KEY \
        -namespace test-e2e-$CI_PIPELINE_ID-$MODULE_NAME
  rules:
    - if: $CI_PIPELINE_SOURCE == "push"
    - if: $CI_PIPELINE_SOURCE == "merge_request_event"

# Медленные тесты
test-scheduled:
  stage: test-scheduled
  image: golang:1.22
  tags:
    - hypervisor
    - k8s-cluster
    - virtualization
    - nested-vm
  script:
    - git clone https://github.com/$E2E_REPO.git sds-e2e
    - cd sds-e2e/testkit_v2
    - go test -v -timeout 120m ./tests/... \
        -stand metal \
        -run "Test.*" \
        -hypervisorkconfig $KUBECONFIG_HYPERVISOR \
        -sshhost $SSH_HOST \
        -sshkey $SSH_PRIVATE_KEY \
        -namespace test-e2e-$CI_PIPELINE_ID-$MODULE_NAME
  rules:
    - if: $CI_PIPELINE_SOURCE == "schedule"
    - if: $CI_PIPELINE_SOURCE == "web"
  when: manual

# Уведомление о готовности к тестированию
notify-e2e:
  stage: notify
  image: alpine:latest
  script:
    - apk add --no-cache curl
    - |
      curl -X POST \
        -H "Authorization: token $GITHUB_TOKEN" \
        -H "Accept: application/vnd.github.v3+json" \
        -d '{
          "event_type": "e2e-test-request", 
          "client_payload": {
            "module": "$MODULE_NAME", 
            "version": "$CI_COMMIT_SHA",
            "pipeline_id": "$CI_PIPELINE_ID"
          }
        }' \
        https://api.github.com/repos/$E2E_REPO/dispatches
  rules:
    - if: $CI_PIPELINE_SOURCE == "push" && $CI_COMMIT_BRANCH =~ /^(main|develop)$/
  when: on_success
```

### sds-node-configurator

#### GitHub Actions

```yaml
# .github/workflows/ci.yml в репозитории sds-node-configurator
name: SDS Node Configurator CI

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

env:
  MODULE_NAME: "sds-node-configurator"
  E2E_REPO: "deckhouse/sds-e2e"

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.22'
      
      - name: Build
        run: go build ./...

  smoke-tests:
    needs: build
    runs-on: ubuntu-latest
    steps:
      - name: Checkout e2e tests
        uses: actions/checkout@v4
        with:
          repository: ${{ env.E2E_REPO }}
          path: sds-e2e
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.22'
      
      - name: Run smoke tests
        run: |
          cd sds-e2e/testkit_v2
          go test -v -timeout 10m ./tests/00_healthcheck_test.go \
            -stand local -verbose -module ${{ env.MODULE_NAME }}

  lvg-tests:
    needs: build
    runs-on: [self-hosted, bare-metal]
    steps:
      - name: Checkout e2e tests
        uses: actions/checkout@v4
        with:
          repository: ${{ env.E2E_REPO }}
          path: sds-e2e
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.22'
      
      - name: Run LVG tests
        run: |
          cd sds-e2e/testkit_v2
          go test -v -timeout 45m ./tests/01_sds_nc_test.go \
            -stand metal \
            -run "TestLvg" \
            -kconfig ${{ secrets.KUBECONFIG_BARE_METAL }} \
            -sshhost ${{ secrets.SSH_HOST }} \
            -sshkey ${{ secrets.SSH_PRIVATE_KEY }} \
            -namespace test-e2e-${{ github.run_id }}-${{ env.MODULE_NAME }}
```

### data-export

#### GitHub Actions

```yaml
# .github/workflows/ci.yml в репозитории data-export
name: Data Export CI

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

env:
  MODULE_NAME: "data-export"
  E2E_REPO: "deckhouse/sds-e2e"

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.22'
      
      - name: Build
        run: go build ./...

  smoke-tests:
    needs: build
    runs-on: ubuntu-latest
    steps:
      - name: Checkout e2e tests
        uses: actions/checkout@v4
        with:
          repository: ${{ env.E2E_REPO }}
          path: sds-e2e
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.22'
      
      - name: Run smoke tests
        run: |
          cd sds-e2e/testkit_v2
          go test -v -timeout 10m ./tests/00_healthcheck_test.go \
            -stand local -verbose -module ${{ env.MODULE_NAME }}

  data-export-tests:
    needs: build
    runs-on: [self-hosted, bare-metal]
    steps:
      - name: Checkout e2e tests
        uses: actions/checkout@v4
        with:
          repository: ${{ env.E2E_REPO }}
          path: sds-e2e
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.22'
      
      - name: Run data export tests
        run: |
          cd sds-e2e/testkit_v2
          go test -v -timeout 30m ./tests/base_test.go \
            -stand metal \
            -run "TestDataExport/routing|TestDataExport/auth|TestDataExport/files_content" \
            -kconfig ${{ secrets.KUBECONFIG_BARE_METAL }} \
            -sshhost ${{ secrets.SSH_HOST }} \
            -sshkey ${{ secrets.SSH_PRIVATE_KEY }} \
            -namespace test-e2e-${{ github.run_id }}-${{ env.MODULE_NAME }}
```

## Сценарий 2: Отдельный пайплайн тестирования

### GitHub Actions

```yaml
# .github/workflows/e2e-testing.yml в репозитории sds-e2e
name: E2E Testing Pipeline

on:
  schedule:
    - cron: '0 2 * * 1'  # Еженедельно в понедельник в 2:00 UTC
  workflow_dispatch:
    inputs:
      module:
        description: 'Модуль для тестирования'
        required: true
        type: choice
        options:
          - all
          - sds-replicated-volume
          - sds-node-configurator
          - data-export
      test_type:
        description: 'Тип тестов'
        required: true
        type: choice
        options:
          - full
          - stress
          - regression
      environment:
        description: 'Тестовая среда'
        required: true
        type: choice
        options:
          - bare-metal
          - hypervisor
  repository_dispatch:
    types: [e2e-test-request]

env:
  TEST_NAMESPACE: "test-e2e-${{ github.run_id }}"

jobs:
  # Полные e2e тесты
  full-e2e-tests:
    runs-on: [self-hosted, hypervisor]
    strategy:
      fail-fast: false
      matrix:
        module: [sds-replicated-volume, sds-node-configurator, data-export]
    steps:
      - name: Checkout
        uses: actions/checkout@v4
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.22'
      
      - name: Run full e2e tests
        run: |
          cd testkit_v2
          go test -v -timeout 120m ./tests/... \
            -stand metal \
            -run "Test.*" \
            -hypervisorkconfig ${{ secrets.KUBECONFIG_HYPERVISOR }} \
            -sshhost ${{ secrets.SSH_HOST }} \
            -sshkey ${{ secrets.SSH_PRIVATE_KEY }} \
            -namespace ${{ env.TEST_NAMESPACE }}-${{ matrix.module }}
        env:
          MODULE_UNDER_TEST: ${{ matrix.module }}

  # Stress тесты
  stress-tests:
    runs-on: [self-hosted, hypervisor]
    if: github.event_name == 'schedule'
    steps:
      - name: Checkout
        uses: actions/checkout@v4
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.22'
      
      - name: Run stress tests
        run: |
          cd testkit_v2
          go test -v -timeout 240m ./tests/stress/... \
            -stand metal \
            -run "Test.*Stress.*" \
            -hypervisorkconfig ${{ secrets.KUBECONFIG_HYPERVISOR }} \
            -sshhost ${{ secrets.SSH_HOST }} \
            -sshkey ${{ secrets.SSH_PRIVATE_KEY }} \
            -namespace ${{ env.TEST_NAMESPACE }}-stress

  # Регрессионные тесты
  regression-tests:
    runs-on: [self-hosted, bare-metal]
    if: github.event_name == 'workflow_dispatch' && github.event.inputs.test_type == 'regression'
    steps:
      - name: Checkout
        uses: actions/checkout@v4
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.22'
      
      - name: Run regression tests
        run: |
          cd testkit_v2
          go test -v -timeout 180m ./tests/... \
            -stand metal \
            -run "Test.*Regression.*" \
            -kconfig ${{ secrets.KUBECONFIG_BARE_METAL }} \
            -sshhost ${{ secrets.SSH_HOST }} \
            -sshkey ${{ secrets.SSH_PRIVATE_KEY }} \
            -namespace ${{ env.TEST_NAMESPACE }}-regression

  # Обработка webhook запросов
  handle-webhook:
    if: github.event_name == 'repository_dispatch'
    runs-on: [self-hosted, hypervisor]
    steps:
      - name: Checkout
        uses: actions/checkout@v4
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.22'
      
      - name: Run webhook triggered tests
        run: |
          cd testkit_v2
          MODULE="${{ github.event.client_payload.module }}"
          VERSION="${{ github.event.client_payload.version }}"
          
          echo "Running tests for module: $MODULE, version: $VERSION"
          
          go test -v -timeout 120m ./tests/... \
            -stand metal \
            -run "Test.*" \
            -hypervisorkconfig ${{ secrets.KUBECONFIG_HYPERVISOR }} \
            -sshhost ${{ secrets.SSH_HOST }} \
            -sshkey ${{ secrets.SSH_PRIVATE_KEY }} \
            -namespace ${{ env.TEST_NAMESPACE }}-$MODULE-$VERSION
        env:
          MODULE_UNDER_TEST: ${{ github.event.client_payload.module }}
          MODULE_VERSION: ${{ github.event.client_payload.version }}

  # Генерация отчета
  generate-report:
    needs: [full-e2e-tests, stress-tests, regression-tests, handle-webhook]
    if: always()
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4
      
      - name: Generate test report
        run: |
          mkdir -p test-report
          echo "# E2E Test Results" > test-report/README.md
          echo "Generated at: $(date)" >> test-report/README.md
          echo "" >> test-report/README.md
          echo "## Pipeline Information" >> test-report/README.md
          echo "- Pipeline ID: ${{ github.run_id }}" >> test-report/README.md
          echo "- Commit: ${{ github.sha }}" >> test-report/README.md
          echo "- Triggered by: ${{ github.event_name }}" >> test-report/README.md
          echo "" >> test-report/README.md
          
          # Создаем JSON отчет
          cat > test-report/summary.json << EOF
          {
            "pipeline_id": "${{ github.run_id }}",
            "commit_sha": "${{ github.sha }}",
            "triggered_by": "${{ github.event_name }}",
            "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
            "results": {
              "full_e2e_tests": "${{ needs.full-e2e-tests.result }}",
              "stress_tests": "${{ needs.stress-tests.result }}",
              "regression_tests": "${{ needs.regression-tests.result }}",
              "webhook_tests": "${{ needs.handle-webhook.result }}"
            }
          }
          EOF
      
      - name: Upload test report
        uses: actions/upload-artifact@v3
        with:
          name: test-report
          path: test-report/
          retention-days: 30
```

### GitLab CI

```yaml
# .gitlab-ci.yml в репозитории sds-e2e
stages:
  - full-e2e
  - stress
  - regression
  - reporting

variables:
  TEST_NAMESPACE: "test-e2e-${CI_PIPELINE_ID}"

# Полные e2e тесты
full-e2e-tests:
  stage: full-e2e
  image: golang:1.22
  tags:
    - hypervisor
    - k8s-cluster
    - virtualization
    - nested-vm
  parallel:
    matrix:
      - MODULE: ["sds-replicated-volume", "sds-node-configurator", "data-export"]
  script:
    - go test -v -timeout 120m ./tests/... \
        -stand metal \
        -run "Test.*" \
        -hypervisorkconfig $KUBECONFIG_HYPERVISOR \
        -sshhost $SSH_HOST \
        -sshkey $SSH_PRIVATE_KEY \
        -namespace ${TEST_NAMESPACE}-${MODULE}
  rules:
    - if: $CI_PIPELINE_SOURCE == "schedule"
    - if: $CI_PIPELINE_SOURCE == "web"
  artifacts:
    reports:
      junit: test-results/full-e2e-*.xml
    paths:
      - test-results/
    expire_in: 1 month

# Stress тесты
stress-tests:
  stage: stress
  image: golang:1.22
  tags:
    - hypervisor
    - k8s-cluster
    - virtualization
    - nested-vm
  script:
    - go test -v -timeout 240m ./tests/stress/... \
        -stand metal \
        -run "Test.*Stress.*" \
        -hypervisorkconfig $KUBECONFIG_HYPERVISOR \
        -sshhost $SSH_HOST \
        -sshkey $SSH_PRIVATE_KEY \
        -namespace ${TEST_NAMESPACE}-stress
  rules:
    - if: $CI_PIPELINE_SOURCE == "schedule"
  artifacts:
    reports:
      junit: test-results/stress-*.xml
    paths:
      - test-results/
    expire_in: 1 month

# Регрессионные тесты
regression-tests:
  stage: regression
  image: golang:1.22
  tags:
    - bare-metal
    - k8s-cluster
    - lvm-support
  script:
    - go test -v -timeout 180m ./tests/... \
        -stand metal \
        -run "Test.*Regression.*" \
        -kconfig $KUBECONFIG_BARE_METAL \
        -sshhost $SSH_HOST \
        -sshkey $SSH_PRIVATE_KEY \
        -namespace ${TEST_NAMESPACE}-regression
  rules:
    - if: $CI_PIPELINE_SOURCE == "web"
  when: manual
  artifacts:
    reports:
      junit: test-results/regression-*.xml
    paths:
      - test-results/
    expire_in: 1 month

# Генерация отчета
generate-report:
  stage: reporting
  image: alpine:latest
  needs:
    - job: full-e2e-tests
      artifacts: true
    - job: stress-tests
      artifacts: true
    - job: regression-tests
      artifacts: true
  script:
    - apk add --no-cache jq
    - mkdir -p test-report
    - |
      echo "# E2E Test Results" > test-report/README.md
      echo "Generated at: $(date)" >> test-report/README.md
      echo "" >> test-report/README.md
      echo "## Pipeline Information" >> test-report/README.md
      echo "- Pipeline ID: $CI_PIPELINE_ID" >> test-report/README.md
      echo "- Commit: $CI_COMMIT_SHA" >> test-report/README.md
      echo "- Triggered by: $CI_PIPELINE_SOURCE" >> test-report/README.md
      echo "" >> test-report/README.md
      
      # Создаем JSON отчет
      cat > test-report/summary.json << EOF
      {
        "pipeline_id": "$CI_PIPELINE_ID",
        "commit_sha": "$CI_COMMIT_SHA",
        "triggered_by": "$CI_PIPELINE_SOURCE",
        "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
        "total_artifacts": $(find . -name "*.xml" -type f | wc -l)
      }
      EOF
  artifacts:
    paths:
      - test-report/
    expire_in: 3 months
```

## Рекомендации по выбору сценария

### Используйте Сценарий 1 когда:
- Модуль имеет высокую частоту изменений
- Критически важна быстрая обратная связь
- Тесты выполняются быстро (< 30 минут)
- Доступны стабильные тестовые среды

### Используйте Сценарий 2 когда:
- Тесты выполняются долго (> 60 минут)
- Требуются специализированные ресурсы
- Нужно комплексное тестирование
- Важна независимость от пайплайнов модулей

### Гибридный подход:
- Быстрые тесты (smoke, unit) - Сценарий 1
- Медленные тесты (full e2e, stress) - Сценарий 2
- Критические тесты - оба сценария

## Примеры использования лейблов

### GitHub Pull Requests

```bash
# Пропустить все e2e тесты
[skip-e2e]

# Пропустить только медленные тесты
[skip-slow-tests]

# Запустить все тесты, включая медленные
[force-full-e2e]

# Запустить stress тесты
[force-stress-tests]

# Запустить только на bare-metal
[env:bare-metal]

# Запустить только на hypervisor
[env:hypervisor]

# Запустить только тесты data-export
[test:data-export]

# Запустить только тесты sds-node-configurator
[test:sds-node-configurator]

# Запустить только healthcheck тесты
[test:healthcheck]

# Запустить только LVG тесты
[test:lvg]

# Запустить только PVC тесты
[test:pvc]

# Тестировать с ветки develop модуля
[module-branch:develop]

# Тестировать с тега модуля
[module-tag:v1.2.3]

# Высокий приоритет
[priority:high]

# Низкий приоритет
[priority:low]
```

### GitLab Merge Requests

```bash
# Пропустить все e2e тесты
[skip-e2e]

# Пропустить только медленные тесты
[skip-slow-tests]

# Запустить все тесты, включая медленные
[force-full-e2e]

# Запустить stress тесты
[force-stress-tests]

# Запустить только на bare-metal
[env:bare-metal]

# Запустить только на hypervisor
[env:hypervisor]

# Запустить только тесты data-export
[test:data-export]

# Запустить только тесты sds-node-configurator
[test:sds-node-configurator]

# Запустить только healthcheck тесты
[test:healthcheck]

# Запустить только LVG тесты
[test:lvg]

# Запустить только PVC тесты
[test:pvc]

# Тестировать с ветки develop модуля
[module-branch:develop]

# Тестировать с тега модуля
[module-tag:v1.2.3]

# Высокий приоритет
[priority:high]

# Низкий приоритет
[priority:low]
```

### Преимущества использования лейблов

- **Гибкость**: Разработчики могут точно контролировать, какие тесты запускать
- **Экономия ресурсов**: Пропуск ненужных тестов экономит время и ресурсы
- **Простота использования**: Лейблы в PR/MR понятны всем разработчикам
- **Масштабируемость**: Легко добавлять новые лейблы и фильтры
- **Интеграция**: Работает как с GitHub Actions, так и с GitLab CI
