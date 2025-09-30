# Управление Image Tags для модулей

Этот документ описывает способы указания конкретных `imageTag` для модулей Deckhouse при запуске e2e тестов.

## 🎯 Способы указания тегов

### 1. Флаги командной строки

```bash
# Указание тегов через флаги
go test -v ./tests/... \
  -sds-node-configurator-tag="v1.0.0" \
  -sds-replicated-volume-tag="v2.1.0" \
  -sds-local-volume-tag="v1.5.0" \
  -data-export-tag="v1.2.0"
```

### 2. Переменные окружения

```bash
# Установка переменных окружения
export SDS_NODE_CONFIGURATOR_TAG="v1.0.0"
export SDS_REPLICATED_VOLUME_TAG="v2.1.0"
export SDS_LOCAL_VOLUME_TAG="v1.5.0"
export DATA_EXPORT_TAG="v1.2.0"

# Запуск тестов
go test -v ./tests/...
```

### 3. Комбинированный подход

Флаги командной строки имеют приоритет над переменными окружения:

```bash
# Переменная окружения
export SDS_NODE_CONFIGURATOR_TAG="v1.0.0"

# Флаг переопределит переменную окружения
go test -v ./tests/... -sds-node-configurator-tag="v1.1.0"
```

## 📋 Доступные флаги

| Флаг | Переменная окружения | Модуль | По умолчанию |
|------|---------------------|--------|--------------|
| `-sds-node-configurator-tag` | `SDS_NODE_CONFIGURATOR_TAG` | sds-node-configurator | `main` |
| `-sds-replicated-volume-tag` | `SDS_REPLICATED_VOLUME_TAG` | sds-replicated-volume | `main` |
| `-sds-local-volume-tag` | `SDS_LOCAL_VOLUME_TAG` | sds-local-volume | `main` |
| `-data-export-tag` | `DATA_EXPORT_TAG` | data-export | `main` |

## 🏷️ Поддерживаемые форматы тегов

- **`main`** - основная ветка (по умолчанию)
- **`prXX`** - для GitHub Pull Request (например, `pr123`, `pr456`)
- **`mrXX`** - для GitLab Merge Request (например, `mr789`, `mr101`)

## 🚀 Примеры использования

### Тестирование конкретной версии sds-node-configurator

```bash
# Через флаг
go test -v ./tests/05_sds_node_configurator_test.go -sds-node-configurator-tag="pr123"

# Через переменную окружения
export SDS_NODE_CONFIGURATOR_TAG="pr123"
go test -v ./tests/05_sds_node_configurator_test.go
```

### Тестирование нескольких модулей с разными версиями

```bash
go test -v ./tests/... \
  -sds-node-configurator-tag="pr123" \
  -sds-replicated-volume-tag="pr456" \
  -data-export-tag="main"
```

### Тестирование с тегами из CI/CD

```bash
# В CI/CD пайплайне
export SDS_NODE_CONFIGURATOR_TAG="pr${GITHUB_PR_NUMBER}"
export SDS_REPLICATED_VOLUME_TAG="pr${GITHUB_PR_NUMBER}"
go test -v ./tests/...
```

## 🔧 Технические детали

### Генерация resources.yml

При запуске тестов автоматически генерируется файл `resources.yml` на основе шаблона `resources.yml.tpl` с подстановкой указанных тегов:

```yaml
# resources.yml (сгенерированный)
apiVersion: deckhouse.io/v1alpha1
kind: ModulePullOverride
metadata:
  name: sds-node-configurator
spec:
  imageTag: pr123  # ← указанный тег
  scanInterval: 15s
  source: deckhouse
```

### Приоритет настроек

1. **Флаги командной строки** (высший приоритет)
2. **Переменные окружения**
3. **Значения по умолчанию** (`main`)

### Валидация

- Теги должны соответствовать формату Docker тегов
- Поддерживаются следующие форматы:
  - `main` - основная ветка
  - `prXX` - для GitHub Pull Request (например, `pr123`)
  - `mrXX` - для GitLab Merge Request (например, `mr456`)

## 🎯 Использование в CI/CD

### GitHub Actions

```yaml
- name: Run e2e tests with specific tags
  env:
    SDS_NODE_CONFIGURATOR_TAG: pr${{ github.event.number }}
    SDS_REPLICATED_VOLUME_TAG: pr${{ github.event.number }}
  run: |
    go test -v ./tests/...
```

### GitLab CI

```yaml
test:
  variables:
    SDS_NODE_CONFIGURATOR_TAG: mr${{ gitlab.merge_request.iid }}
    SDS_REPLICATED_VOLUME_TAG: mr${{ gitlab.merge_request.iid }}
  script:
    - go test -v ./tests/...
```

## 🔄 Одновременное тестирование GitHub и GitLab

Для тестирования модулей из разных репозиториев (GitHub и GitLab) одновременно, используйте разные теги:

```bash
# Тестирование с модулями из GitHub PR и GitLab MR
go test -v ./tests/... \
  -sds-node-configurator-tag="pr123" \
  -sds-replicated-volume-tag="mr456" \
  -data-export-tag="main"
```

### Автоматическое определение платформы

Можно использовать переменные окружения для автоматического определения формата тегов:

```bash
# Скрипт для автоматического определения тегов
if [ -n "$GITHUB_PR_NUMBER" ]; then
  export SDS_NODE_CONFIGURATOR_TAG="pr${GITHUB_PR_NUMBER}"
elif [ -n "$GITLAB_MERGE_REQUEST_IID" ]; then
  export SDS_NODE_CONFIGURATOR_TAG="mr${GITLAB_MERGE_REQUEST_IID}"
else
  export SDS_NODE_CONFIGURATOR_TAG="main"
fi

go test -v ./tests/...
```

## 🛠️ Утилитарные скрипты

### Автоматическое определение тегов

Скрипт `scripts/auto-detect-tags.sh` автоматически определяет теги на основе переменных окружения:

```bash
# Запуск с автоматическим определением тегов
source scripts/auto-detect-tags.sh
go test -v ./tests/...
```

### Смешанное тестирование (GitHub + GitLab)

Скрипт `scripts/mixed-platform-test.sh` позволяет тестировать модули из разных платформ:

```bash
# Тестирование с модулями из GitHub PR и GitLab MR
./scripts/mixed-platform-test.sh -g pr123 -l mr456

# Указание конкретных тестов
./scripts/mixed-platform-test.sh -g pr123 -l mr456 -t ./tests/05_sds_node_configurator_test.go
```

## 🐛 Отладка

### Проверка текущих тегов

```bash
# Вывод текущих значений тегов
go test -v ./tests/... -verbose
```

### Логирование

При включенном флаге `-verbose` в логах будет видно, какие теги используются:

```
✎ Using module tags:
✎   sds-node-configurator: pr123
✎   sds-replicated-volume: pr456
✎   sds-local-volume: main
✎   data-export: mr789
```

## 📝 Примечания

- Изменение тегов требует пересоздания кластера
- Теги применяются ко всем тестам в рамках одного запуска
- Для тестирования разных версий в разных тестах используйте отдельные запуски
- Поддерживаются только форматы тегов: `main`, `prXX`, `mrXX`
- Скрипты в каталоге `scripts/` упрощают работу с тегами
