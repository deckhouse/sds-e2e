# CI Pipeline для e2e тестирования модулей Deckhouse

## 🚀 Быстрый старт

### 1. Настройка секретов

#### GitHub Actions
Добавьте следующие секреты в настройки репозитория (Settings → Secrets and variables → Actions):

```bash
# Конфигурации кластеров (base64 encoded)
KUBECONFIG_BARE_METAL=<base64-encoded-kubeconfig>
KUBECONFIG_HYPERVISOR=<base64-encoded-kubeconfig>

# SSH доступ
SSH_PRIVATE_KEY=<private-ssh-key>
SSH_HOST=user@hostname

# Лицензия Deckhouse
DECKHOUSE_LICENSE=<license-key>

# Уведомления (опционально)
SLACK_WEBHOOK=<slack-webhook-url>
TEAMS_WEBHOOK=<teams-webhook-url>
```

#### GitLab CI
Добавьте следующие переменные в настройки проекта (Settings → CI/CD → Variables):

```bash
# File type variables (зашифрованы)
KUBECONFIG_BARE_METAL=<kubeconfig-file>
KUBECONFIG_HYPERVISOR=<kubeconfig-file>
SSH_PRIVATE_KEY=<private-key-file>

# Variable type variables (зашифрованы)
SSH_HOST=user@hostname
DECKHOUSE_LICENSE=<license-key>
SLACK_WEBHOOK=<slack-webhook-url>
TEAMS_WEBHOOK=<teams-webhook-url>
GITHUB_TOKEN=<github-token>  # для создания issues
```

### 2. Настройка runners

#### GitHub Actions
Для bare-metal и hypervisor тестов настройте self-hosted runners с лейблами:
- `self-hosted`
- `bare-metal` или `hypervisor`

#### GitLab CI
Для bare-metal и hypervisor тестов настройте GitLab runners с тегами:
- `bare-metal`, `k8s-cluster`, `lvm-support`
- `hypervisor`, `k8s-cluster`, `virtualization`, `nested-vm`

### 3. Запуск тестов

#### Автоматический запуск
Тесты запускаются автоматически при:
- Push в `main`/`develop`
- Pull Request / Merge Request
- Еженедельно по расписанию

#### Ручной запуск

**GitHub Actions:**
1. Перейдите в Actions → "CI - Main Pipeline"
2. Нажмите "Run workflow"
3. Выберите параметры:
   - **Module**: all, sds-replicated-volume, sds-node-configurator, data-export
   - **Environment**: bare-metal, hypervisor, local
   - **Test Type**: smoke, integration, full
   - **Go Version**: 1.21, 1.22

**GitLab CI:**
1. Перейдите в CI/CD → Pipelines
2. Нажмите "Run pipeline"
3. Выберите параметры:
   - **MODULE_UNDER_TEST**: all, sds-replicated-volume, sds-node-configurator, data-export
   - **TEST_ENVIRONMENT**: bare-metal, hypervisor, local
   - **TEST_TYPE**: smoke, integration, full

## 📊 Поддерживаемые модули

| Модуль | Описание | Тестовые среды |
|--------|----------|----------------|
| `sds-replicated-volume` | LVM и Linstor функциональность | bare-metal, hypervisor, local |
| `sds-node-configurator` | Управление LVM Volume Groups | bare-metal, hypervisor, local |
| `data-export` | Экспорт данных | bare-metal, hypervisor, local |

## 🏗️ Тестовые среды

| Среда | Описание | Время выполнения | Ресурсы | GitHub Actions | GitLab CI |
|-------|----------|------------------|---------|----------------|-----------|
| **local** | Hosted runners | 5-10 мин | Ограниченные | ✅ | ✅ |
| **bare-metal** | Физические серверы | 20-60 мин | Полные | ✅ | ✅ |
| **hypervisor** | Виртуализация | 30-120 мин | VM | ✅ | ✅ |

## 📈 Типы тестов

| Тип | Описание | Время | Покрытие |
|-----|----------|-------|----------|
| **smoke** | Базовые проверки | 5-15 мин | Основная функциональность |
| **integration** | Интеграционные тесты | 20-60 мин | Взаимодействие компонентов |
| **full** | Полный набор | 60-120 мин | Все сценарии + edge cases |

## 🎯 Управление тестами через лейблы

### Лейблы в PR/MR описании

Для управления тестами используйте специальные лейблы в описании Pull Request или Merge Request:

#### Пропуск тестов

```bash
# Пропустить все e2e тесты
[skip-e2e]

# Пропустить только медленные тесты
[skip-slow-tests]

# Пропустить тесты для конкретного модуля
[skip-e2e:sds-replicated-volume]
```

#### Принудительный запуск

```bash
# Запустить все тесты, включая медленные
[force-full-e2e]

# Запустить stress тесты
[force-stress-tests]

# Запустить тесты для всех модулей
[force-all-modules]
```

#### Выбор среды

```bash
# Запустить только на bare-metal
[env:bare-metal]

# Запустить только на hypervisor
[env:hypervisor]

# Запустить на обеих средах
[env:all]
```

#### Выбор конкретных тестов

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

#### Указание ветки модуля

```bash
# Тестировать с ветки develop модуля
[module-branch:develop]

# Тестировать с конкретной ветки модуля
[module-branch:hotfix/storage-issue]

# Тестировать с тега модуля
[module-tag:v1.2.3]
```

#### Приоритизация

```bash
# Высокий приоритет - запустить немедленно
[priority:high]

# Низкий приоритет - запустить в свободное время
[priority:low]
```

## 🔧 Команды для разработчиков

### Локальный запуск тестов

```bash
# Smoke тесты
cd testkit_v2
go test -v -timeout 10m ./tests/00_healthcheck_test.go -stand local -verbose

# Конкретный модуль
go test -v -timeout 30m ./tests/01_sds_nc_test.go -stand metal -verbose -debug

# С кастомными параметрами
go test -v -timeout 60m ./tests/... \
  -stand metal \
  -verbose \
  -debug \
  -kconfig /path/to/kubeconfig \
  -sshhost user@host \
  -sshkey /path/to/key \
  -namespace test-e2e-$(date +%s)
```

### GitHub CLI

```bash
# Запуск smoke тестов
gh workflow run "CI - Main Pipeline" \
  -f module=sds-replicated-volume \
  -f environment=local \
  -f test_type=smoke

# Просмотр результатов
gh run list --workflow="CI - Main Pipeline"
gh run view <run-id>
```

### GitLab CLI

```bash
# Запуск пайплайна
glab ci run --variable MODULE_UNDER_TEST=sds-replicated-volume \
  --variable TEST_ENVIRONMENT=bare-metal \
  --variable TEST_TYPE=integration

# Просмотр результатов
glab ci list
glab ci view <pipeline-id>
```

## 📋 Мониторинг

### Просмотр результатов
- **GitHub Actions UI** - Основной интерфейс для GitHub
- **GitLab CI UI** - Основной интерфейс для GitLab
- **Артефакты** - Логи и отчеты
- **Slack/Teams** - Уведомления (если настроено)

### Ключевые метрики
- Время выполнения тестов
- Процент успешных тестов
- Количество флаки тестов
- Покрытие тестами

## 🚨 Troubleshooting

### Частые проблемы

1. **Timeout ошибки**
   ```bash
   # Проверьте доступность кластера
   kubectl cluster-info
   kubectl get nodes
   ```

2. **SSH подключение**
   ```bash
   # Проверьте SSH доступ
   ssh -i /path/to/key user@host
   ```

3. **Kubernetes доступ**
   ```bash
   # Проверьте kubeconfig
   kubectl config current-context
   kubectl get pods -A
   ```

### Отладка

**GitHub Actions:**
```bash
# Скачивание логов
gh run download <run-id>

# Просмотр логов job
gh run view <run-id> --log --job="Test SDS Replicated Volume"

# Проверка ресурсов кластера
kubectl top nodes
kubectl top pods -A
```

**GitLab CI:**
```bash
# Скачивание артефактов
glab ci download <pipeline-id>

# Просмотр логов job
glab ci view <pipeline-id> --log

# Проверка ресурсов кластера
kubectl top nodes
kubectl top pods -A
```

## 📚 Документация

- [ADR-001: Архитектура CI пайплайна](docs/ADR-001-CI-Pipeline-Architecture.md)
- [Руководство по использованию](docs/CI-Pipeline-Usage.md)
- [Структура тестов](testkit_v2/README.md)

## 🤝 Поддержка

- **GitHub Issues** - Баги и feature requests
- **Slack** - #ci-support для быстрых вопросов
- **Документация** - ADR и руководства

## 🔄 Обновления

Пайплайн автоматически обновляется при изменениях в:
- `.github/workflows/` - GitHub Actions workflow файлы
- `.gitlab-ci.yml` и `.gitlab-ci/` - GitLab CI конфигурация
- `testkit_v2/` - Тестовый код
- `images/` - Docker образы

## 🎯 Ключевые особенности

- **Кроссплатформенность**: Поддержка GitHub Actions и GitLab CI
- **Модульность**: Легкое добавление новых модулей
- **Масштабируемость**: Матричное тестирование различных комбинаций
- **Безопасность**: Централизованное управление секретами
- **Мониторинг**: Комплексная система отчетности и алертов
- **Автоматизация**: Полная автоматизация процесса тестирования

---

**Примечание**: Для полной функциональности требуется настройка self-hosted runners (GitHub Actions) или GitLab runners и доступ к тестовым кластерам.
