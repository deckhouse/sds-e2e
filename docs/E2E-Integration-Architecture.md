# Архитектура интеграции e2e тестов через лейблы

## Обзор архитектуры

```mermaid
graph TB
    subgraph "Модули (GitHub)"
        PR[Pull Request с лейблами]
        GA[GitHub Actions]
    end
    
    subgraph "Модули (GitLab)"
        MR[Merge Request с лейблами]
        GCI[GitLab CI]
    end
    
    subgraph "Центральный репозиторий (sds-e2e)"
        WH[Webhook Handler]
        E2E[e2e тесты]
    end
    
    subgraph "Внешний кластер"
        DVP[DVP - Deckhouse Virtual Platform]
        K8S[Kubernetes кластер]
        TESTS[Выполнение тестов]
    end
    
    subgraph "Обратная связь"
        COMMENT[Комментарий в PR/MR]
        STATUS[Статус тестов]
    end
    
    PR -->|Лейблы e2e| GA
    MR -->|Лейблы e2e| GCI
    GA -->|Webhook| WH
    GCI -->|Webhook| WH
    WH -->|Запуск тестов| E2E
    E2E -->|Поднятие DVP| DVP
    DVP -->|Выполнение| TESTS
    TESTS -->|Результаты| COMMENT
    TESTS -->|Статус| STATUS
    COMMENT --> PR
    COMMENT --> MR
    STATUS --> PR
    STATUS --> MR
```

## Детальная архитектура

### 1. Система лейблов

```mermaid
graph LR
    subgraph "Лейблы управления"
        RUN[e2e:run]
        SKIP[e2e:skip]
        SMOKE[e2e:smoke]
        FULL[e2e:full]
    end
    
    subgraph "Лейблы модулей"
        NC[e2e:sds-node-configurator]
        DE[e2e:data-export]
        RV[e2e:sds-replicated-volume]
    end
    
    subgraph "Лейблы окружения"
        BM[e2e:bare-metal]
        HV[e2e:hypervisor]
    end
    
    RUN --> NC
    RUN --> DE
    RUN --> RV
    SMOKE --> BM
    FULL --> HV
```

### 2. Workflow выполнения

```mermaid
sequenceDiagram
    participant Dev as Разработчик
    participant PR as Pull Request / Merge Request
    participant CI as CI/CD (GitHub Actions / GitLab CI)
    participant WH as Webhook Handler
    participant EC as Внешний кластер
    participant DVP as DVP
    participant E2E as e2e тесты
    
    Dev->>PR: Добавляет лейбл e2e:run
    PR->>CI: Триггер события
    CI->>CI: Парсинг лейблов
    CI->>WH: Отправка webhook
    WH->>EC: Подключение к кластеру
    EC->>DVP: Развертывание DVP
    DVP->>E2E: Запуск тестов
    E2E->>PR: Отправка результатов
```

### 3. Компоненты системы

#### GitHub Actions в модуле (GitHub)
```yaml
# Основные компоненты:
- Парсинг лейблов PR
- Валидация лейблов
- Отправка webhook в sds-e2e
- Обработка результатов
```

#### GitLab CI в модуле (GitLab)
```yaml
# Основные компоненты:
- Парсинг лейблов MR
- Валидация лейблов
- Отправка webhook в sds-e2e
- Обработка результатов
```

#### Webhook Handler в sds-e2e
```yaml
# Основные компоненты:
- Получение webhook от GitHub и GitLab
- Валидация payload
- Запуск тестов на внешнем кластере
- Отправка результатов в PR/MR
```

#### Внешний кластер
```yaml
# Основные компоненты:
- Kubernetes кластер
- DVP (Deckhouse Virtual Platform)
- Ресурсы для выполнения тестов
- Мониторинг и логирование
```

## Примеры использования

### Пример 1: Smoke тесты для sds-node-configurator (GitHub)
```bash
# Лейблы в PR:
- e2e:run
- e2e:smoke
- e2e:sds-node-configurator
- e2e:bare-metal
```

### Пример 2: Полные тесты для data-export (GitLab)
```bash
# Лейблы в MR:
- e2e:run
- e2e:full
- e2e:data-export
- e2e:hypervisor
```

### Пример 3: Пропуск тестов (GitHub/GitLab)
```bash
# Лейблы в PR/MR:
- e2e:skip
```

### Пример 4: Высокий приоритет (GitLab)
```bash
# Лейблы в MR:
- e2e:run
- e2e:smoke
- e2e:sds-node-configurator
- e2e:priority:high
```

## Безопасность

### Доступ к внешнему кластеру
- Использование GitHub Secrets для хранения kubeconfig
- RBAC для ограничения доступа
- Временные токены для выполнения тестов

### Изоляция тестов
- Отдельные namespace для каждого запуска
- Очистка ресурсов после выполнения
- Лимиты ресурсов для тестов

## Мониторинг

### Метрики
- Количество запусков по лейблам
- Время выполнения тестов
- Процент успешных тестов
- Использование ресурсов кластера

### Логирование
- Централизованное логирование в sds-e2e
- Структурированные логи для анализа
- Интеграция с системами мониторинга

## Масштабирование

### Горизонтальное масштабирование
- Несколько внешних кластеров
- Балансировка нагрузки
- Географическое распределение

### Вертикальное масштабирование
- Увеличение ресурсов кластера
- Оптимизация DVP
- Кэширование образов
