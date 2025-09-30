# Настройка интеграции e2e тестов через лейблы

## Обзор

Данный документ описывает процесс настройки интеграции e2e тестов с модулями Deckhouse через систему лейблов GitHub и GitLab.

## Предварительные требования

### 1. Внешний кластер
- Kubernetes кластер с достаточными ресурсами
- Доступ к кластеру через kubeconfig
- Возможность развертывания DVP (Deckhouse Virtual Platform)

### 2. Секреты и переменные
Необходимо настроить следующие секреты/переменные в репозиториях:

#### В репозитории модуля (GitHub):
```bash
# Токен для отправки webhook в sds-e2e
E2E_TRIGGER_TOKEN=<github-token-with-repo-access>
```

#### В репозитории модуля (GitLab):
```bash
# Токен для отправки webhook в sds-e2e
E2E_TRIGGER_TOKEN=<github-token-with-repo-access>

# Токен для отправки комментариев в MR
GITLAB_TOKEN=<gitlab-token-with-api-access>
```

#### В репозитории sds-e2e:
```bash
# Доступ к внешнему кластеру
EXTERNAL_CLUSTER_KUBECONFIG=<base64-encoded-kubeconfig>

# SSH доступ (если требуется)
SSH_PRIVATE_KEY=<private-ssh-key>
SSH_HOST=user@hostname

# Лицензия Deckhouse (если требуется)
DECKHOUSE_LICENSE=<license-key>

# Токен для отправки комментариев в GitLab MR
GITLAB_TOKEN=<gitlab-token-with-api-access>
```

## Настройка модуля

### 1. Создание лейблов

#### GitHub репозитории
В настройках репозитория модуля создайте следующие лейблы:

#### GitLab репозитории
В настройках проекта модуля создайте следующие лейблы:

#### Основные лейблы:
- `e2e:run` - запустить e2e тесты
- `e2e:skip` - пропустить e2e тесты
- `e2e:smoke` - запустить только smoke тесты
- `e2e:full` - запустить полный набор тестов

#### Лейблы модулей:
- `e2e:sds-node-configurator` - тесты для sds-node-configurator
- `e2e:data-export` - тесты для data-export
- `e2e:sds-replicated-volume` - тесты для sds-replicated-volume

#### Лейблы окружения:
- `e2e:bare-metal` - тесты на bare-metal
- `e2e:hypervisor` - тесты на hypervisor

#### Лейблы приоритета:
- `e2e:priority:high` - высокий приоритет
- `e2e:priority:low` - низкий приоритет

### 2. Настройка CI/CD

#### GitHub Actions (для GitHub репозиториев)

Скопируйте файл `templates/github-actions-module-integration.yml` в `.github/workflows/e2e-integration.yml` в репозитории модуля.

Обновите переменные в файле:
```yaml
env:
  E2E_REPO: "deckhouse/sds-e2e"
  MODULE_NAME: "sds-node-configurator"  # Замените на имя вашего модуля
```

#### GitLab CI (для GitLab репозиториев)

Скопируйте файл `templates/gitlab-ci-module-integration.yml` в `.gitlab-ci.yml` в репозитории модуля.

Обновите переменные в файле:
```yaml
variables:
  E2E_REPO: "deckhouse/sds-e2e"
  MODULE_NAME: "sds-node-configurator"  # Замените на имя вашего модуля
```

### 3. Настройка секретов/переменных

#### GitHub репозитории
Добавьте секрет `E2E_TRIGGER_TOKEN` в настройки репозитория:
1. Перейдите в Settings → Secrets and variables → Actions
2. Нажмите "New repository secret"
3. Добавьте секрет с токеном GitHub, имеющим доступ к репозиторию sds-e2e

#### GitLab репозитории
Добавьте переменные в настройки проекта:
1. Перейдите в Settings → CI/CD → Variables
2. Добавьте переменные:
   - `E2E_TRIGGER_TOKEN` - токен GitHub для отправки webhook
   - `GITLAB_TOKEN` - токен GitLab для отправки комментариев в MR

## Настройка sds-e2e репозитория

### 1. Настройка GitHub Actions

Скопируйте файл `templates/github-actions-e2e-handler-multiplatform.yml` в `.github/workflows/module-e2e-handler.yml` в репозитории sds-e2e.

### 2. Настройка секретов

Добавьте необходимые секреты в настройки репозитория sds-e2e:

#### EXTERNAL_CLUSTER_KUBECONFIG
```bash
# Кодируем kubeconfig в base64
cat ~/.kube/config | base64 -w 0
```

#### SSH_PRIVATE_KEY (если требуется)
```bash
# Приватный ключ для SSH доступа
cat ~/.ssh/id_rsa
```

#### SSH_HOST
```bash
# Хост для SSH доступа
user@hostname
```

#### DECKHOUSE_LICENSE (если требуется)
```bash
# Лицензия Deckhouse
your-license-key
```

### 3. Настройка внешнего кластера

#### Создание namespace для тестов
```bash
kubectl create namespace e2e-tests
```

#### Развертывание DVP
```bash
kubectl apply -f templates/deployment-and-roles.yaml -n e2e-tests
```

#### Проверка готовности
```bash
kubectl wait --for=condition=ready pod -l app=dvp -n e2e-tests --timeout=300s
```

## Использование

### 1. Добавление лейблов в PR/MR

Разработчики могут добавлять лейблы в PR (GitHub) или MR (GitLab) для управления запуском e2e тестов:

#### Пример 1: Smoke тесты для sds-node-configurator (GitHub)
```bash
# Лейблы в PR:
- e2e:run
- e2e:smoke
- e2e:sds-node-configurator
- e2e:bare-metal
```

#### Пример 2: Полные тесты для data-export (GitLab)
```bash
# Лейблы в MR:
- e2e:run
- e2e:full
- e2e:data-export
- e2e:hypervisor
```

#### Пример 3: Пропуск тестов (GitHub/GitLab)
```bash
# Лейблы в PR/MR:
- e2e:skip
```

#### Пример 4: Высокий приоритет (GitLab)
```bash
# Лейблы в MR:
- e2e:run
- e2e:smoke
- e2e:sds-node-configurator
- e2e:priority:high
```

### 2. Ручной запуск тестов

#### GitHub Actions
Через GitHub Actions можно запустить тесты вручную:

1. Перейдите в Actions → Module E2E Tests Handler
2. Нажмите "Run workflow"
3. Выберите параметры:
   - Source repository
   - Source PR number
   - Module
   - Test type
   - Environment
   - Priority
   - CI Platform

#### GitLab CI
Через GitLab CI можно запустить тесты вручную:

1. Перейдите в CI/CD → Pipelines
2. Нажмите "Run pipeline"
3. Выберите переменные:
   - `E2E_TRIGGER_TOKEN`
   - `GITLAB_TOKEN`
   - `MODULE_NAME`
   - `TEST_TYPE`
   - `ENVIRONMENT`
   - `PRIORITY`

### 3. Мониторинг результатов

#### Просмотр результатов в PR/MR
После выполнения тестов в PR/MR появится комментарий с результатами:
- Статус тестов (PASSED/FAILED/CANCELLED)
- Детали выполнения
- Ссылка на артефакты

#### Просмотр логов
**GitHub:**
1. Перейдите в Actions в репозитории sds-e2e
2. Найдите выполнение "Module E2E Tests Handler"
3. Просмотрите логи каждого шага

**GitLab:**
1. Перейдите в CI/CD → Pipelines в репозитории sds-e2e
2. Найдите выполнение "Module E2E Tests Handler"
3. Просмотрите логи каждого job

#### Скачивание артефактов
**GitHub:**
1. В результатах выполнения найдите "Artifacts"
2. Скачайте архив с результатами тестов

**GitLab:**
1. В результатах выполнения найдите "Job artifacts"
2. Скачайте архив с результатами тестов

## Отладка

### Частые проблемы

#### 1. Тесты не запускаются
- Проверьте наличие лейбла `e2e:run` в PR
- Убедитесь, что нет лейбла `e2e:skip`
- Проверьте настройку секрета `E2E_TRIGGER_TOKEN`

#### 2. Ошибки подключения к кластеру
- Проверьте корректность `EXTERNAL_CLUSTER_KUBECONFIG`
- Убедитесь в доступности внешнего кластера
- Проверьте права доступа к кластеру

#### 3. Ошибки развертывания DVP
- Проверьте доступность шаблонов в `templates/`
- Убедитесь в достаточности ресурсов кластера
- Проверьте логи развертывания

#### 4. Тесты падают
- Проверьте логи выполнения тестов
- Убедитесь в корректности конфигурации тестов
- Проверьте доступность тестовых ресурсов

### Логирование

#### Включение подробного логирования
```yaml
# В workflow добавьте:
- name: Enable verbose logging
  run: |
    export GO_TEST_VERBOSE=1
    export KUBECTL_VERBOSE=1
```

#### Просмотр логов DVP
```bash
kubectl logs -l app=dvp -n e2e-tests
```

#### Просмотр логов тестов
```bash
kubectl logs -l app=e2e-tests -n e2e-tests
```

## Масштабирование

### Горизонтальное масштабирование
- Настройка нескольких внешних кластеров
- Балансировка нагрузки между кластерами
- Географическое распределение

### Вертикальное масштабирование
- Увеличение ресурсов кластера
- Оптимизация DVP
- Кэширование образов

## Безопасность

### Управление доступом
- Использование RBAC для ограничения доступа
- Временные токены для выполнения тестов
- Изоляция тестовых сред

### Очистка ресурсов
- Автоматическая очистка после выполнения тестов
- Лимиты времени жизни ресурсов
- Мониторинг использования ресурсов

## Мониторинг

### Метрики
- Количество запусков по лейблам
- Время выполнения тестов
- Процент успешных тестов
- Использование ресурсов кластера

### Уведомления
- Slack/Teams уведомления о результатах
- Email уведомления о критических ошибках
- Интеграция с системами мониторинга
