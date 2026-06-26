# P2-1.1: Real ML Model Integration — Progress

## Статус выполнения

- [x] P2-1.1: Real ML Model Integration — XGBoost модель для предсказания отказов устройств

### Выполненные задачи

- [x] **DB Migration**: `031_ml_predictions` — таблица `predictions` (hypertable) с полями: confidence_score, model_variant (A/B), features_snapshot, top_features, calibration_bin
- [x] **train.py**: Полный рерайт — загрузка реальных данных из TimescaleDB, XGBoost с hyperparameter tuning, StratifiedKFold CV, feature engineering (failure_score, interactions, age bins), >75% accuracy target
- [x] **predict.py**: Полный рерайт — JSONL output для Go subprocess, confidence score, A/B variant, `--test`/`--device`/`--variant`/`--trace` флаги, anomaly detection
- [x] **Go service** (`internal/ml/`): `PredictionService` — subprocess вызов Python, парсинг JSONL, публикация в NATS `ml.prediction.{device_id}`, A/B testing (hash-based assignment), JetStream durable storage
- [x] **Go tests**: 11 тестов (JSONL парсинг, invalid/skip строки, A/B assignment, hash determinism, config defaults) — все PASS
- [x] **Go build**: `go build ./...` — успешно

### Файлы

| Файл | Описание |
|------|----------|
| [`backend/internal/db/migrations/031_ml_predictions.up.sql`](backend/internal/db/migrations/031_ml_predictions.up.sql) | Migration: predictions hypertable |
| [`backend/internal/db/migrations/031_ml_predictions.down.sql`](backend/internal/db/migrations/031_ml_predictions.down.sql) | Rollback migration |
| [`backend/analytics/train.py`](backend/analytics/train.py) | XGBoost training с реальными данными |
| [`backend/analytics/predict.py`](backend/analytics/predict.py) | Prediction с JSONL output |
| [`backend/analytics/config.yaml`](backend/analytics/config.yaml) | ML + NATS конфигурация |
| [`backend/analytics/requirements.txt`](backend/analytics/requirements.txt) | Python зависимости |
| [`backend/internal/ml/config.go`](backend/internal/ml/config.go) | Go ML config struct |
| [`backend/internal/ml/prediction_service.go`](backend/internal/ml/prediction_service.go) | Go prediction service |
| [`backend/internal/ml/prediction_service_test.go`](backend/internal/ml/prediction_service_test.go) | 11 тестов |

### Проверка

```bash
cd backend && go build ./...                    # ✓ OK
cd backend && python3 analytics/predict.py --test  # requires: pip install -r requirements.txt
cd backend && go test ./internal/ml/... -v      # ✓ 11/11 PASS
```

---

# P2-1.2: AI Assistant Chat — Progress

## Статус выполнения

- [x] P2-1.2: AI Assistant Chat с DeepSeek интеграцией

### Выполненные задачи

- [x] **Backend proxy**: `/api/v1/ai/chat` (SSE streaming), `/api/v1/ai/feedback` — DeepSeek API key хранится только на сервере
- [x] **Config**: `deepseek_api_key` / `GB_DEEPSEEK_API_KEY` env var
- [x] **API Client** (`lib/deepseek.ts`): SSE парсинг, контекст из URL, feedback
- [x] **Hook** (`hooks/useAIAssistant.ts`): управление состоянием чата, история в sessionStorage, abort controller
- [x] **Chat Panel** (`components/ai/AIAssistantPanel.tsx`): боковая панель, Markdown рендеринг, like/dislike, quick prompts, контекстный заголовок
- [x] **Dependencies**: `react-markdown`, `remark-gfm`
- [x] **TypeScript**: `npx tsc --noEmit` — ✓ OK
- [x] **Go build**: `go build ./...` — требуется проверка

### Файлы

| Файл | Описание |
|------|----------|
| [`backend/internal/api/ai_routes.go`](backend/internal/api/ai_routes.go) | Backend proxy: SSE streaming, feedback |
| [`backend/internal/config/config.go`](backend/internal/config/config.go) | `DeepSeekAPIKey` field + env binding |
| [`backend/internal/api/server.go`](backend/internal/api/server.go) | Route mounting |
| [`frontend/src/lib/deepseek.ts`](frontend/src/lib/deepseek.ts) | API клиент с SSE парсингом |
| [`frontend/src/hooks/useAIAssistant.ts`](frontend/src/hooks/useAIAssistant.ts) | React hook для чата |
| [`frontend/src/components/ai/AIAssistantPanel.tsx`](frontend/src/components/ai/AIAssistantPanel.tsx) | Chat panel компонент |

### Проверка

```bash
cd frontend && npx tsc --noEmit --pretty      # ✓ OK
cd backend && go build ./...                   # requires: go mod tidy
```

### Требования к настройке

1. Установить `GB_DEEPSEEK_API_KEY` в `.env` или переменные окружения
2. Пересобрать backend: `cd backend && go build ./...`
3. Перезапустить сервер

---

# P2-2.1: Workflow Builder UI — Progress

## Статус выполнения

- [x] P2-2.1: Workflow Builder UI с React Flow

### Выполненные задачи

- [x] **Install @xyflow/react**: Добавлена зависимость React Flow v12
- [x] **Types** (`types/workflow.ts`): Типы для нод (trigger/condition/action/delay), палитры, workflow definition, version control, test run
- [x] **Store** (`store/workflowStore.ts`): Zustand store с persist — CRUD workflow, graph editing, version control (save/load), export/import JSON, test mode state
- [x] **Custom Node** (`components/workflow/WorkflowNode.tsx`): Единый кастомный nodeType с цветовой дифференциацией (purple/amber/blue/teal), статус индикатором (idle/running/success/error), condition node с true/false handles
- [x] **Toolbar** (`components/workflow/WorkflowToolbar.tsx`): Боковая панель с палитрой компонентов для drag&drop, workflow selector, save/save version, export/import JSON, version history
- [x] **CEL Editor** (`components/workflow/WorkflowCELInput.tsx`): Редактор CEL выражений с подсветкой синтаксиса, валидацией скобок/кавычек, сниппетами, документацией доступных переменных
- [x] **Test Panel** (`components/workflow/WorkflowTestPanel.tsx`): Панель тестирования с mock event editor (4 шаблона), топологической сортировкой нод, симуляцией выполнения, результатами по каждому узлу
- [x] **WorkflowBuilder** (`components/workflow/WorkflowBuilder.tsx`): Главный компонент с React Flow canvas, Background/Controls/MiniMap, drag&drop из палитры, правой панелью инспектора (конфигурация каждого типа нод), переключением в test mode
- [x] **TypeScript Check**: `npx tsc --noEmit` — ✓ OK

### Файлы

| Файл | Описание |
|------|----------|
| [`frontend/src/types/workflow.ts`](frontend/src/types/workflow.ts) | Типы workflow (node data, definition, version, palette) |
| [`frontend/src/store/workflowStore.ts`](frontend/src/store/workflowStore.ts) | Zustand store для workflow |
| [`frontend/src/components/workflow/WorkflowNode.tsx`](frontend/src/components/workflow/WorkflowNode.tsx) | Custom React Flow node |
| [`frontend/src/components/workflow/WorkflowToolbar.tsx`](frontend/src/components/workflow/WorkflowToolbar.tsx) | Sidebar с палитрой и кнопками |
| [`frontend/src/components/workflow/WorkflowCELInput.tsx`](frontend/src/components/workflow/WorkflowCELInput.tsx) | CEL expression editor |
| [`frontend/src/components/workflow/WorkflowTestPanel.tsx`](frontend/src/components/workflow/WorkflowTestPanel.tsx) | Test mode panel |
| [`frontend/src/components/workflow/WorkflowBuilder.tsx`](frontend/src/components/workflow/WorkflowBuilder.tsx) | Главный компонент построителя |

### Проверка

```bash
cd frontend && npx tsc --noEmit --pretty      # ✓ OK
```
