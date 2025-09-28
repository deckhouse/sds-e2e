# ADR-001: Архитектура CI пайплайна для e2e тестирования модулей Deckhouse

## Описание

Автоматизированный CI пайплайн для end-to-end тестирования модулей Deckhouse с поддержкой кроссплатформенности (GitHub Actions и GitLab CI) и гибкой интеграцией тестов.

## Контекст

Репозиторий `sds-e2e` содержит общие e2e тесты для различных модулей Deckhouse:
- `sds-replicated-volume` (LVM, Linstor)
- `sds-node-configurator` (управление LVM Volume Groups)  
- `data-export` (экспорт данных)
- Другие модули, которые будут добавляться в будущем

Тесты требуют доступ к Kubernetes кластерам (bare metal или hypervisor), SSH доступ к тестовым узлам, Go runtime и конфигурационные файлы для различных сред.

## Мотивация / Боль

На текущий момент e2e тестирование модулей выполняется вручную или частично автоматизировано, что приводит к:
- Задержкам в обнаружении проблем
- Несогласованности в тестовых сценариях
- Сложности масштабирования при добавлении новых модулей
- Отсутствию единой системы отчетности

## Область

### Цели
- Полная автоматизация e2e тестирования
- Поддержка двух сценариев интеграции тестов
- Кроссплатформенность (GitHub Actions + GitLab CI)
- Масштабируемость для новых модулей
- Детальная отчетность и мониторинг

### Не цели
- Unit тестирование (выполняется в пайплайнах модулей)
- Performance тестирование (отдельная область)
- Тестирование пользовательского интерфейса

## Детальное описание решения

### Сценарии интеграции тестов

#### Сценарий 1: Встраивание в пайплайны модулей

Тесты интегрируются в CI/CD пайплайн каждого модуля после этапа сборки.

**Классификация тестов по приблизительному времени выполнения:**

| Тип тестов | Время | Триггер | Среда |
|------------|-------|---------|-------|
| Smoke tests | 5-10 мин | Каждый push | local |
| Fast e2e | 10-30 мин | Каждый push | bare-metal |
| Integration | 30-60 мин | PR/MR | bare-metal |
| Full e2e | 60-120 мин | Ночью/выходные | hypervisor |
| Stress tests | 2+ часа | Еженедельно | hypervisor |

**Пример интеграции в GitHub Actions:**
```yaml
# В пайплайне модуля sds-replicated-volume
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - name: Build
        run: go build ./...

  smoke-tests:
    needs: build
    runs-on: ubuntu-latest
    steps:
      - name: Checkout e2e tests
        uses: actions/checkout@v4
        with:
          repository: deckhouse/sds-e2e
          path: sds-e2e
      - name: Run smoke tests
        run: |
          cd sds-e2e/testkit_v2
          go test -v -timeout 10m ./tests/00_healthcheck_test.go \
            -stand local -verbose -module sds-replicated-volume
```

#### Сценарий 2: Отдельный пайплайн тестирования

Тесты существуют в отдельном пайплайне, запускаемом независимо от пайплайнов модулей.

**Архитектура отдельного пайплайна:**
```yaml
# .github/workflows/e2e-testing.yml
name: E2E Testing Pipeline

on:
  schedule:
    - cron: '0 2 * * 1'  # Еженедельно в понедельник в 2:00 UTC
  workflow_dispatch:
    inputs:
      module:
        description: 'Модуль для тестирования'
        type: choice
        options: [all, sds-replicated-volume, sds-node-configurator, data-export]
      test_type:
        description: 'Тип тестов'
        type: choice
        options: [full, stress, regression]
  repository_dispatch:
    types: [e2e-test-request]

jobs:
  full-e2e-tests:
    runs-on: [self-hosted, hypervisor]
    strategy:
      matrix:
        module: [sds-replicated-volume, sds-node-configurator, data-export]
    steps:
      - name: Run full e2e tests
        run: |
          cd testkit_v2
          go test -v -timeout 120m ./tests/... \
            -stand metal -run "Test.*" \
            -hypervisorkconfig ${{ secrets.KUBECONFIG_HYPERVISOR }}
```

### Выбор набора тестов разработчиком

#### Управление через лейблы

Разработчики могут контролировать процесс тестирования через лейблы в описании PR/MR:

**Лейблы для пропуска тестов:**
```bash
# Пропустить все e2e тесты
[skip-e2e]

# Пропустить только медленные тесты
[skip-slow-tests]

# Пропустить тесты для конкретного модуля
[skip-e2e:sds-replicated-volume]
```

**Лейблы для принудительного запуска:**
```bash
# Запустить все тесты, включая медленные
[force-full-e2e]

# Запустить stress тесты
[force-stress-tests]

# Запустить тесты для всех модулей
[force-all-modules]
```

**Лейблы для выбора среды:**
```bash
# Запустить только на bare-metal
[env:bare-metal]

# Запустить только на hypervisor
[env:hypervisor]

# Запустить на обеих средах
[env:all]
```

**Лейблы для приоритизации:**
```bash
# Высокий приоритет - запустить немедленно
[priority:high]

# Низкий приоритет - запустить в свободное время
[priority:low]
```

**Лейблы для выбора конкретных тестов:**
```bash
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
```

**Лейблы для указания ветки модуля:**
```bash
# Тестировать с ветки develop модуля
[module-branch:develop]

# Тестировать с конкретной ветки модуля
[module-branch:hotfix/storage-issue]

# Тестировать с тега модуля
[module-tag:v1.2.3]
```


#### GitHub Actions

**Парсинг лейблов из PR:**
```yaml
# .github/workflows/e2e-testing.yml
name: E2E Testing Pipeline

on:
  pull_request:
    types: [opened, synchronize, reopened]
  workflow_dispatch:

jobs:
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

  smoke-tests:
    needs: parse-labels
    if: needs.parse-labels.outputs.skip-e2e != 'true'
    runs-on: ubuntu-latest
    steps:
      - name: Run smoke tests
        run: |
          cd testkit_v2
          # Определяем какие тесты запускать на основе фильтра
          TEST_FILTER="${{ needs.parse-labels.outputs.test-filter }}"
          if [ "$TEST_FILTER" = "healthcheck" ] || [ -z "$TEST_FILTER" ]; then
            go test -v -timeout 10m ./tests/00_healthcheck_test.go \
              -stand local -verbose
          fi

  fast-e2e:
    needs: [parse-labels, smoke-tests]
    if: needs.parse-labels.outputs.skip-e2e != 'true' && needs.parse-labels.outputs.skip-slow != 'true'
    runs-on: [self-hosted, bare-metal]
    steps:
      - name: Run fast e2e tests
        run: |
          cd testkit_v2
          # Определяем какие тесты запускать на основе фильтра
          TEST_FILTER="${{ needs.parse-labels.outputs.test-filter }}"
          if [ "$TEST_FILTER" = "sds-node-configurator" ] || [ "$TEST_FILTER" = "lvg" ] || [ -z "$TEST_FILTER" ]; then
            go test -v -timeout 30m ./tests/01_sds_nc_test.go \
              -stand metal -run "TestLvg" \
              -kconfig ${{ secrets.KUBECONFIG_BARE_METAL }}
          fi
          if [ "$TEST_FILTER" = "pvc" ] || [ -z "$TEST_FILTER" ]; then
            go test -v -timeout 30m ./tests/03_sds_lv_test.go \
              -stand metal -run "TestPVC" \
              -kconfig ${{ secrets.KUBECONFIG_BARE_METAL }}
          fi

  full-e2e:
    needs: [parse-labels, fast-e2e]
    if: |
      needs.parse-labels.outputs.skip-e2e != 'true' && 
      (needs.parse-labels.outputs.force-full == 'true' || 
       needs.parse-labels.outputs.skip-slow != 'true')
    runs-on: [self-hosted, hypervisor]
    steps:
      - name: Run full e2e tests
        run: |
          cd testkit_v2
          # Определяем какие тесты запускать на основе фильтра
          TEST_FILTER="${{ needs.parse-labels.outputs.test-filter }}"
          if [ "$TEST_FILTER" = "data-export" ] || [ -z "$TEST_FILTER" ]; then
            go test -v -timeout 120m ./tests/base_test.go \
              -stand metal -run "TestDataExport" \
              -hypervisorkconfig ${{ secrets.KUBECONFIG_HYPERVISOR }}
          fi
          if [ "$TEST_FILTER" = "sds-node-configurator" ] || [ "$TEST_FILTER" = "lvg" ] || [ -z "$TEST_FILTER" ]; then
            go test -v -timeout 120m ./tests/05_sds_node_configurator_test.go \
              -stand metal -run "TestLvg.*" \
              -hypervisorkconfig ${{ secrets.KUBECONFIG_HYPERVISOR }}
          fi

  stress-tests:
    needs: [parse-labels, full-e2e]
    if: |
      needs.parse-labels.outputs.skip-e2e != 'true' && 
      (needs.parse-labels.outputs.force-stress == 'true' || 
       needs.parse-labels.outputs.skip-slow != 'true')
    runs-on: [self-hosted, hypervisor]
    steps:
      - name: Run stress tests
        run: |
          cd testkit_v2
          go test -v -timeout 180m ./tests/... \
            -stand metal -run "Test.*" \
            -hypervisorkconfig ${{ secrets.KUBECONFIG_HYPERVISOR }}
```

**Ручной запуск через UI:**
1. Перейти в Actions → "E2E Testing Pipeline"
2. Нажать "Run workflow"
3. Выбрать параметры:
   - **Module**: all, sds-replicated-volume, sds-node-configurator, data-export
   - **Test Type**: full, stress, regression
   - **Environment**: bare-metal, hypervisor

**Ручной запуск через CLI:**
```bash
# Запуск smoke тестов для конкретного модуля
gh workflow run "E2E Testing Pipeline" \
  -f module=sds-replicated-volume \
  -f test_type=full

# Просмотр результатов
gh run list --workflow="E2E Testing Pipeline"
gh run view <run-id>
```

**Запуск через webhook из пайплайна модуля:**
```yaml
notify-e2e-pipeline:
  needs: [build, deploy]
  runs-on: ubuntu-latest
  steps:
    - name: Trigger e2e tests
      run: |
        curl -X POST \
          -H "Authorization: token ${{ secrets.GITHUB_TOKEN }}" \
          -d '{"event_type": "e2e-test-request", "client_payload": {"module": "sds-replicated-volume", "version": "${{ github.sha }}"}}' \
          https://api.github.com/repos/deckhouse/sds-e2e/dispatches
```

#### GitLab CI

**Парсинг лейблов из MR:**
```yaml
# .gitlab-ci.yml
variables:
  SKIP_E2E: "false"
  SKIP_SLOW: "false"
  FORCE_FULL: "false"
  FORCE_STRESS: "false"
  ENV_FILTER: "all"
  PRIORITY: "normal"
  TEST_FILTER: "all"
  MODULE_BRANCH: "main"
  MODULE_TAG: ""

parse-labels:
  stage: .pre
  image: alpine:latest
  script:
    - |
      # Извлекаем лейблы из описания MR
      if echo "$CI_MERGE_REQUEST_DESCRIPTION" | grep -q "\[skip-e2e\]"; then
        echo "SKIP_E2E=true" >> .env
      fi
      
      if echo "$CI_MERGE_REQUEST_DESCRIPTION" | grep -q "\[skip-slow-tests\]"; then
        echo "SKIP_SLOW=true" >> .env
      fi
      
      if echo "$CI_MERGE_REQUEST_DESCRIPTION" | grep -q "\[force-full-e2e\]"; then
        echo "FORCE_FULL=true" >> .env
      fi
      
      if echo "$CI_MERGE_REQUEST_DESCRIPTION" | grep -q "\[force-stress-tests\]"; then
        echo "FORCE_STRESS=true" >> .env
      fi
      
      # Извлекаем среду
      if echo "$CI_MERGE_REQUEST_DESCRIPTION" | grep -q "\[env:bare-metal\]"; then
        echo "ENV_FILTER=bare-metal" >> .env
      elif echo "$CI_MERGE_REQUEST_DESCRIPTION" | grep -q "\[env:hypervisor\]"; then
        echo "ENV_FILTER=hypervisor" >> .env
      elif echo "$CI_MERGE_REQUEST_DESCRIPTION" | grep -q "\[env:all\]"; then
        echo "ENV_FILTER=all" >> .env
      fi
      
      # Извлекаем приоритет
      if echo "$CI_MERGE_REQUEST_DESCRIPTION" | grep -q "\[priority:high\]"; then
        echo "PRIORITY=high" >> .env
      elif echo "$CI_MERGE_REQUEST_DESCRIPTION" | grep -q "\[priority:low\]"; then
        echo "PRIORITY=low" >> .env
      fi
      
      # Извлекаем конкретные тесты
      if echo "$CI_MERGE_REQUEST_DESCRIPTION" | grep -q "\[test:data-export\]"; then
        echo "TEST_FILTER=data-export" >> .env
      elif echo "$CI_MERGE_REQUEST_DESCRIPTION" | grep -q "\[test:sds-node-configurator\]"; then
        echo "TEST_FILTER=sds-node-configurator" >> .env
      elif echo "$CI_MERGE_REQUEST_DESCRIPTION" | grep -q "\[test:healthcheck\]"; then
        echo "TEST_FILTER=healthcheck" >> .env
      elif echo "$CI_MERGE_REQUEST_DESCRIPTION" | grep -q "\[test:lvg\]"; then
        echo "TEST_FILTER=lvg" >> .env
      elif echo "$CI_MERGE_REQUEST_DESCRIPTION" | grep -q "\[test:pvc\]"; then
        echo "TEST_FILTER=pvc" >> .env
      fi
      
      # Извлекаем ветку модуля
      if echo "$CI_MERGE_REQUEST_DESCRIPTION" | grep -q "\[module-branch:"; then
        MODULE_BRANCH=$(echo "$CI_MERGE_REQUEST_DESCRIPTION" | grep -o "\[module-branch:[^]]*\]" | sed 's/\[module-branch://;s/\]//')
        echo "MODULE_BRANCH=$MODULE_BRANCH" >> .env
      fi
      
      # Извлекаем тег модуля
      if echo "$CI_MERGE_REQUEST_DESCRIPTION" | grep -q "\[module-tag:"; then
        MODULE_TAG=$(echo "$CI_MERGE_REQUEST_DESCRIPTION" | grep -o "\[module-tag:[^]]*\]" | sed 's/\[module-tag://;s/\]//')
        echo "MODULE_TAG=$MODULE_TAG" >> .env
        echo "MODULE_BRANCH=$MODULE_TAG" >> .env
      fi
  artifacts:
    reports:
      dotenv: .env

smoke-tests:
  stage: test
  image: golang:1.22
  tags:
    - local
  script:
    - |
      cd testkit_v2
      # Определяем какие тесты запускать на основе фильтра
      if [ "$TEST_FILTER" = "healthcheck" ] || [ -z "$TEST_FILTER" ]; then
        go test -v -timeout 10m ./tests/00_healthcheck_test.go \
          -stand local -verbose
      fi
  needs:
    - job: parse-labels
      artifacts: true

fast-e2e:
  stage: test
  image: golang:1.22
  tags:
    - bare-metal
  script:
    - |
      cd testkit_v2
      # Определяем какие тесты запускать на основе фильтра
      if [ "$TEST_FILTER" = "sds-node-configurator" ] || [ "$TEST_FILTER" = "lvg" ] || [ -z "$TEST_FILTER" ]; then
        go test -v -timeout 30m ./tests/01_sds_nc_test.go \
          -stand metal -run "TestLvg" \
          -kconfig $KUBECONFIG_BARE_METAL
      fi
      if [ "$TEST_FILTER" = "pvc" ] || [ -z "$TEST_FILTER" ]; then
        go test -v -timeout 30m ./tests/03_sds_lv_test.go \
          -stand metal -run "TestPVC" \
          -kconfig $KUBECONFIG_BARE_METAL
      fi
  needs:
    - job: parse-labels
      artifacts: true
  rules:
    - if: $SKIP_E2E != "true" && $SKIP_SLOW != "true"

full-e2e:
  stage: test
  image: golang:1.22
  tags:
    - hypervisor
  script:
    - |
      cd testkit_v2
      # Определяем какие тесты запускать на основе фильтра
      if [ "$TEST_FILTER" = "data-export" ] || [ -z "$TEST_FILTER" ]; then
        go test -v -timeout 120m ./tests/base_test.go \
          -stand metal -run "TestDataExport" \
          -hypervisorkconfig $KUBECONFIG_HYPERVISOR
      fi
      if [ "$TEST_FILTER" = "sds-node-configurator" ] || [ "$TEST_FILTER" = "lvg" ] || [ -z "$TEST_FILTER" ]; then
        go test -v -timeout 120m ./tests/05_sds_node_configurator_test.go \
          -stand metal -run "TestLvg.*" \
          -hypervisorkconfig $KUBECONFIG_HYPERVISOR
      fi
  needs:
    - job: parse-labels
      artifacts: true
  rules:
    - if: $SKIP_E2E != "true" && ($FORCE_FULL == "true" || $SKIP_SLOW != "true")

stress-tests:
  stage: test
  image: golang:1.22
  tags:
    - hypervisor
  script:
    - |
      cd testkit_v2
      go test -v -timeout 180m ./tests/... \
        -stand metal -run "Test.*" \
        -hypervisorkconfig $KUBECONFIG_HYPERVISOR
  needs:
    - job: parse-labels
      artifacts: true
  rules:
    - if: $SKIP_E2E != "true" && ($FORCE_STRESS == "true" || $SKIP_SLOW != "true")
```

**Ручной запуск через UI:**
1. Перейти в CI/CD → Pipelines
2. Нажать "Run pipeline"
3. Выбрать переменные:
   - **MODULE_UNDER_TEST**: all, sds-replicated-volume, sds-node-configurator, data-export
   - **TEST_TYPE**: full, stress, regression
   - **TEST_ENVIRONMENT**: bare-metal, hypervisor

**Ручной запуск через CLI:**
```bash
# Запуск пайплайна с переменными
glab ci run --variable MODULE_UNDER_TEST=sds-replicated-volume \
  --variable TEST_TYPE=full \
  --variable TEST_ENVIRONMENT=bare-metal

# Просмотр результатов
glab ci list
glab ci view <pipeline-id>
```

### Ночные e2e тесты

#### GitHub Actions

**Расписание:**
```yaml
# .github/workflows/nightly-e2e.yml
name: Nightly E2E Tests

on:
  schedule:
    - cron: '0 2 * * 1-5'  # Понедельник-пятница в 2:00 UTC
  workflow_dispatch:

jobs:
  nightly-tests:
    runs-on: [self-hosted, hypervisor]
    steps:
      - name: Run nightly e2e tests
        run: |
          cd testkit_v2
          go test -v -timeout 120m ./tests/... \
            -stand metal -run "Test.*" \
            -hypervisorkconfig ${{ secrets.KUBECONFIG_HYPERVISOR }}
```

#### GitLab CI

**Расписание:**
```yaml
# В .gitlab-ci.yml
nightly-e2e-tests:
  stage: nightly
  image: golang:1.22
  tags:
    - hypervisor
    - k8s-cluster
  script:
    - go test -v -timeout 120m ./tests/... \
        -stand metal -run "Test.*" \
        -hypervisorkconfig $KUBECONFIG_HYPERVISOR
  rules:
    - if: $CI_PIPELINE_SOURCE == "schedule"
  when: on_success
```

### Кроссплатформенная архитектура

**GitHub Actions (основная платформа):**
- Self-hosted runners для bare-metal и hypervisor
- GitHub-hosted runners для local тестов
- Интеграция через webhook

**GitLab CI (дополнительная платформа):**
- GitLab runners с тегами для различных сред
- Переменные окружения для конфигурации
- Параллельное выполнение тестов

### Управление ресурсами

**Приоритизация тестов:**
1. **P0 (Критические)** - встраиваются в пайплайн, время < 15 минут
2. **P1 (Важные)** - встраиваются в пайплайн, время < 30 минут  
3. **P2 (Желательные)** - отдельный пайплайн, время < 60 минут
4. **P3 (Дополнительные)** - отдельный пайплайн, время > 60 минут

**Расписание выполнения:**
- При каждом push: Smoke, Fast e2e
- При PR/MR: Integration
- Ночью (2:00 UTC): Full e2e
- Выходные: Stress, Regression
- Еженедельно: Performance, Security

### Техническая реализация обработки лейблов

#### GitHub Actions

**Обработка лейблов в workflow:**
```yaml
# .github/workflows/e2e-testing.yml
name: E2E Testing Pipeline

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

jobs:
  parse-labels:
    runs-on: ubuntu-latest
    outputs:
      skip-e2e: ${{ steps.parse.outputs.skip-e2e }}
      skip-slow: ${{ steps.parse.outputs.skip-slow }}
      force-full: ${{ steps.parse.outputs.force-full }}
      force-stress: ${{ steps.parse.outputs.force-stress }}
      env-filter: ${{ steps.parse.outputs.env-filter }}
      priority: ${{ steps.parse.outputs.priority }}
    steps:
      - name: Parse commit message for labels
        id: parse
        run: |
          COMMIT_MSG="${{ github.event.head_commit.message }}"
          
          # Проверяем лейблы
          if echo "$COMMIT_MSG" | grep -q "\[skip-e2e\]"; then
            echo "skip-e2e=true" >> $GITHUB_OUTPUT
          fi
          
          if echo "$COMMIT_MSG" | grep -q "\[skip-slow-tests\]"; then
            echo "skip-slow=true" >> $GITHUB_OUTPUT
          fi
          
          if echo "$COMMIT_MSG" | grep -q "\[force-full-e2e\]"; then
            echo "force-full=true" >> $GITHUB_OUTPUT
          fi
          
          if echo "$COMMIT_MSG" | grep -q "\[force-stress-tests\]"; then
            echo "force-stress=true" >> $GITHUB_OUTPUT
          fi
          
          # Извлекаем среду
          if echo "$COMMIT_MSG" | grep -q "\[env:bare-metal\]"; then
            echo "env-filter=bare-metal" >> $GITHUB_OUTPUT
          elif echo "$COMMIT_MSG" | grep -q "\[env:hypervisor\]"; then
            echo "env-filter=hypervisor" >> $GITHUB_OUTPUT
          elif echo "$COMMIT_MSG" | grep -q "\[env:all\]"; then
            echo "env-filter=all" >> $GITHUB_OUTPUT
          fi
          
          # Извлекаем приоритет
          if echo "$COMMIT_MSG" | grep -q "\[priority:high\]"; then
            echo "priority=high" >> $GITHUB_OUTPUT
          el          if echo "$COMMIT_MSG" | grep -q "\[priority:low\]"; then
            echo "priority=low" >> $GITHUB_OUTPUT
          fi
          
          # Извлекаем конкретные тесты
          if echo "$COMMIT_MSG" | grep -q "\[test:data-export\]"; then
            echo "test-filter=data-export" >> $GITHUB_OUTPUT
          elif echo "$COMMIT_MSG" | grep -q "\[test:sds-node-configurator\]"; then
            echo "test-filter=sds-node-configurator" >> $GITHUB_OUTPUT
          elif echo "$COMMIT_MSG" | grep -q "\[test:healthcheck\]"; then
            echo "test-filter=healthcheck" >> $GITHUB_OUTPUT
          elif echo "$COMMIT_MSG" | grep -q "\[test:lvg\]"; then
            echo "test-filter=lvg" >> $GITHUB_OUTPUT
          elif echo "$COMMIT_MSG" | grep -q "\[test:pvc\]"; then
            echo "test-filter=pvc" >> $GITHUB_OUTPUT
          fi
          
          # Извлекаем ветку модуля
          MODULE_BRANCH=$(echo "$COMMIT_MSG" | grep -o "\[module-branch:[^]]*\]" | sed 's/\[module-branch://;s/\]//')
          if [ -n "$MODULE_BRANCH" ]; then
            echo "module-branch=$MODULE_BRANCH" >> $GITHUB_OUTPUT
          fi
          
          # Извлекаем тег модуля
          MODULE_TAG=$(echo "$COMMIT_MSG" | grep -o "\[module-tag:[^]]*\]" | sed 's/\[module-tag://;s/\]//')
          if [ -n "$MODULE_TAG" ]; then
            echo "module-tag=$MODULE_TAG" >> $GITHUB_OUTPUT
          fi

  smoke-tests:
    needs: parse-labels
    if: needs.parse-labels.outputs.skip-e2e != 'true'
    runs-on: ubuntu-latest
    steps:
      - name: Run smoke tests
        run: |
          cd testkit_v2
          # Определяем какие тесты запускать на основе фильтра
          TEST_FILTER="${{ needs.parse-labels.outputs.test-filter }}"
          if [ "$TEST_FILTER" = "healthcheck" ] || [ -z "$TEST_FILTER" ]; then
            go test -v -timeout 10m ./tests/00_healthcheck_test.go \
              -stand local -verbose
          fi

  fast-e2e:
    needs: [parse-labels, smoke-tests]
    if: needs.parse-labels.outputs.skip-e2e != 'true' && needs.parse-labels.outputs.skip-slow != 'true'
    runs-on: [self-hosted, bare-metal]
    steps:
      - name: Run fast e2e tests
        run: |
          cd testkit_v2
          # Определяем какие тесты запускать на основе фильтра
          TEST_FILTER="${{ needs.parse-labels.outputs.test-filter }}"
          if [ "$TEST_FILTER" = "sds-node-configurator" ] || [ "$TEST_FILTER" = "lvg" ] || [ -z "$TEST_FILTER" ]; then
            go test -v -timeout 30m ./tests/01_sds_nc_test.go \
              -stand metal -run "TestLvg" \
              -kconfig ${{ secrets.KUBECONFIG_BARE_METAL }}
          fi
          if [ "$TEST_FILTER" = "pvc" ] || [ -z "$TEST_FILTER" ]; then
            go test -v -timeout 30m ./tests/03_sds_lv_test.go \
              -stand metal -run "TestPVC" \
              -kconfig ${{ secrets.KUBECONFIG_BARE_METAL }}
          fi

  full-e2e:
    needs: [parse-labels, fast-e2e]
    if: |
      needs.parse-labels.outputs.skip-e2e != 'true' && 
      (needs.parse-labels.outputs.force-full == 'true' || 
       needs.parse-labels.outputs.skip-slow != 'true')
    runs-on: [self-hosted, hypervisor]
    steps:
      - name: Run full e2e tests
        run: |
          cd testkit_v2
          # Определяем какие тесты запускать на основе фильтра
          TEST_FILTER="${{ needs.parse-labels.outputs.test-filter }}"
          if [ "$TEST_FILTER" = "data-export" ] || [ -z "$TEST_FILTER" ]; then
            go test -v -timeout 120m ./tests/base_test.go \
              -stand metal -run "TestDataExport" \
              -hypervisorkconfig ${{ secrets.KUBECONFIG_HYPERVISOR }}
          fi
          if [ "$TEST_FILTER" = "sds-node-configurator" ] || [ "$TEST_FILTER" = "lvg" ] || [ -z "$TEST_FILTER" ]; then
            go test -v -timeout 120m ./tests/05_sds_node_configurator_test.go \
              -stand metal -run "TestLvg.*" \
              -hypervisorkconfig ${{ secrets.KUBECONFIG_HYPERVISOR }}
          fi

  stress-tests:
    needs: [parse-labels, full-e2e]
    if: |
      needs.parse-labels.outputs.skip-e2e != 'true' && 
      needs.parse-labels.outputs.force-stress == 'true'
    runs-on: [self-hosted, hypervisor]
    steps:
      - name: Run stress tests
        run: |
          cd testkit_v2
          go test -v -timeout 240m ./tests/stress/... \
            -stand metal -run "Test.*Stress.*" \
            -hypervisorkconfig ${{ secrets.KUBECONFIG_HYPERVISOR }}
```

#### GitLab CI

**Обработка лейблов в GitLab CI:**
```yaml
# .gitlab-ci.yml
variables:
  SKIP_E2E: "false"
  SKIP_SLOW: "false"
  FORCE_FULL: "false"
  FORCE_STRESS: "false"
  ENV_FILTER: "all"
  PRIORITY: "normal"

# Парсинг лейблов из commit message
parse-labels:
  stage: .pre
  image: alpine:latest
  script:
    - |
      # Извлекаем лейблы из commit message
      if echo "$CI_COMMIT_MESSAGE" | grep -q "\[skip-e2e\]"; then
        echo "SKIP_E2E=true" >> .env
      fi
      
      if echo "$CI_COMMIT_MESSAGE" | grep -q "\[skip-slow-tests\]"; then
        echo "SKIP_SLOW=true" >> .env
      fi
      
      if echo "$CI_COMMIT_MESSAGE" | grep -q "\[force-full-e2e\]"; then
        echo "FORCE_FULL=true" >> .env
      fi
      
      if echo "$CI_COMMIT_MESSAGE" | grep -q "\[force-stress-tests\]"; then
        echo "FORCE_STRESS=true" >> .env
      fi
      
      # Извлекаем среду
      if echo "$CI_COMMIT_MESSAGE" | grep -q "\[env:bare-metal\]"; then
        echo "ENV_FILTER=bare-metal" >> .env
      elif echo "$CI_COMMIT_MESSAGE" | grep -q "\[env:hypervisor\]"; then
        echo "ENV_FILTER=hypervisor" >> .env
      fi
      
      # Извлекаем приоритет
      if echo "$CI_COMMIT_MESSAGE" | grep -q "\[priority:high\]"; then
        echo "PRIORITY=high" >> .env
      elif echo "$CI_COMMIT_MESSAGE" | grep -q "\[priority:low\]"; then
        echo "PRIORITY=low" >> .env
      fi
  artifacts:
    reports:
      dotenv: .env

smoke-tests:
  stage: test
  image: golang:1.22
  script:
    - cd testkit_v2
    - go test -v -timeout 10m ./tests/00_healthcheck_test.go -stand local
  rules:
    - if: $SKIP_E2E != "true"
  needs:
    - job: parse-labels
      artifacts: true

fast-e2e:
  stage: test
  image: golang:1.22
  tags:
    - bare-metal
    - k8s-cluster
  script:
    - cd testkit_v2
    - go test -v -timeout 30m ./tests/... -stand metal -run "Test.*LVG.*"
  rules:
    - if: $SKIP_E2E != "true" && $SKIP_SLOW != "true"
  needs:
    - job: parse-labels
      artifacts: true

full-e2e:
  stage: test
  image: golang:1.22
  tags:
    - hypervisor
    - k8s-cluster
  script:
    - cd testkit_v2
    - go test -v -timeout 120m ./tests/... -stand metal -run "Test.*"
  rules:
    - if: $SKIP_E2E != "true" && ($FORCE_FULL == "true" || $SKIP_SLOW != "true")
  needs:
    - job: parse-labels
      artifacts: true

stress-tests:
  stage: test
  image: golang:1.22
  tags:
    - hypervisor
    - k8s-cluster
  script:
    - cd testkit_v2
    - go test -v -timeout 240m ./tests/stress/... -stand metal -run "Test.*Stress.*"
  rules:
    - if: $SKIP_E2E != "true" && $FORCE_STRESS == "true"
  needs:
    - job: parse-labels
      artifacts: true
```

### Лейблы для PR/MR

**GitHub Pull Requests:**
```bash
# В описании PR можно указать лейблы
# [skip-e2e] - пропустить все e2e тесты
# [skip-slow-tests] - пропустить медленные тесты
# [force-full-e2e] - запустить полные тесты
# [force-stress-tests] - запустить stress тесты
# [env:bare-metal] - запустить только на bare-metal
# [env:hypervisor] - запустить только на hypervisor
# [priority:high] - высокий приоритет
# [priority:low] - низкий приоритет
```

**GitLab Merge Requests:**
```bash
# В описании MR можно указать лейблы
# [skip-e2e] - пропустить все e2e тесты
# [skip-slow-tests] - пропустить медленные тесты
# [force-full-e2e] - запустить полные тесты
# [force-stress-tests] - запустить stress тесты
# [env:bare-metal] - запустить только на bare-metal
# [env:hypervisor] - запустить только на hypervisor
# [priority:high] - высокий приоритет
# [priority:low] - низкий приоритет
# [test:data-export] - запустить только тесты data-export
# [test:sds-node-configurator] - запустить только тесты sds-node-configurator
# [test:healthcheck] - запустить только healthcheck тесты
# [test:lvg] - запустить только LVG тесты
# [test:pvc] - запустить только PVC тесты
# [module-branch:develop] - использовать ветку develop модуля
# [module-tag:v1.2.3] - использовать тег v1.2.3 модуля
```

### Работа с ветками модулей

#### GitHub Actions

**Настройка ветки модуля в workflow:**
```yaml
# .github/workflows/e2e-testing.yml
jobs:
  setup-module:
    runs-on: ubuntu-latest
    outputs:
      module-branch: ${{ steps.setup.outputs.module-branch }}
      module-tag: ${{ steps.setup.outputs.module-tag }}
    steps:
      - name: Setup module version
        id: setup
        run: |
          # Извлекаем ветку или тег из commit message
          MODULE_BRANCH="${{ needs.parse-labels.outputs.module-branch }}"
          MODULE_TAG="${{ needs.parse-labels.outputs.module-tag }}"
          
          if [ -n "$MODULE_TAG" ]; then
            echo "module-branch=$MODULE_TAG" >> $GITHUB_OUTPUT
            echo "module-tag=$MODULE_TAG" >> $GITHUB_OUTPUT
          elif [ -n "$MODULE_BRANCH" ]; then
            echo "module-branch=$MODULE_BRANCH" >> $GITHUB_OUTPUT
          else
            echo "module-branch=main" >> $GITHUB_OUTPUT
          fi

  test-with-module:
    needs: [parse-labels, setup-module]
    runs-on: [self-hosted, hypervisor]
    steps:
      - name: Checkout e2e tests
        uses: actions/checkout@v4
        with:
          repository: deckhouse/sds-e2e
          path: sds-e2e
      
      - name: Setup module from branch/tag
        run: |
          MODULE_BRANCH="${{ needs.setup-module.outputs.module-branch }}"
          MODULE_TAG="${{ needs.setup-module.outputs.module-tag }}"
          
          # Клонируем модуль с нужной ветки/тега
          if [ -n "$MODULE_TAG" ]; then
            git clone --branch $MODULE_TAG https://github.com/deckhouse/sds-replicated-volume.git module
          else
            git clone --branch $MODULE_BRANCH https://github.com/deckhouse/sds-replicated-volume.git module
          fi
          
          # Копируем модуль в тестовую среду
          cp -r module/* sds-e2e/testkit_v2/module/
      
      - name: Run tests with specific module version
        run: |
          cd sds-e2e/testkit_v2
          go test -v -timeout 120m ./tests/... \
            -stand metal -run "Test.*" \
            -hypervisorkconfig ${{ secrets.KUBECONFIG_HYPERVISOR }} \
            -module-version ${{ needs.setup-module.outputs.module-branch }}
```

#### GitLab CI

**Настройка ветки модуля в GitLab CI:**
```yaml
# .gitlab-ci.yml
variables:
  MODULE_BRANCH: "main"
  MODULE_TAG: ""

setup-module:
  stage: .pre
  image: alpine:latest
  script:
    - |
      # Извлекаем ветку или тег из commit message
      if echo "$CI_COMMIT_MESSAGE" | grep -q "\[module-branch:"; then
        MODULE_BRANCH=$(echo "$CI_COMMIT_MESSAGE" | grep -o "\[module-branch:[^]]*\]" | sed 's/\[module-branch://;s/\]//')
        echo "MODULE_BRANCH=$MODULE_BRANCH" >> .env
      fi
      
      if echo "$CI_COMMIT_MESSAGE" | grep -q "\[module-tag:"; then
        MODULE_TAG=$(echo "$CI_COMMIT_MESSAGE" | grep -o "\[module-tag:[^]]*\]" | sed 's/\[module-tag://;s/\]//')
        echo "MODULE_TAG=$MODULE_TAG" >> .env
        echo "MODULE_BRANCH=$MODULE_TAG" >> .env
      fi
  artifacts:
    reports:
      dotenv: .env

test-with-module:
  stage: test
  image: golang:1.22
  tags:
    - hypervisor
    - k8s-cluster
  script:
    - |
      # Клонируем модуль с нужной ветки/тега
      if [ -n "$MODULE_TAG" ]; then
        git clone --branch $MODULE_TAG https://github.com/deckhouse/sds-replicated-volume.git module
      else
        git clone --branch $MODULE_BRANCH https://github.com/deckhouse/sds-replicated-volume.git module
      fi
      
      # Копируем модуль в тестовую среду
      cp -r module/* testkit_v2/module/
      
      # Запускаем тесты
      cd testkit_v2
      go test -v -timeout 120m ./tests/... \
        -stand metal -run "Test.*" \
        -hypervisorkconfig $KUBECONFIG_HYPERVISOR \
        -module-version $MODULE_BRANCH
  needs:
    - job: setup-module
      artifacts: true
```

### Доступные тесты

**Структура тестов в репозитории:**

| Файл | Тесты | Описание |
|------|-------|----------|
| `00_healthcheck_test.go` | `TestNodeHealthCheck` | Проверка здоровья узлов кластера |
| `01_sds_nc_test.go` | `TestLvg` | Тесты LVM Volume Groups |
| `03_sds_lv_test.go` | `TestPVC` | Тесты Persistent Volume Claims |
| `05_sds_node_configurator_test.go` | `TestLvg.*` | Расширенные тесты LVG |
| `base_test.go` | `TestDataExport` | Тесты экспорта данных |
| `99_finalizer_test.go` | `TestFinalizer` | Тесты финализаторов |

**Подтесты в TestDataExport:**
- `routing` - проверка маршрутизации
- `auth` - проверка аутентификации  
- `files_content` - проверка содержимого файлов
- `files_headers` - проверка заголовков файлов
- `directories` - проверка директорий (закомментирован)
- `methods_not_allowed` - проверка недопустимых методов (закомментирован)
- `block_mode` - проверка блочного режима (закомментирован)
- `ttl_expired` - проверка истечения TTL (закомментирован)
- `delete_data_export` - удаление data export (закомментирован)
- `export_type_already_exported` - проверка уже экспортированного типа (закомментирован)
- `unsupported_export_type` - проверка неподдерживаемого типа экспорта (закомментирован)
- `nonexistent_export_type` - проверка несуществующего типа экспорта (закомментирован)
- `data_exporter_creation` - создание data exporter (закомментирован)

## Минусы внедрения решения

- Увеличение времени пайплайнов модулей (до 20 минут)
- Сложность настройки и поддержки инфраструктуры
- Высокие затраты на self-hosted runners
- Зависимость от внешних сервисов (кластеры, SSH)

## Вопросы на будущее и дальнейшие планы

- Добавление новых модулей в тестирование
- Интеграция с внешними системами (TestRail, Jira)
- ИИ/ML функции для предсказания сбоев
- Облачная интеграция (AWS, GCP, Azure)

## Почему так решили?

- **Кроссплатформенность**: Поддержка различных CI систем для гибкости
- **Два сценария**: Оптимальный баланс между скоростью и полнотой тестирования
- **Модульность**: Легкое добавление новых модулей и тестовых сред
- **Автоматизация**: Полная автоматизация процесса тестирования

## Ответственные контактные лица

- Архитектор: [Имя]
- SRE Lead: [Имя]  
- SRE: [Имя]
