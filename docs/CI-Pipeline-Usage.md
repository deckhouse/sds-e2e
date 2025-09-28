# Руководство по использованию CI пайплайна

## Обзор

CI пайплайн для e2e тестирования модулей Deckhouse автоматизирует процесс тестирования различных компонентов системы хранения данных. Пайплайн поддерживает несколько тестовых сред и типов тестов.

## Структура пайплайна

### Основные компоненты

1. **ci-main.yml** - Главный workflow, координирующий выполнение всех тестов
2. **modules/** - Специфичные workflow для каждого модуля
3. **utils/** - Утилитарные workflow для настройки и очистки

### Поддерживаемые модули

- `sds-replicated-volume` - Тестирование LVM и Linstor функциональности
- `sds-node-configurator` - Тестирование управления LVM Volume Groups
- `data-export` - Тестирование экспорта данных

### Тестовые среды

- **local** - Быстрые smoke тесты на GitHub-hosted runners
- **bare-metal** - Полные тесты на физических серверах
- **hypervisor** - Тесты с виртуализацией через Deckhouse

### Типы тестов

- **smoke** - Быстрые базовые проверки
- **integration** - Интеграционные тесты
- **full** - Полный набор тестов включая edge cases

## Настройка

### 1. GitHub Secrets

Настройте следующие секреты в настройках репозитория:

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
```

### 2. Self-hosted Runners

Для bare-metal и hypervisor тестов настройте self-hosted runners:

```yaml
# .github/runner-labels.yml
bare-metal:
  - ubuntu-20.04
  - 8cpu-16gb
  - k8s-cluster

hypervisor:
  - ubuntu-20.04
  - 16cpu-32gb
  - k8s-cluster
  - virtualization
```

### 3. Конфигурация кластеров

#### Bare Metal кластер
- Минимум 3 узла
- Установленный Deckhouse
- Модули: sds-replicated-volume, sds-node-configurator
- SSH доступ к узлам

#### Hypervisor кластер
- Минимум 1 узел с поддержкой виртуализации
- Установленный Deckhouse
- Модуль virtualization
- Достаточные ресурсы для создания VM

## Запуск тестов

### Автоматический запуск

Тесты запускаются автоматически при:
- Push в ветки `main` или `develop`
- Создании Pull Request
- Еженедельно по расписанию (понедельник 2:00 UTC)

### Ручной запуск

1. Перейдите в раздел Actions
2. Выберите "CI - Main Pipeline"
3. Нажмите "Run workflow"
4. Выберите параметры:
   - **Module**: all, sds-replicated-volume, sds-node-configurator, data-export
   - **Environment**: bare-metal, hypervisor, local
   - **Test Type**: smoke, integration, full

### Запуск через GitHub CLI

```bash
# Запуск smoke тестов для sds-replicated-volume
gh workflow run "CI - Main Pipeline" \
  -f module=sds-replicated-volume \
  -f environment=local \
  -f test_type=smoke

# Запуск полных тестов на bare metal
gh workflow run "CI - Main Pipeline" \
  -f module=all \
  -f environment=bare-metal \
  -f test_type=full
```

## Управление тестами через лейблы

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

## Мониторинг и отладка

### Просмотр результатов

1. **GitHub Actions UI** - Основной интерфейс для мониторинга
2. **Артефакты** - Скачивание логов и отчетов
3. **Slack уведомления** - Мгновенные уведомления о результатах

### Анализ логов

```bash
# Скачивание артефактов
gh run download <run-id>

# Просмотр логов конкретного job
gh run view <run-id> --log --job="Test SDS Replicated Volume"
```

### Отладка проблем

#### Частые проблемы

1. **Timeout ошибки**
   - Увеличьте timeout в workflow
   - Проверьте доступность кластера
   - Убедитесь в достаточности ресурсов

2. **SSH подключение**
   - Проверьте SSH ключи
   - Убедитесь в доступности хоста
   - Проверьте firewall настройки

3. **Kubernetes доступ**
   - Проверьте kubeconfig
   - Убедитесь в корректности контекста
   - Проверьте RBAC права

#### Команды для отладки

```bash
# Проверка доступности кластера
kubectl cluster-info

# Проверка узлов
kubectl get nodes

# Проверка подов
kubectl get pods -A

# Проверка модулей Deckhouse
kubectl get moduleconfigs
```

## Кастомизация

### Добавление нового модуля

1. Создайте новый workflow в `modules/`
2. Добавьте модуль в матрицу в `ci-main.yml`
3. Обновите документацию

Пример:

```yaml
# .github/workflows/modules/new-module.yml
name: New Module Tests

on:
  workflow_call:
    inputs:
      environment:
        required: true
        type: string
      test_type:
        required: true
        type: string
    secrets:
      KUBECONFIG_BARE_METAL:
        required: false
      # ... другие секреты

jobs:
  test-new-module:
    name: Test New Module
    runs-on: [self-hosted, bare-metal]
    steps:
      # ... шаги тестирования
```

### Добавление новых тестовых сред

1. Настройте self-hosted runner с соответствующими лейблами
2. Добавьте среду в матрицу workflow
3. Обновите утилитарные workflow

### Кастомизация уведомлений

```yaml
# В workflow добавьте кастомные уведомления
- name: Custom notification
  uses: 8398a7/action-slack@v3
  with:
    status: ${{ job.status }}
    channel: '#custom-channel'
    text: |
      Custom message for ${{ inputs.module_name }}
      Results: ${{ steps.test-results.outputs.summary }}
```

## Лучшие практики

### Разработка тестов

1. **Идемпотентность** - Тесты должны быть повторяемыми
2. **Изоляция** - Каждый тест должен быть независимым
3. **Очистка** - Всегда очищайте ресурсы после тестов
4. **Таймауты** - Устанавливайте разумные таймауты

### Управление ресурсами

1. **Namespace isolation** - Используйте уникальные namespace для каждого запуска
2. **Resource limits** - Устанавливайте лимиты на ресурсы
3. **Cleanup** - Автоматическая очистка после тестов
4. **Monitoring** - Мониторинг использования ресурсов

### Безопасность

1. **Secrets management** - Используйте GitHub Secrets
2. **Access control** - Ограничивайте доступ к кластерам
3. **Audit logging** - Ведите логи доступа
4. **Regular updates** - Регулярно обновляйте зависимости

## Troubleshooting

### Проблемы с производительностью

```bash
# Проверка ресурсов кластера
kubectl top nodes
kubectl top pods -A

# Проверка событий
kubectl get events --sort-by=.metadata.creationTimestamp

# Проверка логов Deckhouse
kubectl logs -n d8-system deployment/deckhouse
```

### Проблемы с сетью

```bash
# Проверка DNS
kubectl run test-pod --image=busybox --rm -it -- nslookup kubernetes.default

# Проверка сетевых политик
kubectl get networkpolicies -A

# Проверка CNI
kubectl get pods -n kube-system | grep -E "(cilium|flannel|calico)"
```

### Проблемы с хранилищем

```bash
# Проверка StorageClasses
kubectl get storageclass

# Проверка PV/PVC
kubectl get pv,pvc -A

# Проверка LVM ресурсов
kubectl get lvg,lv -A
```

## Поддержка

### Получение помощи

1. **GitHub Issues** - Создайте issue для багов и feature requests
2. **Slack** - Используйте канал #ci-support для быстрых вопросов
3. **Документация** - Проверьте ADR и другие документы

### Вклад в развитие

1. Fork репозитория
2. Создайте feature branch
3. Внесите изменения
4. Создайте Pull Request
5. Дождитесь review и merge

### Отчеты о проблемах

При создании issue укажите:
- Версию Go и Kubernetes
- Тип тестовой среды
- Полные логи ошибок
- Шаги для воспроизведения
- Ожидаемое и фактическое поведение
