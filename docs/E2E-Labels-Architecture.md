# Архитектура E2E тестов через лейблы

## Диаграмма архитектуры

```mermaid
graph TB
    subgraph "GitHub PR"
        PR[Pull Request]
        LABELS[Лейблы e2e:*]
        PR --> LABELS
    end
    
    subgraph "GitHub Actions"
        TRIGGER[E2E Tests Trigger]
        PARSE[Parse PR Labels]
        RUN[Run E2E Tests]
        NOTIFY[Notify Results]
        
        TRIGGER --> PARSE
        PARSE --> RUN
        RUN --> NOTIFY
    end
    
    subgraph "Test Execution"
        LOCAL[Local Tests<br/>ubuntu-latest]
        BARE[Bare Metal Tests<br/>self-hosted]
        HYPER[Hypervisor Tests<br/>self-hosted]
        
        RUN --> LOCAL
        RUN --> BARE
        RUN --> HYPER
    end
    
    subgraph "Results"
        ARTIFACTS[Test Artifacts]
        COMMENTS[PR Comments]
        SLACK[Slack Notifications]
        
        RUN --> ARTIFACTS
        NOTIFY --> COMMENTS
        NOTIFY --> SLACK
    end
    
    LABELS --> TRIGGER
    COMMENTS --> PR
```

## Поток выполнения

```mermaid
sequenceDiagram
    participant Dev as Developer
    participant PR as GitHub PR
    participant GH as GitHub Actions
    participant Runner as Test Runner
    participant Cluster as Test Cluster
    
    Dev->>PR: Add e2e:run label
    PR->>GH: Trigger E2E Tests Trigger
    GH->>GH: Parse PR labels
    
    alt Local Tests
        GH->>Runner: Run on ubuntu-latest
        Runner->>Runner: Execute smoke/integration tests
    else Bare Metal Tests
        GH->>Runner: Run on self-hosted bare-metal
        Runner->>Cluster: Setup bare-metal environment
        Runner->>Cluster: Execute tests
    else Hypervisor Tests
        GH->>Runner: Run on self-hosted hypervisor
        Runner->>Cluster: Setup hypervisor environment
        Runner->>Cluster: Execute tests
    end
    
    Runner->>GH: Upload test results
    GH->>PR: Comment with results
    GH->>Dev: Notify completion
```

## Матрица лейблов

```mermaid
graph LR
    subgraph "Основные лейблы"
        RUN[e2e:run]
        SKIP[e2e:skip]
    end
    
    subgraph "Типы тестов"
        SMOKE[e2e:smoke]
        FULL[e2e:full]
        INTEGRATION[integration<br/>по умолчанию]
    end
    
    subgraph "Модули"
        NC[e2e:sds-node-configurator]
        RV[e2e:sds-replicated-volume]
        DE[e2e:data-export]
        ALL[all<br/>по умолчанию]
    end
    
    subgraph "Окружения"
        LOCAL[local<br/>по умолчанию]
        BARE[e2e:bare-metal]
        HYPER[e2e:hypervisor]
    end
    
    subgraph "Приоритеты"
        HIGH[e2e:priority:high]
        LOW[e2e:priority:low]
        NORMAL[normal<br/>по умолчанию]
    end
    
    RUN --> SMOKE
    RUN --> FULL
    RUN --> INTEGRATION
    
    SMOKE --> NC
    SMOKE --> RV
    SMOKE --> DE
    SMOKE --> ALL
    
    NC --> LOCAL
    NC --> BARE
    NC --> HYPER
    
    LOCAL --> HIGH
    LOCAL --> LOW
    LOCAL --> NORMAL
```

## Компоненты системы

### 1. Workflow файлы

```
.github/workflows/
├── ci-main.yml              # Основной CI (существующий)
├── e2e-trigger.yml          # Новый: E2E через лейблы
└── utils/
    ├── setup-cluster.yml
    ├── cleanup.yml
    └── reporting.yml
```

### 2. Лейблы GitHub

| Категория | Лейбл | Описание |
|-----------|-------|----------|
| **Основные** | `e2e:run` | Запустить E2E тесты |
| | `e2e:skip` | Пропустить E2E тесты |
| **Типы** | `e2e:smoke` | Smoke тесты |
| | `e2e:full` | Полные тесты |
| **Модули** | `e2e:sds-node-configurator` | Тесты sds-node-configurator |
| | `e2e:sds-replicated-volume` | Тесты sds-replicated-volume |
| | `e2e:data-export` | Тесты data-export |
| **Окружения** | `e2e:bare-metal` | Bare-metal окружение |
| | `e2e:hypervisor` | Hypervisor окружение |
| **Приоритеты** | `e2e:priority:high` | Высокий приоритет |
| | `e2e:priority:low` | Низкий приоритет |

### 3. Runners

| Тип | Лейблы | Использование |
|------|--------|---------------|
| **GitHub-hosted** | `ubuntu-latest` | Локальные тесты |
| **Self-hosted** | `self-hosted`, `bare-metal` | Bare-metal тесты |
| **Self-hosted** | `self-hosted`, `hypervisor` | Hypervisor тесты |

## Логика принятия решений

### Условия запуска

```yaml
if: needs.parse-labels.outputs.run-e2e == 'true' && needs.parse-labels.outputs.skip-e2e == 'false'
```

### Матрица выполнения

```yaml
strategy:
  matrix:
    include:
      - module: ${{ needs.parse-labels.outputs.module }}
        environment: ${{ needs.parse-labels.outputs.environment }}
        test_type: ${{ needs.parse-labels.outputs.test_type }}
```

### Параметры по умолчанию

| Параметр | Значение по умолчанию | Лейбл для изменения |
|----------|----------------------|-------------------|
| `module` | `all` | `e2e:*` |
| `environment` | `local` | `e2e:*` |
| `test_type` | `integration` | `e2e:*` |
| `priority` | `normal` | `e2e:priority:*` |

## Интеграция с существующим CI

### Параллельное выполнение

```mermaid
graph TB
    subgraph "Существующий CI"
        CI[CI - Main Pipeline]
        CODE[Code Quality]
        SMOKE[Smoke Tests]
        INTEG[Integration Tests]
        
        CI --> CODE
        CI --> SMOKE
        CI --> INTEG
    end
    
    subgraph "Новый E2E Trigger"
        E2E[E2E Tests Trigger]
        PARSE[Parse Labels]
        RUN[Run E2E Tests]
        
        E2E --> PARSE
        PARSE --> RUN
    end
    
    subgraph "Результаты"
        RESULTS[Test Results]
        COMMENTS[PR Comments]
        
        SMOKE --> RESULTS
        INTEG --> RESULTS
        RUN --> RESULTS
        RESULTS --> COMMENTS
    end
```

### Совместимость

- ✅ **Не нарушает** существующий CI
- ✅ **Дополняет** функциональность
- ✅ **Опциональное** использование
- ✅ **Обратная совместимость**

## Мониторинг и отчетность

### Метрики

- Количество запусков по лейблам
- Время выполнения тестов
- Процент успешных тестов
- Использование ресурсов

### Уведомления

- Комментарии в PR
- Slack уведомления
- Email уведомления
- GitHub status checks

## Безопасность

### Доступ к секретам

- `KUBECONFIG_BARE_METAL` - доступ к bare-metal кластеру
- `KUBECONFIG_HYPERVISOR` - доступ к hypervisor кластеру
- `SSH_PRIVATE_KEY` - SSH доступ к узлам
- `DECKHOUSE_LICENSE` - лицензия Deckhouse

### Изоляция тестов

- Отдельные namespace для каждого запуска
- Автоматическая очистка ресурсов
- Ограничение времени выполнения
- Мониторинг использования ресурсов

## Масштабирование

### Горизонтальное масштабирование

- Добавление новых self-hosted runners
- Распределение нагрузки между runners
- Автоматическое масштабирование кластеров

### Вертикальное масштабирование

- Увеличение ресурсов runners
- Оптимизация тестов
- Кэширование зависимостей

## Troubleshooting

### Частые проблемы

1. **Тесты не запускаются**
   - Проверить наличие лейбла `e2e:run`
   - Убедиться в отсутствии лейбла `e2e:skip`

2. **Неправильное окружение**
   - Проверить правильность лейблов
   - Убедиться в доступности runners

3. **Ошибки доступа**
   - Проверить настройку секретов
   - Убедиться в правильности kubeconfig

### Отладка

- Логи в GitHub Actions
- Проверка статуса runners
- Мониторинг ресурсов кластера
- Анализ артефактов тестов
