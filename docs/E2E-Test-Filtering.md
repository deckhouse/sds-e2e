# E2E Test Filtering: Как фильтровать тесты

## 🎯 Проблема

При запуске отдельных тестовых файлов в Go:
```bash
go test ./tests/00_healthcheck_test.go ./tests/01_sds_nc_test.go
```

Возникают ошибки типа:
```
Error: tests/01_sds_nc_test.go:33:3: undefined: removeTestDisks
Error: tests/01_sds_nc_test.go:39:2: undefined: prepareClr
Error: tests/01_sds_nc_test.go:70:58: undefined: HvStorageClass
```

**Причина:** Go не видит вспомогательные функции и переменные из других файлов пакета (`tools.go`, `base_test.go` и т.д.)

## ✅ Решение

Вместо запуска отдельных файлов используем флаг `-run` для фильтрации тестов:

```bash
# НЕПРАВИЛЬНО (файлы по отдельности)
go test ./tests/00_healthcheck_test.go ./tests/01_sds_nc_test.go

# ПРАВИЛЬНО (используем полный путь модуля + фильтр)
go test -run '^(TestNodeHealthCheck|TestLvg)$' github.com/deckhouse/sds-e2e/tests
```

**Важно:** Нужно использовать полный путь модуля (`github.com/deckhouse/sds-e2e/tests`), а не относительный путь (`./tests/`), чтобы Go правильно разрешал импорты из `../util/`.

## 📋 Примеры фильтрации

### Smoke тесты для sds-node-configurator

```bash
go test -v -timeout 30m \
  -run '^(TestNodeHealthCheck|TestLvg)$' \
  github.com/deckhouse/sds-e2e/tests \
  -stand metal \
  -hypervisorkconfig /path/to/kubeconfig \
  -verbose -debug
```

**Запустятся тесты:**
- `TestNodeHealthCheck` из `00_healthcheck_test.go`
- `TestLvg` из `01_sds_nc_test.go`

### Smoke тесты для sds-replicated-volume

```bash
go test -v -timeout 30m \
  -run '^(TestNodeHealthCheck|TestPVC)$' \
  github.com/deckhouse/sds-e2e/tests \
  -stand metal \
  -hypervisorkconfig /path/to/kubeconfig \
  -verbose -debug
```

**Запустятся тесты:**
- `TestNodeHealthCheck` из `00_healthcheck_test.go`
- `TestPVC` из `03_sds_lv_test.go`

### Smoke тесты для data-export

```bash
go test -v -timeout 30m \
  -run '^(TestNodeHealthCheck|TestDataExport)$' \
  github.com/deckhouse/sds-e2e/tests \
  -stand metal \
  -hypervisorkconfig /path/to/kubeconfig \
  -verbose -debug
```

**Запустятся тесты:**
- `TestNodeHealthCheck` из `00_healthcheck_test.go`
- `TestDataExport` из `base_test.go`

### Только healthcheck (для неизвестных модулей)

```bash
go test -v -timeout 30m \
  -run '^TestNodeHealthCheck$' \
  github.com/deckhouse/sds-e2e/tests \
  -stand metal \
  -hypervisorkconfig /path/to/kubeconfig \
  -verbose -debug
```

## 🔍 Синтаксис -run

Флаг `-run` использует регулярные выражения Go:

```bash
# Точное совпадение
-run '^TestNodeHealthCheck$'

# Несколько тестов (OR)
-run '^(TestNodeHealthCheck|TestLvg)$'

# Все тесты начинающиеся с "TestLvg"
-run '^TestLvg'

# Конкретные под-тесты
-run 'TestDataExport/(routing|auth)'

# Все тесты кроме определенных (через grep)
go test ./tests/ | grep -v "TestSomething"
```

## 📊 Структура тестов в testkit_v2

```
testkit_v2/tests/
├── 00_healthcheck_test.go      # TestNodeHealthCheck
├── 01_sds_nc_test.go           # TestLvg
├── 03_sds_lv_test.go           # TestPVC
├── 05_sds_node_configurator_test.go  # TestLvgThick*, TestLvgThin*
├── base_test.go                # TestDataExport
├── tools.go                    # Вспомогательные функции (prepareClr, removeTestDisks, etc.)
└── 99_finalizer_test.go        # TestFinalizer
```

**Важно:** Все файлы в директории `tests/` принадлежат одному пакету, и тесты импортируют `../util/`, поэтому:
- ✅ `go test github.com/deckhouse/sds-e2e/tests` - видит все файлы пакета и правильно разрешает импорты
- ❌ `go test ./tests/` - ошибки импорта `../util/`
- ❌ `go test ./tests/01_sds_nc_test.go` - видит только один файл

## 🎨 Применение в CI

### В Kubernetes Job манифесте

```yaml
command:
- /bin/bash
- -c
- |
  cd /workspace/sds-e2e/testkit_v2
  
  # Определяем фильтр в зависимости от модуля
  if [[ "$MODULE_NAME" == *"sds-node-configurator"* ]]; then
    TEST_RUN="-run '^(TestNodeHealthCheck|TestLvg)$'"
  elif [[ "$MODULE_NAME" == *"sds-replicated-volume"* ]]; then
    TEST_RUN="-run '^(TestNodeHealthCheck|TestPVC)$'"
  elif [[ "$MODULE_NAME" == *"data-export"* ]]; then
    TEST_RUN="-run '^(TestNodeHealthCheck|TestDataExport)$'"
  else
    TEST_RUN="-run '^TestNodeHealthCheck$'"
  fi
  
  # Запускаем с фильтром (используем полный путь модуля)
  go test -v -timeout 30m ${TEST_RUN} github.com/deckhouse/sds-e2e/tests \
    -stand metal \
    -hypervisorkconfig /etc/e2e/kubeconfigs/hypervisor \
    -verbose -debug
```

### В GitHub Actions

```yaml
- name: Create E2E test Job
  run: |
    # Определяем фильтр
    MODULE_NAME="${{ vars.MODULE_NAME }}"
    if [[ "$MODULE_NAME" == *"sds-node-configurator"* ]]; then
      TEST_RUN="-run '^(TestNodeHealthCheck|TestLvg)$'"
    fi
    
    # Создаем Job с фильтром
    kubectl apply -f - <<EOF
    # ... (Job manifest with TEST_RUN)
    EOF
```

## 🐛 Troubleshooting

### Ошибка: undefined: functionName или undefined: util.Something

**Проблема 1:** Запускаете отдельные файлы вместо пакета  
**Проблема 2:** Используете `./tests/` вместо полного пути модуля

**Решение:**
```bash
# НЕПРАВИЛЬНО (отдельные файлы)
go test ./tests/01_sds_nc_test.go

# НЕПРАВИЛЬНО (относительный путь - ошибки импорта util)
go test -run '^TestLvg$' ./tests/

# ПРАВИЛЬНО (полный путь модуля)
go test -run '^TestLvg$' github.com/deckhouse/sds-e2e/tests
```

### Тесты не найдены

**Проблема:** Неверное регулярное выражение в `-run`

**Решение:**
```bash
# Проверить какие тесты доступны
go test -list='.*' ./tests/

# Использовать точный синтаксис
-run '^TestName$'  # Точное совпадение
-run '^(Test1|Test2)$'  # Несколько тестов
```

### Запускаются лишние тесты

**Проблема:** Слишком широкое регулярное выражение

**Решение:**
```bash
# НЕПРАВИЛЬНО (запустит TestLvg, TestLvgThick, TestLvgThin, и т.д.)
-run 'TestLvg'

# ПРАВИЛЬНО (только TestLvg)
-run '^TestLvg$'
```

## 📚 Дополнительные ресурсы

- [Go test documentation](https://pkg.go.dev/cmd/go#hdr-Test_packages)
- [Go regexp syntax](https://pkg.go.dev/regexp/syntax)
- [testkit_v2 README](../testkit_v2/README.md)

## 💡 Best Practices

1. **Всегда указывайте директорию пакета:**
   ```bash
   go test ./tests/  # ✅
   ```

2. **Используйте ^ и $ для точного совпадения:**
   ```bash
   -run '^TestName$'  # ✅ только TestName
   -run 'TestName'    # ❌ может захватить TestName2, TestNameFoo, etc.
   ```

3. **Группируйте связанные тесты:**
   ```bash
   -run '^(TestHealthCheck|TestBasic)$'  # Smoke тесты
   -run '^TestLvg(Thick|Thin)'          # Только LVG тесты
   ```

4. **Комбинируйте с другими флагами:**
   ```bash
   go test -v -run '^TestSmoke' -timeout 10m -count=1 ./tests/
   ```

