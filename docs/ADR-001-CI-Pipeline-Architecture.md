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

**Классификация тестов по времени выполнения:**

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

#### GitHub Actions

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
