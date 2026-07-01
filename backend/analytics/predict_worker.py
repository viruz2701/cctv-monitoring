#!/usr/bin/env python3
"""
P0-CR-04: NATS JetStream Worker для распределённых предсказаний отказа устройств.
==============================================================================

Заменяет subprocess + stdout/stderr pipes на NATS JetStream WorkQueue.

Архитектура:
  Go PredictionService → публикует PredictionTask в JetStream ml.predict
  Python predict_worker.py → потребляет задачи, предсказывает, сохраняет в БД

Критические изменения:
  - НЕТ subprocess: worker импортирует predict.py напрямую
  - НЕТ pipe deadlock: Go больше не читает stdout/stderr Python
  - Backpressure: MaxAckPending лимитирует конкурентные задачи
  - Graceful shutdown: SIGTERM → завершить текущую задачу → остановить consumer
  - Per-device: каждая задача обрабатывает одно устройство (нет OOM)

Usage:
  python3 predict_worker.py                          # с config.yaml
  python3 predict_worker.py --nats nats://host:4222  # переопределить NATS URL

Compliance:
  - IEC 62443-3-3 SR 3.1 (Queue-based processing)
  - ISO 27001 A.12.4.1 (Event logging)
  - Приказ ОАЦ №66 п. 7.18 (Управление удалённым доступом)
"""

import argparse
import json
import os
import signal
import sys
import time
import warnings
from datetime import datetime, timezone
from typing import Optional

import yaml

warnings.filterwarnings("ignore")

# Добавляем путь для импорта predict.py
BASE_DIR = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, BASE_DIR)

# ── NATS imports (with graceful fallback for docs) ──
try:
    import nats
    from nats.js import api as js_api
    NATS_AVAILABLE = True
except ImportError:
    NATS_AVAILABLE = False


# ── Constants ──────────────────────────────────────────────────────────────

# DEFAULT_NATS_URL — URL NATS сервера по умолчанию.
DEFAULT_NATS_URL = "nats://localhost:4222"

# PREDICTION_STREAM — имя JetStream стрима (должен совпадать с Go).
PREDICTION_STREAM = "ML_PREDICT"

# PREDICTION_SUBJECT — subject для задач.
PREDICTION_SUBJECT = "ml.predict"

# PREDICTION_CONSUMER — имя durable consumer.
PREDICTION_CONSUMER = "predict-worker"

# MAX_DELIVER — максимальное количество попыток доставки.
MAX_DELIVER = 3

# DEFAULT_MAX_ACTIVE — лимит конкурентных задач (backpressure).
DEFAULT_MAX_ACTIVE = 5

# ACK_WAIT — время на обработку одной задачи (сек).
ACK_WAIT = 300  # 5 минут

# RECONNECT_WAIT — пауза перед переподключением.
RECONNECT_WAIT = 2


# ── Signal Handling ────────────────────────────────────────────────────────

_shutdown_requested = False


def _handle_signal(signum, frame):
    """Обработчик сигналов для graceful shutdown."""
    global _shutdown_requested
    if _shutdown_requested:
        print(f"[worker] Forced exit (SIGTERM repeated)", file=sys.stderr)
        sys.exit(1)
    _shutdown_requested = True
    signame = signal.Signals(signum).name
    print(f"[worker] {signame} received — finishing current task, "
          f"stopping consumer...", file=sys.stderr)


def is_shutdown_requested() -> bool:
    """Проверяет, был ли запрошен graceful shutdown."""
    return _shutdown_requested


# ── Model Cache ────────────────────────────────────────────────────────────

_model_cache = {}
_metadata_cache = {}


def get_model(cfg: dict, variant: str):
    """
    Загружает модель XGBoost с кэшированием.
    Модель загружается один раз и кэшируется по variant'у.
    """
    global _model_cache, _metadata_cache

    cache_key = variant or cfg["ml"].get("model_variant", "A")
    if cache_key in _model_cache:
        return _model_cache[cache_key], _metadata_cache[cache_key]

    # Определяем пути к модели
    import joblib as _joblib

    base_path = os.path.join(BASE_DIR, cfg["ml"]["model_path"])
    meta_path = os.path.join(BASE_DIR, cfg["ml"]["model_metadata_path"])

    if variant and variant != cfg["ml"].get("model_variant", "A"):
        variant_path = base_path.replace(".pkl", f"_{variant}.pkl")
        variant_meta = meta_path.replace(".json", f"_{variant}.json")
        if os.path.exists(variant_path):
            base_path = variant_path
            meta_path = variant_meta

    if not os.path.exists(base_path):
        raise FileNotFoundError(f"Model not found: {base_path}")

    model = _joblib.load(base_path)
    metadata = {}
    if os.path.exists(meta_path):
        with open(meta_path) as f:
            metadata = json.load(f)

    _model_cache[cache_key] = model
    _metadata_cache[cache_key] = metadata

    model_version = metadata.get("version", "xgboost_v1")
    print(f"[worker] Loaded model: {model_version} (variant={cache_key})",
          file=sys.stderr)

    return model, metadata


# ── Prediction Task Handler ────────────────────────────────────────────────


def process_task(
    cfg: dict,
    device_id: str,
    variant: str,
    trace_id: str,
    nc: Optional["nats.NATS"],
) -> dict:
    """
    Обрабатывает одну задачу предсказания для устройства.
    Возвращает результат предсказания в виде словаря.

    Args:
        cfg: Конфигурация из config.yaml
        device_id: ID устройства
        variant: Вариант модели (A/B)
        trace_id: Trace ID для трейсинга
        nc: NATS соединение (для публикации результата)
    """
    import psycopg2
    from predict import (
        load_features,
        predict as _predict,
        save_predictions,
    )

    print(f"[worker] Processing device={device_id} variant={variant} "
          f"trace={trace_id[:16]}...", file=sys.stderr)

    # ── 1. Загружаем модель (из кэша) ──
    model, metadata = get_model(cfg, variant)
    model_version = metadata.get("version", "xgboost_v1")

    # ── 2. Подключаемся к БД ──
    conn = psycopg2.connect(**cfg["db"])

    try:
        # ── 3. Загружаем признаки для устройства ──
        df = load_features(conn, cfg, device_id=device_id)
        if df.empty:
            print(f"[worker] No features for device {device_id}, skipping",
                  file=sys.stderr)
            return {"device_id": device_id, "status": "skipped", "reason": "no_features"}

        # ── 4. Предсказываем ──
        results = _predict(
            cfg, conn, model, metadata, df,
            trace_id=trace_id, variant=variant,
        )

        if not results:
            print(f"[worker] No predictions for device {device_id}",
                  file=sys.stderr)
            return {"device_id": device_id, "status": "no_predictions"}

        # ── 5. Сохраняем в БД ──
        save_predictions(conn, results)

        # ── 6. Публикуем результат в NATS ──
        if nc is not None and nc.is_connected:
            for r in results:
                _publish_result(nc, r, cfg)

        result_summary = results[0]
        print(f"[worker] Device {device_id}: "
              f"prob={result_summary.get('failure_probability', 'N/A'):.4f}, "
              f"actionable={result_summary.get('is_actionable', False)}",
              file=sys.stderr)

        return {"device_id": device_id, "status": "ok",
                "predictions": len(results)}

    except Exception as e:
        print(f"[worker] Error processing device {device_id}: {e}",
              file=sys.stderr)
        raise
    finally:
        conn.close()


def _publish_result(nc, result: dict, cfg: dict):
    """
    Публикует результат предсказания в NATS топик ml.prediction.{device_id}.
    """
    device_id = result.get("device_id", "unknown")
    subject = f"ml.prediction.{device_id}"

    # Добавляем мета-поля для трейсинга
    payload = dict(result)
    payload["_source"] = "predict_worker"
    payload["_processed_at"] = datetime.now(timezone.utc).isoformat()

    try:
        data = json.dumps(payload, ensure_ascii=False, default=str).encode()
        nc.publish(subject, data)
        print(f"[worker] Published result to {subject}", file=sys.stderr)
    except Exception as e:
        print(f"[worker] Failed to publish result to {subject}: {e}",
              file=sys.stderr)


# ── NATS Consumer ──────────────────────────────────────────────────────────


async def run_worker(cfg: dict, nats_url: str = None):
    """
    Запускает NATS JetStream consumer для обработки задач предсказания.

    Args:
        cfg: Конфигурация из config.yaml
        nats_url: URL NATS сервера (переопределяет config)
    """
    global NATS_AVAILABLE

    if not NATS_AVAILABLE:
        print("[worker] FATAL: nats-py not installed. "
              "Run: pip install nats-py", file=sys.stderr)
        sys.exit(1)

    url = nats_url or cfg.get("nats", {}).get("url", DEFAULT_NATS_URL)
    max_active = cfg.get("ml", {}).get("max_active_workers", DEFAULT_MAX_ACTIVE)

    print(f"[worker] Connecting to NATS: {url}", file=sys.stderr)
    print(f"[worker] Max active tasks: {max_active}", file=sys.stderr)

    # ── 1. Подключаемся к NATS ──
    nc = await nats.connect(
        url,
        max_reconnect_attempts=-1,
        reconnect_time_wait=RECONNECT_WAIT,
        error_cb=lambda e: print(f"[worker] NATS error: {e}", file=sys.stderr),
        disconnected_cb=lambda: print(f"[worker] NATS disconnected",
                                      file=sys.stderr),
        reconnected_cb=lambda: print(f"[worker] NATS reconnected "
                                     f"{nc.connected_url if hasattr(nc, 'connected_url') else ''}",
                                     file=sys.stderr),
        closed_cb=lambda: print(f"[worker] NATS connection closed",
                                file=sys.stderr),
    )

    # ── 2. Создаём JetStream контекст ──
    js = nc.jetstream()

    # ── 3. Создаём или обновляем стрим ──
    try:
        await js.add_stream(
            name=PREDICTION_STREAM,
            subjects=[PREDICTION_SUBJECT],
            storage="file",
            retention="workqueue",
            max_age=24 * 3600,  # 24 часа в секундах
        )
        print(f"[worker] Stream {PREDICTION_STREAM} ready", file=sys.stderr)
    except Exception as e:
        if "already exists" in str(e).lower():
            print(f"[worker] Stream {PREDICTION_STREAM} already exists",
                  file=sys.stderr)
        else:
            print(f"[worker] Stream creation warning: {e}", file=sys.stderr)

    # ── 4. Создаём pull consumer с backpressure ──
    try:
        await js.add_consumer(
            stream=PREDICTION_STREAM,
            config=js_api.ConsumerConfig(
                durable_name=PREDICTION_CONSUMER,
                deliver_policy=js_api.DeliverAllPolicy,
                ack_policy=js_api.AckExplicitPolicy,
                max_deliver=MAX_DELIVER,
                max_ack_pending=max_active,
                ack_wait=ACK_WAIT,
            ),
        )
        print(f"[worker] Consumer {PREDICTION_CONSUMER} ready "
              f"(max_ack_pending={max_active})", file=sys.stderr)
    except Exception as e:
        if "already exists" in str(e).lower():
            print(f"[worker] Consumer {PREDICTION_CONSUMER} already exists",
                  file=sys.stderr)
        else:
            print(f"[worker] Consumer creation error: {e}", file=sys.stderr)
            await nc.close()
            sys.exit(1)

    # ── 5. Основной цикл обработки ──
    print(f"[worker] ═══ Worker started, waiting for tasks... ═══",
          file=sys.stderr)

    sub = await js.pull_subscribe(
        subject=PREDICTION_SUBJECT,
        stream=PREDICTION_STREAM,
        durable=PREDICTION_CONSUMER,
    )

    # Предзагружаем модель (variant A) при старте
    try:
        get_model(cfg, cfg["ml"].get("model_variant", "A"))
        print(f"[worker] Model pre-loaded", file=sys.stderr)
    except Exception as e:
        print(f"[worker] Model pre-load warning: {e}", file=sys.stderr)

    try:
        while not is_shutdown_requested():
            try:
                # Fetch одной задачи (pull-based)
                msgs = await sub.fetch(1, timeout=1.0)

                for msg in msgs:
                    if is_shutdown_requested():
                        # Не nak — оставляем в очереди для другого worker'а
                        break

                    try:
                        # Парсим задачу
                        task_data = json.loads(msg.data.decode())
                        device_id = task_data.get("device_id", "")
                        variant = task_data.get("model_variant", "A")
                        trace_id = task_data.get("trace_id", "")

                        if not device_id:
                            print(f"[worker] Invalid task: missing device_id",
                                  file=sys.stderr)
                            await msg.term()  # Terminate — не ретраить
                            continue

                        # Обрабатываем
                        await msg.in_progress()  # Продлеваем ack_wait
                        result = process_task(cfg, device_id, variant,
                                              trace_id, nc)

                        if result.get("status") == "ok":
                            await msg.ack()
                        else:
                            # Не fatal — но ack (не nak), т.к. повтор не поможет
                            await msg.ack()

                    except json.JSONDecodeError as e:
                        print(f"[worker] Invalid JSON in task: {e}",
                              file=sys.stderr)
                        await msg.term()
                    except Exception as e:
                        print(f"[worker] Task failed: {e}", file=sys.stderr)
                        await msg.nak()

            except nats.errors.TimeoutError:
                # Нет задач в очереди — нормально, продолжаем ждать
                continue
            except Exception as e:
                if not is_shutdown_requested():
                    print(f"[worker] Fetch error: {e}", file=sys.stderr)
                    time.sleep(1)
                continue

    except KeyboardInterrupt:
        print(f"[worker] KeyboardInterrupt", file=sys.stderr)
    finally:
        print(f"[worker] Draining active tasks...", file=sys.stderr)
        await sub.unsubscribe()
        await nc.drain()
        print(f"[worker] ═══ Worker stopped ═══", file=sys.stderr)


# ── CLI ────────────────────────────────────────────────────────────────────


def parse_args():
    parser = argparse.ArgumentParser(
        description="P0-CR-04: NATS JetStream Worker for Device Failure Prediction",
    )
    parser.add_argument("--nats", type=str, default=None,
                        help=f"NATS URL (default: from config or {DEFAULT_NATS_URL})")
    parser.add_argument("--config", type=str, default=None,
                        help="Path to config.yaml (default: analytics/config.yaml)")
    return parser.parse_args()


def load_config(config_path: str = None) -> dict:
    """Загружает конфигурацию."""
    if config_path is None:
        config_path = os.path.join(BASE_DIR, "config.yaml")
    if not os.path.exists(config_path):
        print(f"[worker] Config not found: {config_path}", file=sys.stderr)
        print(f"[worker] Using minimal defaults", file=sys.stderr)
        return {
            "db": {
                "host": os.environ.get("GB_DB_HOST", "localhost"),
                "port": int(os.environ.get("GB_DB_PORT", "5432")),
                "dbname": os.environ.get("GB_DB_NAME", "cctv"),
                "user": os.environ.get("GB_DB_USER", "cctv"),
                "password": os.environ.get("GB_DB_PASSWORD", "cctv"),
            },
            "ml": {
                "model_path": "models/xgboost_model.pkl",
                "model_metadata_path": "models/xgboost_metadata.json",
                "model_variant": "A",
                "prediction_window_days": 30,
                "feature_columns": [],
                "max_active_workers": DEFAULT_MAX_ACTIVE,
            },
            "service": {
                "probability_threshold": 0.5,
                "min_confidence_threshold": 0.3,
            },
            "nats": {
                "url": DEFAULT_NATS_URL,
            },
        }
    with open(config_path) as f:
        return yaml.safe_load(f)


def main():
    args = parse_args()
    cfg = load_config(args.config)

    # Устанавливаем обработчики сигналов
    signal.signal(signal.SIGTERM, _handle_signal)
    signal.signal(signal.SIGINT, _handle_signal)

    print(f"[worker] P0-CR-04: NATS JetStream Worker starting...",
          file=sys.stderr)
    print(f"[worker] Config: {args.config or os.path.join(BASE_DIR, 'config.yaml')}",
          file=sys.stderr)

    import asyncio
    try:
        asyncio.run(run_worker(cfg, nats_url=args.nats))
    except KeyboardInterrupt:
        print(f"[worker] Exiting...", file=sys.stderr)


if __name__ == "__main__":
    main()
