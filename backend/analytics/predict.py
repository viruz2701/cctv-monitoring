#!/usr/bin/env python3
"""
P2-1.1: XGBoost Prediction Script — Device Failure Prediction
==============================================================

Загружает обученную XGBoost модель, извлекает признаки из TimescaleDB,
генерирует предсказания для всех устройств и выводит JSON в stdout.

Режимы:
  - Обычный: загружает модель, предсказывает, сохраняет в БД, выводит JSON
  - --test: проверяет модель на тестовых данных (без сохранения в БД)
  - --device DEVICE_ID: предсказание для конкретного устройства

Формат JSON (каждое предсказание — одна строка):
  {
    "device_id": "CAM-001",
    "failure_probability": 0.87,
    "confidence_score": 0.92,
    "model_version": "xgboost_v1",
    "model_variant": "A",
    "prediction_date": "2026-06-26T14:00:00+00:00",
    "prediction_window_days": 30,
    "is_actionable": true,
    "is_anomaly": false,
    "calibration_bin": 8,
    "top_features": [
      {"feature": "offline_ratio", "importance": 0.45, "value": 0.32},
      {"feature": "error_count", "importance": 0.28, "value": 15},
      {"feature": "reboot_count", "importance": 0.12, "value": 6}
    ],
    "features_snapshot": {
      "offline_ratio": 0.32,
      "error_count": 15,
      ...
    },
    "trace_id": "a1b2c3d4e5f6a7b8"
  }

Usage:
  python3 predict.py                           # batch prediction
  python3 predict.py --test                    # test mode
  python3 predict.py --device CAM-001          # single device
  python3 predict.py --variant B               # use variant B model
  python3 predict.py --trace TRACE_ID          # set trace_id
"""

import argparse
import json
import os
import random
import sys
import warnings
from datetime import datetime, timezone

import joblib
import numpy as np
import pandas as pd
import psycopg2
import yaml

warnings.filterwarnings("ignore")

BASE_DIR = os.path.dirname(os.path.abspath(__file__))
CONFIG_PATH = os.path.join(BASE_DIR, "config.yaml")
MODEL_DIR = os.path.join(BASE_DIR, "models")
RANDOM_STATE = 42

# ── Utilities ─────────────────────────────────────────────────────────


def load_config() -> dict:
    with open(CONFIG_PATH) as f:
        return yaml.safe_load(f)


def get_model_paths(cfg: dict, variant: str = None) -> tuple:
    """
    Возвращает (model_path, metadata_path) для указанного variant'а.
    Для A/B тестирования: model variant B может быть отдельной моделью.
    """
    base_path = os.path.join(BASE_DIR, cfg["ml"]["model_path"])
    meta_path = os.path.join(BASE_DIR, cfg["ml"]["model_metadata_path"])

    if variant and variant != cfg["ml"].get("model_variant", "A"):
        # Пробуем загрузить variant-specific модель
        variant_path = base_path.replace(".pkl", f"_{variant}.pkl")
        variant_meta = meta_path.replace(".json", f"_{variant}.json")
        if os.path.exists(variant_path):
            return variant_path, variant_meta

    return base_path, meta_path


def generate_trace_id() -> str:
    """Генерирует trace_id (16 байт hex)."""
    import hashlib
    raw = os.urandom(16)
    return hashlib.sha256(raw).hexdigest()[:32]


# ── Feature Loading ───────────────────────────────────────────────────


def load_features(conn, cfg: dict, device_id: str = None, days: int = None) -> pd.DataFrame:
    """
    Загружает признаки из TimescaleDB через ETL.
    Если device_id указан — только для него, иначе для всех.
    """
    sys.path.insert(0, BASE_DIR)
    from etl import extract_features  # noqa

    if days is None:
        days = cfg["ml"]["prediction_window_days"]

    return extract_features(conn, device_id=device_id, days=days)


def prepare_features(df: pd.DataFrame, cfg: dict) -> pd.DataFrame:
    """
    Приводит признаки к формату, ожидаемому моделью.
    Добавляет engineered features (те же, что и в train.py).
    """
    features = cfg["ml"]["feature_columns"]

    # Приводим все к float
    for col in features:
        if col in df.columns:
            df[col] = pd.to_numeric(df[col], errors="coerce").fillna(0).astype(float)
        else:
            df[col] = 0.0

    # ── Те же engineered features, что и при обучении ──
    if all(f in df.columns for f in ["offline_ratio", "error_count", "age_days"]):
        df["failure_score"] = (
            df["offline_ratio"] * 3.0
            + np.log1p(df["error_count"]) * 0.5
            + np.log1p(df["age_days"]) * 0.01
        )

    if "offline_ratio" in df.columns and "error_count" in df.columns:
        df["offline_error_interaction"] = (
            df["offline_ratio"] * np.log1p(df["error_count"])
        )

    if "age_days" in df.columns:
        df["age_group"] = (
            pd.cut(df["age_days"], bins=[0, 90, 365, 730, 3650], labels=[0, 1, 2, 3])
            .astype(float)
            .fillna(3)
        )

    # Фильтруем только те фичи, что есть в DataFrame
    available = [f for f in features if f in df.columns]

    # Добавляем engineered features если их нет в списке
    for ef in ["failure_score", "offline_error_interaction", "age_group"]:
        if ef in df.columns and ef not in available:
            available.append(ef)

    return df, available


# ── Prediction Logic ──────────────────────────────────────────────────


def compute_confidence_score(model, X_row: pd.DataFrame, proba: float) -> float:
    """
    Вычисляет confidence score на основе:
    1. Калибровки модели (расстояние от decision boundary)
    2. Согласованности предсказания с соседними деревьями (std)

    Возвращает значение [0..1], где 1 = максимальная уверенность.
    """
    # Расстояние от decision boundary (0.5)
    distance_from_boundary = abs(proba - 0.5) * 2  # [0..1]

    # Если модель поддерживает предсказания отдельных деревьев — используем std
    try:
        tree_preds = np.array([tree.predict(X_row)[0] for tree in model.estimators_])
        consistency = 1.0 - np.std(tree_preds)  # [0..1], выше = стабильнее
    except Exception:
        consistency = 0.5  # fallback

    # Взвешенная комбинация
    confidence = 0.7 * distance_from_boundary + 0.3 * consistency
    return round(float(np.clip(confidence, 0.0, 1.0)), 4)


def compute_calibration_bin(proba: float, n_bins: int = 10) -> int:
    """Определяет бин калибровки [0..n_bins-1]."""
    return min(int(proba * n_bins), n_bins - 1)


def extract_top_features(
    model, features: list, X_row: pd.Series, top_n: int = 3,
) -> list:
    """Извлекает топ-N признаков с наибольшим влиянием."""
    if not hasattr(model, "feature_importances_"):
        return []

    importance = model.feature_importances_
    total = importance.sum()
    if total > 0:
        importance = importance / total

    # Получаем значения признаков для строки
    feat_values = []
    for f, imp in zip(features, importance):
        val = float(X_row.get(f, 0))
        feat_values.append({"feature": f, "importance": round(float(imp), 4), "value": val})

    # Сортируем по importance
    feat_values.sort(key=lambda x: x["importance"], reverse=True)
    return feat_values[:top_n]


def detect_anomaly(model, X_row: pd.DataFrame, threshold: float = 3.0) -> bool:
    """
    Простая детекция аномалий на основе реконструкции.
    Если модель XGBoost — используем расстояние до среднего предсказания.
    """
    try:
        tree_preds = np.array([tree.predict(X_row)[0] for tree in model.estimators_])
        mean_pred = np.mean(tree_preds)
        std_pred = np.std(tree_preds)
        if std_pred > 0 and abs(mean_pred - 0.5) / std_pred > threshold:
            return True
    except Exception:
        pass
    return False


# ── Main Prediction ───────────────────────────────────────────────────


def predict(
    cfg: dict,
    conn,
    model,
    metadata: dict,
    df: pd.DataFrame,
    trace_id: str = "",
    variant: str = "A",
    test_mode: bool = False,
) -> list:
    """
    Генерирует предсказания для всех устройств в DataFrame.

    Возвращает список словарей с предсказаниями (для JSON вывода).
    """
    df, available_features = prepare_features(df, cfg)
    model_version = metadata.get("version", "xgboost_v1")
    threshold = cfg["service"]["probability_threshold"]
    min_confidence = cfg["service"]["min_confidence_threshold"]
    window_days = cfg["ml"]["prediction_window_days"]

    # Проверяем, что все необходимые фичи есть
    model_features = getattr(model, "feature_names_in_", available_features)
    missing = set(model_features) - set(df.columns)
    if missing:
        print(f"[predict] WARNING: Missing features: {missing}", file=sys.stderr)
        for m in missing:
            df[m] = 0.0

    # Берём только фичи, которые ожидает модель
    X = df[[c for c in model_features if c in df.columns]]

    if X.empty:
        print("[predict] ERROR: No features available for prediction", file=sys.stderr)
        return []

    # ── Predict ──
    probas = model.predict_proba(X)[:, 1]
    anomalies = []
    try:
        anomalies = [detect_anomaly(model, X.iloc[[i]]) for i in range(len(X))]
    except Exception:
        anomalies = [False] * len(X)

    results = []
    for idx, row in df.iterrows():
        device_id = row.get("device_id", f"unknown_{idx}")
        proba = float(probas[idx])

        if proba < min_confidence:
            continue

        # Строка признаков
        X_row = X.iloc[[idx]] if hasattr(X, "iloc") else pd.DataFrame([X[idx]])

        confidence = compute_confidence_score(model, X_row, proba)
        calibration_bin = compute_calibration_bin(proba)
        is_actionable = proba >= threshold
        is_anomaly = anomalies[idx] if idx < len(anomalies) else False

        # Feature snapshot
        features_snapshot = {}
        for f in available_features:
            val = row.get(f)
            if isinstance(val, (np.integer,)):
                val = int(val)
            elif isinstance(val, (np.floating,)):
                val = float(val)
            features_snapshot[f] = val

        top_features = extract_top_features(model, available_features, row)

        result = {
            "device_id": device_id,
            "failure_probability": round(proba, 4),
            "confidence_score": confidence,
            "model_version": model_version,
            "model_variant": variant,
            "prediction_date": datetime.now(timezone.utc).isoformat(),
            "prediction_window_days": window_days,
            "is_actionable": is_actionable,
            "is_anomaly": is_anomaly,
            "calibration_bin": calibration_bin,
            "top_features": top_features,
            "features_snapshot": features_snapshot,
            "trace_id": trace_id,
        }
        results.append(result)

    return results


def save_predictions(conn, results: list):
    """Сохраняет предсказания в таблицу predictions."""
    if not results:
        print("[predict] No predictions to save.", file=sys.stderr)
        return

    cur = conn.cursor()
    saved = 0
    for r in results:
        try:
            cur.execute(
                """
                INSERT INTO predictions
                    (device_id, prediction_date, failure_probability, confidence_score,
                     model_version, model_variant, features_snapshot, top_features,
                     prediction_window_days, is_actionable, is_anomaly, calibration_bin,
                     trace_id)
                VALUES (%s, %s, %s, %s, %s, %s, %s::jsonb, %s::jsonb, %s, %s, %s, %s, %s)
                """,
                (
                    r["device_id"],
                    r["prediction_date"],
                    r["failure_probability"],
                    r["confidence_score"],
                    r["model_version"],
                    r["model_variant"],
                    json.dumps(r["features_snapshot"]),
                    json.dumps(r["top_features"]),
                    r["prediction_window_days"],
                    r["is_actionable"],
                    r["is_anomaly"],
                    r["calibration_bin"],
                    r["trace_id"],
                ),
            )
            saved += 1
        except Exception as e:
            print(f"[predict] Error saving prediction for {r['device_id']}: {e}",
                  file=sys.stderr)

    conn.commit()
    cur.close()
    print(f"[predict] Saved {saved}/{len(results)} predictions to DB.", file=sys.stderr)


def output_json(results: list):
    """
    Выводит предсказания в JSONL формате (одна строка JSON на предсказание)
    в stdout для передачи в Go service через subprocess.

    Последняя строка — мета-информация: {"_meta": {"total": N, "status": "ok"}}
    """
    for r in results:
        print(json.dumps(r, ensure_ascii=False, default=str))

    # Мета-строка
    meta = {
        "_meta": {
            "total": len(results),
            "actionable": sum(1 for r in results if r["is_actionable"]),
            "avg_probability": round(
                sum(r["failure_probability"] for r in results) / max(len(results), 1), 4
            ),
            "status": "ok",
            "timestamp": datetime.now(timezone.utc).isoformat(),
        }
    }
    print(json.dumps(meta, ensure_ascii=False, default=str))


# ── CLI ───────────────────────────────────────────────────────────────


def parse_args():
    parser = argparse.ArgumentParser(
        description="P2-1.1: XGBoost Device Failure Prediction",
    )
    parser.add_argument("--test", action="store_true", help="Test mode (no DB save)")
    parser.add_argument("--device", type=str, default=None, help="Single device ID")
    parser.add_argument("--variant", type=str, default=None,
                        help=f"Model variant: A or B (default: from config)")
    parser.add_argument("--trace", type=str, default="", help="Trace ID")
    return parser.parse_args()


def main():
    args = parse_args()
    cfg = load_config()

    trace_id = args.trace or generate_trace_id()
    variant = args.variant or cfg["ml"].get("model_variant", "A")

    is_test = args.test
    if is_test:
        print("[predict] ═══ TEST MODE ═══", file=sys.stderr)
        print("[predict] Predictions will NOT be saved to DB.", file=sys.stderr)

    print(f"[predict] P2-1.1: Device Failure Prediction"
          f" (variant={variant}, trace={trace_id[:16]}...)", file=sys.stderr)

    # ── 1. Load model ──
    model_path, meta_path = get_model_paths(cfg, variant)
    if not os.path.exists(model_path):
        print(f"[predict] ERROR: Model not found: {model_path}", file=sys.stderr)
        sys.exit(1)

    model = joblib.load(model_path)
    metadata = {}
    if os.path.exists(meta_path):
        with open(meta_path) as f:
            metadata = json.load(f)

    model_version = metadata.get("version", "xgboost_v1")
    print(f"[predict] Loaded model: {model_version} from {model_path}", file=sys.stderr)

    # ── 2. Connect to TimescaleDB ──
    conn = psycopg2.connect(**cfg["db"])

    # ── 3. Load features ──
    df = load_features(conn, cfg, device_id=args.device)
    if df.empty:
        print("[predict] No devices found. Nothing to predict.", file=sys.stderr)
        output_json([])
        conn.close()
        return

    print(f"[predict] Loaded features for {len(df)} devices.", file=sys.stderr)

    # ── 4. Predict ──
    results = predict(cfg, conn, model, metadata, df,
                      trace_id=trace_id, variant=variant, test_mode=is_test)

    # ── 5. Output JSON (stdout — для Go subprocess) ──
    output_json(results)

    # ── 6. Save to DB (skip in test mode) ──
    if not is_test and results:
        save_predictions(conn, results)

    conn.close()

    actionable = sum(1 for r in results if r["is_actionable"])
    print(f"[predict] Done. {len(results)} predictions, "
          f"{actionable} actionable (>{cfg['service']['probability_threshold']:.0%}).",
          file=sys.stderr)


if __name__ == "__main__":
    main()
