# Quick Start: E2E Smoke тесты через лейбл

## ⚡ За 5 минут

### 1. Подготовка (один раз)

```bash
cd /path/to/your-module-repo  # sds-node-configurator, sds-replicated-volume, и т.д.

# Создать лейбл
gh label create "e2e-smoke-test" --description "Run E2E smoke tests" --color "0E8A16"

# Добавить секреты
gh secret set E2E_CLUSTER_KUBECONFIG < e2e-cluster-kubeconfig-base64.txt  # Кластер где запускаются Job'ы
gh secret set KUBECONFIG_HYPERVISOR < hypervisor-kubeconfig-base64.txt    # Используется внутри тестов
gh secret set SSH_PRIVATE_KEY < ssh-key.txt
gh secret set SSH_HOST --body "user@hostname"
gh secret set DECKHOUSE_LICENSE < license.txt
```

**Важно:** Тесты запускаются **в кластере** (Kubernetes Job), а не на GitHub runners!

**Что такое smoke тесты:**
- Healthcheck кластера
- Базовая проверка модуля
- Время выполнения: 10-15 минут
- Окружение: hypervisor

### 2. Интеграция в CI (один раз)

Скопируйте код из `examples/build.yml` (строки 165-359) в ваш `.github/workflows/build.yml` или аналогичный файл.

### 3. Использование (каждый PR)

```bash
# Создать PR
gh pr create --title "My feature" --body "Description"

# Добавить лейбл для запуска smoke тестов
PR_NUMBER=$(gh pr list --head $(git branch --show-current) --json number --jq '.[0].number')
gh pr edit $PR_NUMBER --add-label "e2e-smoke-test"

# Тесты запустятся автоматически после сборки (10-15 минут)
```

## 📋 Основные команды

### Запуск тестов

```bash
# Smoke тесты (единственный вариант пока)
gh pr edit <PR_NUMBER> --add-label "e2e-smoke-test"
```

### Просмотр результатов

```bash
# Статус последнего workflow
gh run view

# Логи smoke тестов
gh run view --log --job run_e2e_smoke_tests

# Скачать артефакты с логами
gh run download

# Посмотреть логи в кластере
export KUBECONFIG=~/.kube/e2e-cluster-config
kubectl logs -n e2e-tests -l app=e2e-tests -f
```

### Отмена/повтор

```bash
# Удалить лейбл
gh pr edit <PR_NUMBER> --remove-label "e2e-smoke-test"

# Повторный запуск (если тесты упали)
gh run rerun <RUN_ID>
```

## 🎯 Что тестируется

### Smoke тесты включают:

**Для sds-node-configurator:**
- `00_healthcheck_test.go` - проверка здоровья кластера
- `01_sds_nc_test.go` - базовые тесты LVG

**Для sds-replicated-volume:**
- `00_healthcheck_test.go` - проверка здоровья кластера
- `03_sds_lv_test.go` - базовые тесты PVC

**Для data-export:**
- `00_healthcheck_test.go` - проверка здоровья кластера

⏱️ **Время выполнения:** 10-15 минут  
🌐 **Окружение:** Hypervisor (DVP cluster)

## 🐛 Troubleshooting

### Тесты не запускаются?

```bash
# Проверить наличие лейбла
gh pr view <PR_NUMBER> --json labels --jq '.labels[].name'

# Должен быть: e2e-smoke-test
```

### Ошибка доступа к e2e кластеру?

```bash
# Проверить kubeconfig для e2e кластера
echo "$E2E_CLUSTER_KUBECONFIG" | base64 -d | kubectl --kubeconfig=- cluster-info

# Обновить kubeconfig
cat ~/.kube/e2e-cluster-config | base64 -w 0 | gh secret set E2E_CLUSTER_KUBECONFIG
```

### Ошибка подключения к кластеру?

```bash
# Проверить kubeconfig
echo "$KUBECONFIG_HYPERVISOR" | base64 -d | kubectl --kubeconfig=- get nodes

# Обновить kubeconfig
cat ~/.kube/config-hypervisor | base64 -w 0 | gh secret set KUBECONFIG_HYPERVISOR
```

## 📚 Документация

- [Полная инструкция](E2E-Integration-Into-Build.md)
- [ADR-002](ADR-002-E2E-Tests-Integration-via-Labels.md)
- [PoC](POC-E2E-Labels.md)

## 💡 Полезные alias'ы

Добавьте в `~/.bashrc` или `~/.zshrc`:

```bash
# E2E smoke test alias
alias pr-e2e-smoke='gh pr edit $(gh pr list --head $(git branch --show-current) --json number --jq ".[0].number") --add-label "e2e-smoke-test"'
alias pr-e2e-logs='gh run view --log --job run_e2e_smoke_tests'
alias pr-e2e-watch='export KUBECONFIG=~/.kube/e2e-cluster-config && kubectl logs -n e2e-tests -l app=e2e-tests -f'
```

Использование:

```bash
# Создать PR и запустить smoke тесты одной командой
gh pr create --title "My feature" && pr-e2e-smoke

# Посмотреть логи в GitHub Actions
pr-e2e-logs

# Следить за логами в кластере в реальном времени
pr-e2e-watch
```

