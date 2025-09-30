# ADR-002: Интеграция e2e тестов через лейблы в PR модулей

## Статус


## Контекст

В рамках развития системы тестирования модулей Deckhouse необходимо обеспечить возможность запуска e2e тестов для модулей через лейблы в Pull Request. Это позволит разработчикам выбирать, когда и какие тесты запускать, обеспечивая гибкость и контроль над процессом тестирования.

### Текущая ситуация
- Модули Deckhouse находятся в отдельных репозиториях (например, [sds-node-configurator](https://github.com/deckhouse/sds-node-configurator))
- e2e тесты находятся в центральном репозитории `sds-e2e`
- Необходима интеграция между репозиториями для запуска тестов
- Тесты должны выполняться на стороннем кластере с достаточными ресурсами

### Проблемы
1. Отсутствие единого механизма запуска e2e тестов для модулей
2. Невозможность выборочного запуска тестов разработчиками
3. Необходимость использования внешних ресурсов для выполнения тестов

## Решение

### Архитектура решения

#### 1. Система лейблов для PR/MR

Разработчики смогут добавлять лейблы в PR (GitHub) или MR (GitLab) для управления запуском e2e тестов:

**Основные лейблы:**
- `e2e:run` - запустить e2e тесты
- `e2e:skip` - пропустить e2e тесты
- `e2e:smoke` - запустить только smoke тесты
- `e2e:full` - запустить полный набор тестов

**Специфичные лейблы:**
- `e2e:data-export` - тесты для модуля data-export
- `e2e:sds-node-configurator` - тесты для модуля sds-node-configurator
- `e2e:sds-replicated-volume` - тесты для модуля sds-replicated-volume

**Лейблы окружения:**
- `e2e:bare-metal` - тесты на bare-metal окружении
- `e2e:hypervisor` - тесты на hypervisor окружении

#### 2. Интеграция с внешним кластером

**Архитектура выполнения:**
```
PR/MR с лейблом → CI/CD → Checkout e2e repo → Run tests → Report results
```

**Компоненты:**
- **CI/CD (GitHub Actions / GitLab CI)** - триггер и оркестрация
- **Внешний кластер** - кластер для выполнения тестов (настраивается через kubeconfig)
- **e2e тесты** - сами тесты поднимают тестовое окружение (DVP/VM) внутри себя

#### 3. Workflow интеграции

##### GitHub Actions (для GitHub репозиториев)

```yaml
# В репозитории модуля (например, sds-node-configurator)
name: E2E Tests Integration

on:
  pull_request:
    types: [opened, synchronize, labeled, unlabeled]

jobs:
  check-labels:
    runs-on: ubuntu-latest
    outputs:
      run-e2e: ${{ steps.check.outputs.run-e2e }}
      test-type: ${{ steps.check.outputs.test-type }}
      module: ${{ steps.check.outputs.module }}
      environment: ${{ steps.check.outputs.environment }}
    steps:
      - name: Check PR labels
        id: check
        run: |
          # Проверяем наличие лейблов e2e
          if echo "${{ github.event.pull_request.labels.*.name }}" | grep -q "e2e:run"; then
            echo "run-e2e=true" >> $GITHUB_OUTPUT
          fi
          
          # Определяем тип тестов
          if echo "${{ github.event.pull_request.labels.*.name }}" | grep -q "e2e:smoke"; then
            echo "test-type=smoke" >> $GITHUB_OUTPUT
          elif echo "${{ github.event.pull_request.labels.*.name }}" | grep -q "e2e:full"; then
            echo "test-type=full" >> $GITHUB_OUTPUT
          else
            echo "test-type=default" >> $GITHUB_OUTPUT
          fi
          
          # Определяем модуль
          if echo "${{ github.event.pull_request.labels.*.name }}" | grep -q "e2e:sds-node-configurator"; then
            echo "module=sds-node-configurator" >> $GITHUB_OUTPUT
          elif echo "${{ github.event.pull_request.labels.*.name }}" | grep -q "e2e:data-export"; then
            echo "module=data-export" >> $GITHUB_OUTPUT
          fi
          
          # Определяем окружение
          if echo "${{ github.event.pull_request.labels.*.name }}" | grep -q "e2e:bare-metal"; then
            echo "environment=bare-metal" >> $GITHUB_OUTPUT
          elif echo "${{ github.event.pull_request.labels.*.name }}" | grep -q "e2e:hypervisor"; then
            echo "environment=hypervisor" >> $GITHUB_OUTPUT
          else
            echo "environment=default" >> $GITHUB_OUTPUT
          fi

  trigger-e2e-tests:
    needs: check-labels
    if: needs.check-labels.outputs.run-e2e == 'true'
    runs-on: ubuntu-latest
    steps:
      - name: Trigger E2E tests
        run: |
          # Отправляем webhook в репозиторий sds-e2e
          curl -X POST \
            -H "Authorization: token ${{ secrets.E2E_TRIGGER_TOKEN }}" \
            -H "Accept: application/vnd.github.v3+json" \
            https://api.github.com/repos/deckhouse/sds-e2e/dispatches \
            -d '{
              "event_type": "module-e2e-tests",
              "client_payload": {
                "source_repo": "${{ github.repository }}",
                "source_pr": "${{ github.event.pull_request.number }}",
                "module": "${{ needs.check-labels.outputs.module }}",
                "test_type": "${{ needs.check-labels.outputs.test-type }}",
                "environment": "${{ needs.check-labels.outputs.environment }}",
                "commit_sha": "${{ github.event.pull_request.head.sha }}"
              }
            }'
```

##### GitLab CI (для GitLab репозиториев)

```yaml
# В репозитории модуля (например, sds-node-configurator)
# .gitlab-ci.yml

stages:
  - parse-labels
  - trigger-e2e

variables:
  E2E_REPO: "deckhouse/sds-e2e"
  MODULE_NAME: "sds-node-configurator"

parse-labels:
  stage: parse-labels
  image: alpine:latest
  rules:
    - if: $CI_PIPELINE_SOURCE == "merge_request_event"
    - if: $CI_PIPELINE_SOURCE == "push"
  script:
    - |
      # Получаем лейблы MR
      LABELS="$CI_MERGE_REQUEST_LABELS"
      echo "MR Labels: $LABELS"
      
      # Проверяем наличие лейбла e2e:run
      if echo "$LABELS" | grep -q "e2e:run"; then
        echo "run-e2e=true" >> .env
      fi
      
      # Проверяем наличие лейбла e2e:skip
      if echo "$LABELS" | grep -q "e2e:skip"; then
        echo "skip-e2e=true" >> .env
      fi
      
      # Определяем тип тестов
      if echo "$LABELS" | grep -q "e2e:smoke"; then
        echo "test-type=smoke" >> .env
      elif echo "$LABELS" | grep -q "e2e:full"; then
        echo "test-type=full" >> .env
      else
        echo "test-type=default" >> .env
      fi
      
      # Определяем модуль
      if echo "$LABELS" | grep -q "e2e:sds-node-configurator"; then
        echo "module=sds-node-configurator" >> .env
      elif echo "$LABELS" | grep -q "e2e:data-export"; then
        echo "module=data-export" >> .env
      elif echo "$LABELS" | grep -q "e2e:sds-replicated-volume"; then
        echo "module=sds-replicated-volume" >> .env
      else
        echo "module=$MODULE_NAME" >> .env
      fi
      
      # Определяем окружение
      if echo "$LABELS" | grep -q "e2e:bare-metal"; then
        echo "environment=bare-metal" >> .env
      elif echo "$LABELS" | grep -q "e2e:hypervisor"; then
        echo "environment=hypervisor" >> .env
      else
        echo "environment=default" >> .env
      fi
      
      # Определяем приоритет
      if echo "$LABELS" | grep -q "e2e:priority:high"; then
        echo "priority=high" >> .env
      elif echo "$LABELS" | grep -q "e2e:priority:low"; then
        echo "priority=low" >> .env
      else
        echo "priority=normal" >> .env
      fi
      
      # Выводим результат
      cat .env
  artifacts:
    reports:
      dotenv: .env
    expire_in: 1 hour

trigger-e2e:
  stage: trigger-e2e
  image: alpine:latest
  needs:
    - job: parse-labels
      artifacts: true
  rules:
    - if: $CI_PIPELINE_SOURCE == "merge_request_event" && $run-e2e == "true"
    - if: $CI_PIPELINE_SOURCE == "push" && $run-e2e == "true"
  script:
    - |
      # Подготавливаем payload для webhook
      PAYLOAD=$(cat <<EOF
      {
        "event_type": "module-e2e-tests",
        "client_payload": {
          "source_repo": "$CI_PROJECT_PATH",
          "source_mr": "$CI_MERGE_REQUEST_IID",
          "source_branch": "$CI_MERGE_REQUEST_SOURCE_BRANCH_NAME",
          "source_commit": "$CI_COMMIT_SHA",
          "module": "$module",
          "test_type": "$test-type",
          "environment": "$environment",
          "priority": "$priority",
          "triggered_by": "label",
          "triggered_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
          "ci_platform": "gitlab"
        }
      }
      EOF
      )
      
      echo "Triggering e2e tests with payload:"
      echo "$PAYLOAD"
      
      # Отправляем webhook в репозиторий sds-e2e
      curl -X POST \
        -H "Authorization: token $E2E_TRIGGER_TOKEN" \
        -H "Accept: application/vnd.github.v3+json" \
        -H "Content-Type: application/json" \
        https://api.github.com/repos/$E2E_REPO/dispatches \
        -d "$PAYLOAD"
      
      echo "E2E tests triggered successfully"
```

#### 4. Обработка в репозитории sds-e2e

```yaml
# В репозитории sds-e2e
name: Module E2E Tests

on:
  repository_dispatch:
    types: [module-e2e-tests]

jobs:
  run-e2e-tests:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout e2e tests
        uses: actions/checkout@v4
        with:
          repository: deckhouse/sds-e2e
          token: ${{ secrets.GITHUB_TOKEN }}
      
      - name: Setup external cluster access
        run: |
          # Настройка доступа к внешнему кластеру
          echo "${{ secrets.EXTERNAL_CLUSTER_KUBECONFIG }}" > kubeconfig
          mkdir -p ../../sds-e2e-cfg
          echo "${{ secrets.KUBECONFIG_HYPERVISOR }}" | base64 -d > ../../sds-e2e-cfg/kube-hypervisor.config
          echo "${{ secrets.SSH_PRIVATE_KEY }}" > ../../sds-e2e-cfg/id_rsa_test
          chmod 600 ../../sds-e2e-cfg/id_rsa_test
      
      - name: Run e2e tests
        run: |
          cd testkit_v2
          # Тесты сами поднимают окружение (DVP, VM и т.д.)
          go test -v -timeout 60m ./tests/... \
            -stand metal \
            -hypervisorkconfig ../../sds-e2e-cfg/kube-hypervisor.config \
            -sshhost ${{ secrets.SSH_HOST }} \
            -sshkey ../../sds-e2e-cfg/id_rsa_test \
            -verbose -debug
      
      - name: Report results
        run: |
          # Определяем платформу и отправляем результаты
          CI_PLATFORM="${{ github.event.client_payload.ci_platform || 'github' }}"
          
          if [ "$CI_PLATFORM" = "gitlab" ]; then
            # Отправка результатов в GitLab MR
            curl -X POST \
              -H "PRIVATE-TOKEN: ${{ secrets.GITLAB_TOKEN }}" \
              -H "Content-Type: application/json" \
              "${{ github.event.client_payload.source_repo }}/merge_requests/${{ github.event.client_payload.source_mr }}/notes" \
              -d '{
                "body": "E2E tests completed successfully! ✅"
              }'
          else
            # Отправка результатов в GitHub PR
            curl -X POST \
              -H "Authorization: token ${{ secrets.GITHUB_TOKEN }}" \
              -H "Accept: application/vnd.github.v3+json" \
              https://api.github.com/repos/${{ github.event.client_payload.source_repo }}/issues/${{ github.event.client_payload.source_pr }}/comments \
              -d '{
                "body": "E2E tests completed successfully! ✅"
              }'
          fi
```

### Преимущества решения

1. **Гибкость**: Разработчики могут выбирать, когда и какие тесты запускать
2. **Простота**: Использование стандартных GitHub лейблов
3. **Масштабируемость**: Легко добавлять новые лейблы и модули
4. **Изоляция**: Тесты выполняются на внешнем кластере, не нагружая GitHub runners
5. **Интеграция**: Единый механизм для всех модулей Deckhouse

### Недостатки

1. **Сложность настройки**: Требует настройки внешнего кластера и DVP
2. **Зависимости**: Зависимость от доступности внешнего кластера
3. **Безопасность**: Необходимость управления доступом к внешнему кластеру

## Proof of Concept (PoC)

### Шаг 1: Подготовка репозитория sds-node-configurator

1. Создайте лейблы в репозитории:
   ```bash
   gh label create "e2e:run" --description "Run E2E tests" --color "0E8A16"
   gh label create "e2e:skip" --description "Skip E2E tests" --color "D93F0B"
   gh label create "e2e:smoke" --description "Run smoke tests" --color "FBCA04"
   gh label create "e2e:full" --description "Run full tests" --color "1D76DB"
   gh label create "e2e:bare-metal" --description "Run on bare-metal" --color "5319E7"
   gh label create "e2e:hypervisor" --description "Run on hypervisor" --color "C5DEF5"
   ```

2. Добавьте секреты:
   ```bash
   gh secret set E2E_TRIGGER_TOKEN
   gh secret set KUBECONFIG_HYPERVISOR
   gh secret set SSH_PRIVATE_KEY
   gh secret set SSH_HOST
   gh secret set DECKHOUSE_LICENSE
   ```

### Шаг 2: Интеграция в CI

Добавьте в существующий `.github/workflows/ci.yml` job'ы из `templates/module-e2e-integration.yml`

### Шаг 3: Тестовый запуск

1. Создайте PR в sds-node-configurator
2. Добавьте лейблы: `e2e:run`, `e2e:smoke`
3. Наблюдайте выполнение e2e тестов в Actions
4. Проверьте комментарий с результатами в PR

### Ожидаемый результат

- ✅ Тесты запускаются автоматически при наличии лейбла
- ✅ Окружение поднимается внутри тестов
- ✅ Результаты публикуются в PR
- ✅ Основной CI не нарушен

## TODO: Будущие улучшения

### 1. Комбинации тестирования

#### 1.1 Тестирование с разными версиями Kubernetes
```yaml
# Добавить лейблы для версий K8s
- e2e:k8s:1.28
- e2e:k8s:1.29
- e2e:k8s:1.30
```

**Реализация:**
- Матрица тестирования с разными версиями K8s
- Автоматическое создание кластеров с нужной версией
- Отчеты о совместимости

#### 1.2 Тестирование с разными версиями модулей
```yaml
# Добавить лейблы для версий модулей
- e2e:module-version:v0.5.0
- e2e:module-version:v0.6.0
- e2e:module-version:latest
```

**Реализация:**
- Указание версии модуля через лейбл
- Тестирование обратной совместимости
- Матрица совместимости модулей

#### 1.3 Миграционное тестирование
```yaml
# Лейблы для миграционных тестов
- e2e:migration
- e2e:upgrade-from:v0.5.0
- e2e:upgrade-to:v0.6.0
```

**Сценарий:**
1. Создание ресурсов на старой версии модуля (v0.5.0)
2. Обновление модуля до новой версии (v0.6.0)
3. Проверка работоспособности ресурсов
4. Проверка миграции данных

**Пример теста:**
```go
func TestMigrationV05ToV06(t *testing.T) {
    // 1. Deploy v0.5.0
    cluster.DeployModule("sds-node-configurator", "v0.5.0")
    
    // 2. Create resources
    cluster.CreateLVG("test-lvg", nodeName, blockDevices)
    
    // 3. Upgrade to v0.6.0
    cluster.UpgradeModule("sds-node-configurator", "v0.6.0")
    
    // 4. Verify resources still work
    lvg, err := cluster.GetLVG("test-lvg")
    assert.NoError(t, err)
    assert.Equal(t, "Ready", lvg.Status.Phase)
}
```

### 2. Локальное тестирование с envtest

#### 2.1 Интеграция controller-runtime/envtest

Использование [envtest](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/envtest) для быстрых локальных тестов без полного кластера.

**Преимущества:**
- Быстрый запуск (секунды вместо минут)
- Не требует внешний кластер
- Идеально для unit/integration тестов контроллеров
- Запуск на GitHub-hosted runners

**Реализация:**
```go
// testkit_v2/util/envtest.go
package integration

import (
    "sigs.k8s.io/controller-runtime/pkg/envtest"
)

type EnvTestCluster struct {
    env    *envtest.Environment
    cfg    *rest.Config
}

func NewEnvTestCluster() (*EnvTestCluster, error) {
    env := &envtest.Environment{
        CRDDirectoryPaths: []string{
            filepath.Join("..", "crds"),
        },
        ErrorIfCRDPathMissing: true,
        BinaryAssetsDirectory: os.Getenv("KUBEBUILDER_ASSETS"),
    }
    
    cfg, err := env.Start()
    if err != nil {
        return nil, err
    }
    
    return &EnvTestCluster{env: env, cfg: cfg}, nil
}

func (e *EnvTestCluster) Stop() error {
    return e.env.Stop()
}
```

**Новый лейбл:**
```yaml
- e2e:envtest  # Запуск с envtest вместо полного кластера
```

**Применение:**
- Тесты контроллеров без внешних зависимостей
- CI/CD на GitHub-hosted runners
- Быстрая проверка логики контроллеров
- Smoke тесты перед полными e2e

#### 2.2 Гибридный подход

Комбинация envtest и полного кластера:
```yaml
jobs:
  envtest-quick:
    # Быстрые тесты на envtest (2-3 минуты)
    if: contains(github.event.pull_request.labels.*.name, 'e2e:envtest')
    
  full-e2e:
    # Полные тесты на реальном кластере (30-60 минут)
    if: contains(github.event.pull_request.labels.*.name, 'e2e:full')
```

### 3. Избирательный запуск тестов

#### 3.1 Лейблы для конкретных тестов

```yaml
# Лейблы для data-export тестов
- e2e:test:routing
- e2e:test:auth
- e2e:test:files-content
- e2e:test:ttl-expired
- e2e:test:block-mode

# Лейблы для sds-node-configurator тестов
- e2e:test:lvg-create
- e2e:test:lvg-resize
- e2e:test:lvg-delete
- e2e:test:thin-pool
```

#### 3.2 Реализация фильтрации тестов

**Парсинг лейблов:**
```yaml
- name: Parse test filter
  id: parse
  run: |
    LABELS="${{ join(github.event.pull_request.labels.*.name, ' ') }}"
    
    # Извлекаем конкретные тесты
    TEST_FILTER=""
    if echo "$LABELS" | grep -q "e2e:test:routing"; then
      TEST_FILTER="${TEST_FILTER}routing,"
    fi
    if echo "$LABELS" | grep -q "e2e:test:auth"; then
      TEST_FILTER="${TEST_FILTER}auth,"
    fi
    
    echo "test-filter=${TEST_FILTER%,}" >> $GITHUB_OUTPUT
```

**Запуск с фильтром:**
```bash
cd testkit_v2

# Запуск конкретных под-тестов
go test -v ./tests/base_test.go -run "TestDataExport/(routing|auth)"

# Или через переменную окружения
export TEST_FILTER="routing,auth,files-content"
go test -v ./tests/base_test.go
```

#### 3.3 Модификация base_test.go

```go
func TestDataExport(t *testing.T) {
    filter := os.Getenv("TEST_FILTER")
    testCases := map[string]func(*testing.T){
        "routing":       testDataExportRoutingValidation,
        "auth":          testDataExportAuth,
        "files-content": testDataExportFilesContent,
        "files-headers": testDataExportFilesHeaders,
        "ttl-expired":   testDataExportTTLExpired,
        "block-mode":    testStorageVolumeDataManagerBlock,
    }
    
    // Если фильтр указан, запускаем только выбранные тесты
    if filter != "" {
        selectedTests := strings.Split(filter, ",")
        for _, test := range selectedTests {
            if fn, ok := testCases[test]; ok {
                t.Run(test, fn)
            }
        }
        return
    }
    
    // Иначе запускаем все тесты
    for name, fn := range testCases {
        t.Run(name, fn)
    }
}
```

#### 3.4 Версионирование тестов

```yaml
# Лейблы для версий тестов
- e2e:test-version:v1  # Стабильные тесты
- e2e:test-version:v2  # Новые/экспериментальные тесты
- e2e:test-version:all # Все версии
```

**Применение:**
- Постепенный роллаут новых тестов
- Обратная совместимость с CI
- A/B тестирование тестов

## Мониторинг и метрики

- Количество запусков тестов по лейблам
- Время выполнения тестов
- Процент успешных тестов
- Использование ресурсов внешнего кластера

## Заключение

Предложенное решение обеспечивает гибкую и масштабируемую интеграцию e2e тестов с модулями Deckhouse через систему лейблов GitHub. Использование внешнего кластера для выполнения тестов позволяет обеспечить достаточные ресурсы и изоляцию тестовой среды.

Решение является опциональным и не нарушает существующие процессы разработки, при этом предоставляя разработчикам мощный инструмент для контроля качества кода.
