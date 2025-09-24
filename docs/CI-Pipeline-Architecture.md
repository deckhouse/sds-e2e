# Архитектура CI пайплайна

## Диаграмма архитектуры

```mermaid
graph TB
    subgraph "GitHub Actions"
        A[Push/PR/Schedule] --> B[CI Main Pipeline]
        B --> C[Code Quality]
        B --> D[Smoke Tests]
        B --> E[Integration Tests]
        
        C --> C1[Go vet]
        C --> C2[Go fmt]
        C --> C3[golangci-lint]
        
        D --> D1[Local Tests]
        D --> D2[GitHub Runners]
        
        E --> E1[Bare Metal Tests]
        E --> E2[Hypervisor Tests]
        
        E1 --> E1A[Self-hosted Runner]
        E2 --> E2A[Self-hosted Runner]
    end
    
    subgraph "Test Environments"
        F[Bare Metal Cluster]
        G[Hypervisor Cluster]
        H[Local Environment]
        
        E1A --> F
        E2A --> G
        D2 --> H
    end
    
    subgraph "Modules"
        I[sds-replicated-volume]
        J[sds-node-configurator]
        K[data-export]
        
        F --> I
        F --> J
        F --> K
        
        G --> I
        G --> J
        G --> K
    end
    
    subgraph "Reporting"
        L[Test Results]
        M[HTML Reports]
        N[Slack Notifications]
        
        E1 --> L
        E2 --> L
        L --> M
        L --> N
    end
    
    subgraph "Secrets & Config"
        O[KUBECONFIG_BARE_METAL]
        P[KUBECONFIG_HYPERVISOR]
        Q[SSH_PRIVATE_KEY]
        R[DECKHOUSE_LICENSE]
        S[SLACK_WEBHOOK]
        
        O --> F
        P --> G
        Q --> F
        Q --> G
        R --> F
        R --> G
        S --> N
    end
```

## Поток выполнения

```mermaid
sequenceDiagram
    participant Dev as Developer
    participant GH as GitHub
    participant Runner as Self-hosted Runner
    participant Cluster as Test Cluster
    participant Slack as Slack
    
    Dev->>GH: Push code / Create PR
    GH->>GH: Trigger CI Main Pipeline
    
    par Code Quality
        GH->>GH: Run go vet, fmt, lint
    and Smoke Tests
        GH->>GH: Run local smoke tests
    and Integration Tests
        GH->>Runner: Deploy to self-hosted runner
        Runner->>Cluster: Setup test environment
        Runner->>Cluster: Run module tests
        Cluster->>Runner: Return test results
        Runner->>GH: Upload artifacts
    end
    
    GH->>Slack: Send notifications
    GH->>Dev: Update PR status
```

## Матрица тестирования

```mermaid
graph LR
    subgraph "Modules"
        M1[sds-replicated-volume]
        M2[sds-node-configurator]
        M3[data-export]
    end
    
    subgraph "Environments"
        E1[bare-metal]
        E2[hypervisor]
        E3[local]
    end
    
    subgraph "Test Types"
        T1[smoke]
        T2[integration]
        T3[full]
    end
    
    M1 --> E1
    M1 --> E2
    M1 --> E3
    
    M2 --> E1
    M2 --> E2
    M2 --> E3
    
    M3 --> E1
    M3 --> E2
    M3 --> E3
    
    E1 --> T1
    E1 --> T2
    E1 --> T3
    
    E2 --> T1
    E2 --> T2
    E2 --> T3
    
    E3 --> T1
```

## Компоненты системы

### 1. Workflow файлы

```
.github/workflows/
├── ci-main.yml              # Главный координатор
├── modules/
│   ├── sds-replicated-volume.yml
│   ├── sds-node-configurator.yml
│   └── data-export.yml
└── utils/
    ├── setup-cluster.yml
    ├── cleanup.yml
    └── reporting.yml
```

### 2. Тестовые среды

| Среда | Runner | Кластер | Время | Ресурсы |
|-------|--------|---------|-------|---------|
| local | GitHub-hosted | N/A | 5-10 мин | Ограниченные |
| bare-metal | Self-hosted | Физический | 20-60 мин | Полные |
| hypervisor | Self-hosted | Виртуальный | 30-120 мин | VM |

### 3. Типы тестов

| Тип | Описание | Время | Покрытие |
|-----|----------|-------|----------|
| smoke | Базовые проверки | 5-15 мин | Основная функциональность |
| integration | Интеграционные тесты | 20-60 мин | Взаимодействие компонентов |
| full | Полный набор | 60-120 мин | Все сценарии + edge cases |

## Жизненный цикл теста

```mermaid
stateDiagram-v2
    [*] --> Triggered: Push/PR/Schedule
    Triggered --> CodeQuality: Start pipeline
    CodeQuality --> SmokeTests: Code quality passed
    CodeQuality --> Failed: Code quality failed
    
    SmokeTests --> IntegrationTests: Smoke tests passed
    SmokeTests --> Failed: Smoke tests failed
    
    IntegrationTests --> SetupCluster: Start integration
    SetupCluster --> RunTests: Cluster ready
    SetupCluster --> Failed: Setup failed
    
    RunTests --> CollectResults: Tests completed
    RunTests --> Failed: Tests failed
    
    CollectResults --> GenerateReport: Results collected
    GenerateReport --> SendNotifications: Report generated
    SendNotifications --> Cleanup: Notifications sent
    
    Cleanup --> Success: Cleanup completed
    Cleanup --> Failed: Cleanup failed
    
    Failed --> [*]
    Success --> [*]
```

## Мониторинг и метрики

### Ключевые метрики

1. **Время выполнения**
   - Общее время пайплайна
   - Время каждого этапа
   - Время тестов по модулям

2. **Успешность**
   - Процент успешных тестов
   - Количество флаки тестов
   - Стабильность по модулям

3. **Ресурсы**
   - Использование CPU/памяти
   - Количество узлов
   - Доступность кластеров

### Алерты

- Критические сбои в основных модулях
- Превышение времени выполнения
- Недоступность тестовых сред
- Проблемы с ресурсами

## Безопасность

### Управление секретами

- GitHub Secrets для конфигураций кластеров
- SSH ключи для доступа к узлам
- Лицензионные ключи Deckhouse
- Webhook URL для уведомлений

### Контроль доступа

- RBAC для Kubernetes кластеров
- Ограниченный SSH доступ
- Изоляция тестовых namespace
- Аудит доступа

## Масштабирование

### Горизонтальное

- Self-hosted runners для различных типов тестов
- Пул runners для bare metal тестов
- Отдельные runners для hypervisor тестов

### Вертикальное

- Кэширование Docker образов
- Параллельное выполнение независимых тестов
- Инкрементальное тестирование

## Развитие

### Планы развития

1. **Краткосрочные (1-2 месяца)**
   - Оптимизация времени выполнения
   - Улучшение отчетности
   - Добавление метрик

2. **Среднесрочные (3-6 месяцев)**
   - Добавление новых модулей
   - Интеграция с другими системами
   - Автоматическое масштабирование

3. **Долгосрочные (6+ месяцев)**
   - ML для предсказания сбоев
   - Автоматическое исправление проблем
   - Интеграция с мониторингом
