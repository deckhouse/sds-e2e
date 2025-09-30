# Proof of Concept: E2E тесты через лейблы

## 🎯 Цель PoC

Проверить работоспособность интеграции e2e тестов через лейблы GitHub в репозитории sds-node-configurator.

## 📋 Шаги выполнения

### 1. Подготовка репозитория sds-node-configurator

#### 1.1 Создание лейблов

```bash
# Перейти в репозиторий sds-node-configurator
cd /path/to/sds-node-configurator

# Создать лейблы через GitHub CLI
gh label create "e2e:run" \
  --description "Run E2E tests" \
  --color "0E8A16"

gh label create "e2e:skip" \
  --description "Skip E2E tests" \
  --color "D93F0B"

gh label create "e2e:smoke" \
  --description "Run smoke tests" \
  --color "FBCA04"

gh label create "e2e:full" \
  --description "Run full tests" \
  --color "1D76DB"

gh label create "e2e:bare-metal" \
  --description "Run on bare-metal" \
  --color "5319E7"

gh label create "e2e:hypervisor" \
  --description "Run on hypervisor" \
  --color "C5DEF5"
```

#### 1.2 Настройка секретов

```bash
# Добавить секреты через GitHub CLI
gh secret set E2E_TRIGGER_TOKEN < token.txt
gh secret set KUBECONFIG_HYPERVISOR < kubeconfig-hypervisor-base64.txt
gh secret set SSH_PRIVATE_KEY < id_rsa.txt
gh secret set SSH_HOST --body "user@hostname"
gh secret set DECKHOUSE_LICENSE < license.txt
```

#### 1.3 Интеграция в CI

Добавьте в существующий `.github/workflows/ci.yml`:

```yaml
# ... существующие job'ы (build-and-checks, go-checks, etc.)

# Добавить новые job'ы для e2e тестов
parse-e2e-labels:
  name: Parse E2E Labels
  runs-on: ubuntu-latest
  if: github.event_name == 'pull_request'
  outputs:
    run-e2e: ${{ steps.parse.outputs.run-e2e }}
    skip-e2e: ${{ steps.parse.outputs.skip-e2e }}
    test-type: ${{ steps.parse.outputs.test-type }}
    environment: ${{ steps.parse.outputs.environment }}
  steps:
    - name: Parse PR labels
      id: parse
      run: |
        LABELS="${{ join(github.event.pull_request.labels.*.name, ' ') }}"
        echo "PR Labels: $LABELS"
        
        if echo "$LABELS" | grep -q "e2e:run"; then
          echo "run-e2e=true" >> $GITHUB_OUTPUT
        else
          echo "run-e2e=false" >> $GITHUB_OUTPUT
        fi
        
        if echo "$LABELS" | grep -q "e2e:skip"; then
          echo "skip-e2e=true" >> $GITHUB_OUTPUT
        else
          echo "skip-e2e=false" >> $GITHUB_OUTPUT
        fi
        
        if echo "$LABELS" | grep -q "e2e:smoke"; then
          echo "test-type=smoke" >> $GITHUB_OUTPUT
        elif echo "$LABELS" | grep -q "e2e:full"; then
          echo "test-type=full" >> $GITHUB_OUTPUT
        else
          echo "test-type=integration" >> $GITHUB_OUTPUT
        fi
        
        if echo "$LABELS" | grep -q "e2e:hypervisor"; then
          echo "environment=hypervisor" >> $GITHUB_OUTPUT
        else
          echo "environment=local" >> $GITHUB_OUTPUT
        fi

run-e2e-tests:
  name: Run E2E Tests
  runs-on: ${{ needs.parse-e2e-labels.outputs.environment == 'local' && 'ubuntu-latest' || 'self-hosted' }}
  needs: [parse-e2e-labels, build-and-checks] # Зависимость от сборки
  if: needs.parse-e2e-labels.outputs.run-e2e == 'true' && needs.parse-e2e-labels.outputs.skip-e2e == 'false'
  steps:
    - name: Checkout e2e tests
      uses: actions/checkout@v4
      with:
        repository: deckhouse/sds-e2e
        token: ${{ secrets.E2E_TRIGGER_TOKEN }}
        path: sds-e2e

    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.22'

    - name: Setup test environment
      run: |
        cd sds-e2e/testkit_v2
        mkdir -p ../../sds-e2e-cfg
        
        if [ "${{ needs.parse-e2e-labels.outputs.environment }}" = "hypervisor" ]; then
          echo "${{ secrets.KUBECONFIG_HYPERVISOR }}" | base64 -d > ../../sds-e2e-cfg/kube-hypervisor.config
          echo "${{ secrets.SSH_PRIVATE_KEY }}" > ../../sds-e2e-cfg/id_rsa_test
          chmod 600 ../../sds-e2e-cfg/id_rsa_test
        fi

    - name: Run E2E tests
      run: |
        cd sds-e2e/testkit_v2
        
        if [ "${{ needs.parse-e2e-labels.outputs.environment }}" = "local" ]; then
          go test -v -timeout 10m ./tests/00_healthcheck_test.go -stand local -verbose
        else
          go test -v -timeout 60m ./tests/*sds*_test.go \
            -stand metal \
            -hypervisorkconfig ../../sds-e2e-cfg/kube-hypervisor.config \
            -sshhost ${{ secrets.SSH_HOST }} \
            -sshkey ../../sds-e2e-cfg/id_rsa_test \
            -verbose -debug
        fi
      env:
        LICENSE_KEY: ${{ secrets.DECKHOUSE_LICENSE }}

    - name: Comment PR with results
      if: always()
      uses: actions/github-script@v6
      with:
        script: |
          const status = '${{ job.status }}' === 'success' ? '✅' : '❌';
          const comment = `## E2E Tests Results ${status}
          
          **Module:** sds-node-configurator
          **Environment:** ${{ needs.parse-e2e-labels.outputs.environment }}
          **Test Type:** ${{ needs.parse-e2e-labels.outputs.test-type }}
          
          Tests ${status === '✅' ? 'passed' : 'failed'}!`;
          
          github.rest.issues.createComment({
            issue_number: context.issue.number,
            owner: context.repo.owner,
            repo: context.repo.repo,
            body: comment
          });
```

### 2. Тестовый запуск

#### 2.1 Создание тестового PR

```bash
# Создать тестовую ветку
git checkout -b test/e2e-labels-poc

# Сделать небольшое изменение
echo "# PoC for e2e labels" >> README.md
git add README.md
git commit -m "test: PoC for e2e labels integration"

# Создать PR
git push origin test/e2e-labels-poc
gh pr create --title "PoC: E2E labels integration" --body "Testing e2e labels"
```

#### 2.2 Добавление лейблов

```bash
# Получить номер PR
PR_NUMBER=$(gh pr list --head test/e2e-labels-poc --json number --jq '.[0].number')

# Добавить лейблы
gh pr edit $PR_NUMBER --add-label "e2e:run"
gh pr edit $PR_NUMBER --add-label "e2e:smoke"
```

#### 2.3 Мониторинг выполнения

```bash
# Проверить статус workflow
gh run list --workflow=ci.yml --limit 1

# Посмотреть логи
gh run view --log
```

### 3. Проверка результатов

#### 3.1 Ожидаемое поведение

1. ✅ CI запускается при создании PR
2. ✅ Job `build-and-checks` выполняется как обычно
3. ✅ Job `parse-e2e-labels` парсит лейблы
4. ✅ Job `run-e2e-tests` запускается после успешной сборки
5. ✅ Комментарий с результатами появляется в PR

#### 3.2 Проверка комментария

```bash
# Посмотреть комментарии PR
gh pr view $PR_NUMBER --comments
```

Ожидаемый комментарий:
```
## E2E Tests Results ✅

**Module:** sds-node-configurator
**Environment:** local
**Test Type:** smoke

Tests passed!
```

### 4. Проверка разных сценариев

#### 4.1 Smoke тесты (local)

```bash
gh pr edit $PR_NUMBER --add-label "e2e:run" --add-label "e2e:smoke"
```

**Ожидаемое время:** 5-10 минут

#### 4.2 Полные тесты (hypervisor)

```bash
gh pr edit $PR_NUMBER --add-label "e2e:run" --add-label "e2e:full" --add-label "e2e:hypervisor"
```

**Ожидаемое время:** 30-60 минут

#### 4.3 Пропуск тестов

```bash
gh pr edit $PR_NUMBER --add-label "e2e:skip"
```

**Ожидаемое поведение:** Комментарий о пропуске тестов

### 5. Критерии успеха PoC

- ✅ Лейблы создаются без ошибок
- ✅ Секреты настроены правильно
- ✅ CI не ломается при добавлении новых job'ов
- ✅ Парсинг лейблов работает корректно
- ✅ E2E тесты запускаются при наличии лейбла `e2e:run`
- ✅ E2E тесты не запускаются без лейбла
- ✅ Комментарии в PR работают
- ✅ Smoke тесты выполняются < 10 минут
- ✅ Полные тесты выполняются на self-hosted runner

## 🐛 Troubleshooting

### Проблема: Тесты не запускаются

**Решение:**
```bash
# Проверить наличие лейблов
gh pr view $PR_NUMBER --json labels --jq '.labels[].name'

# Проверить логи парсинга
gh run view --log | grep "Parse PR labels"
```

### Проблема: Ошибка доступа к sds-e2e

**Решение:**
```bash
# Проверить токен
gh secret list | grep E2E_TRIGGER_TOKEN

# Пересоздать токен с правильными правами
gh auth token | gh secret set E2E_TRIGGER_TOKEN
```

### Проблема: Ошибка доступа к кластеру

**Решение:**
```bash
# Проверить kubeconfig
echo "$KUBECONFIG_HYPERVISOR" | base64 -d | kubectl --kubeconfig=- get nodes

# Проверить SSH ключ
ssh -i "$SSH_PRIVATE_KEY" user@hostname "echo ok"
```

## 📊 Результаты PoC

После успешного выполнения PoC:

1. **Документировать:**
   - Время выполнения smoke тестов
   - Время выполнения полных тестов
   - Проблемы и их решения

2. **Обновить ADR** с реальными данными

3. **Создать план развертывания** на другие модули:
   - sds-replicated-volume
   - data-export
   - другие модули

## 🚀 Следующие шаги

1. Развернуть на все модули
2. Добавить больше лейблов (приоритет, конкретные тесты)
3. Интегрировать с мониторингом
4. Реализовать TODO из ADR-002
