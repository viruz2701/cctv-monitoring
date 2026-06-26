#!/usr/bin/env python3
"""
P2-1.1: XGBoost Model Training — Device Failure Prediction
===========================================================

Загружает реальные данные из TimescaleDB, обучает XGBoost модель
для предсказания отказов устройств видеонаблюдения.

Features:
  - offline_ratio         — доля времени в оффлайн за 30 дней
  - error_count           — количество ошибок за 30 дней
  - reboot_count          — количество перезагрузок за 30 дней
  - age_days              — возраст устройства (дней)
  - avg_alarm_priority    — средний приоритет тревог
  - last_error_code       — последний код ошибки
  - unique_error_types    — количество уникальных типов ошибок
  - avg_cpu_load          — средняя загрузка CPU
  - avg_memory_usage      — среднее использование памяти
  - signal_strength       — средняя сила сигнала

Target: has_failed (1 если устройство отказало в следующие 30 дней)

Output:
  - models/xgboost_model.pkl      — обученная модель
  - models/xgboost_metadata.json  — метаданные (AUC, feature importance)

Usage:
  python3 train.py
"""

import json
import os
import sys
import warnings
from datetime import datetime, timezone

import joblib
import numpy as np
import pandas as pd
import psycopg2
import yaml
from sklearn.metrics import (
    accuracy_score,
    classification_report,
    confusion_matrix,
    precision_recall_curve,
    roc_auc_score,
)
from sklearn.model_selection import StratifiedKFold, cross_val_predict, train_test_split
from sklearn.preprocessing import LabelEncoder

warnings.filterwarnings("ignore")

# ── Constants ─────────────────────────────────────────────────────────

BASE_DIR = os.path.dirname(os.path.abspath(__file__))
CONFIG_PATH = os.path.join(BASE_DIR, "config.yaml")
MODEL_DIR = os.path.join(BASE_DIR, "models")
RANDOM_STATE = 42
N_FOLDS = 5
TARGET_ACCURACY = 0.75


def load_config() -> dict:
    """Загружает конфигурацию из config.yaml."""
    with open(CONFIG_PATH) as f:
        return yaml.safe_load(f)


def load_training_data(conn, cfg: dict) -> pd.DataFrame:
    """
    Загружает реальные данные из TimescaleDB.

    Использует extract_features для получения признаков и создаёт
    целевую переменную на основе исторических отказов (alarms с method=6
    или critical событий в parsed_logs).
    """
    # Импортируем ETL (в той же директории)
    sys.path.insert(0, BASE_DIR)
    from etl import extract_features  # noqa

    days = cfg["ml"]["prediction_window_days"]

    print(f"[train] Loading features for last {days} days from TimescaleDB...")
    df = extract_features(conn, device_id=None, days=days)

    if df.empty:
        print("[train] WARNING: No data returned from extract_features.")
        print("[train] Falling back to synthetic training data for bootstrap.")
        return _generate_synthetic_data(cfg)

    print(f"[train] Loaded {len(df)} devices from DB. Generating labels...")

    # ── Generate labels: has_failed based on actual failure events ──
    cur = conn.cursor()

    # Определяем отказ как: reboot (method=6) ИЛИ critical alarm ИЛИ device marked as failed
    label_query = """
    WITH device_failures AS (
        SELECT DISTINCT device_id
        FROM alarms
        WHERE (method = 6 OR priority >= 3)
          AND time > NOW() - %s * INTERVAL '1 day'
          AND time <= NOW()
        UNION
        SELECT DISTINCT device_id
        FROM parsed_logs
        WHERE log_level = 'CRITICAL'
          AND time > NOW() - %s * INTERVAL '1 day'
          AND time <= NOW()
    )
    SELECT device_id, 1 AS has_failed FROM device_failures
    """
    cur.execute(label_query, (days, days))
    failure_map = {row[0]: row[1] for row in cur.fetchall()}
    cur.close()

    df["has_failed"] = df["device_id"].map(failure_map).fillna(0).astype(int)

    failed_count = df["has_failed"].sum()
    total_count = len(df)
    print(f"[train] Labels: {failed_count}/{total_count} devices marked as failed "
          f"({failed_count / total_count:.1%})")

    if failed_count < 10:
        print("[train] WARNING: Too few failure samples. Augmenting with synthetic data.")
        synth = _generate_synthetic_data(cfg, n_samples=max(200, total_count))
        df = pd.concat([df, synth], ignore_index=True)
        print(f"[train] Augmented dataset: {len(df)} samples, "
              f"{df['has_failed'].sum()} failures ({df['has_failed'].mean():.1%})")

    return df


def _generate_synthetic_data(cfg: dict, n_samples: int = 1000) -> pd.DataFrame:
    """Генерирует синтетические данные для bootstrap-обучения."""
    np.random.seed(RANDOM_STATE)
    features = cfg["ml"]["feature_columns"]

    data = {
        "offline_ratio": np.random.uniform(0, 0.5, n_samples),
        "error_count": np.random.poisson(5, n_samples).astype(float),
        "reboot_count": np.random.poisson(2, n_samples).astype(float),
        "age_days": np.random.uniform(0, 1000, n_samples),
        "avg_alarm_priority": np.random.uniform(1, 3, n_samples),
        "last_error_code": np.random.choice([0, 100, 200, 404, 500], n_samples).astype(float),
        "unique_error_types": np.random.poisson(2, n_samples).astype(float),
        "avg_cpu_load": np.random.uniform(10, 95, n_samples),
        "avg_memory_usage": np.random.uniform(20, 98, n_samples),
        "signal_strength": np.random.uniform(-90, -30, n_samples),
        "device_id": [f"synth_{i:04d}" for i in range(n_samples)],
    }

    df = pd.DataFrame(data)

    # Логика отказа: комбинация признаков
    failure_conditions = (
        (df["offline_ratio"] > 0.3)
        | (df["error_count"] > 10)
        | (df["reboot_count"] > 5)
        | ((df["age_days"] > 700) & (df["avg_cpu_load"] > 80))
        | ((df["signal_strength"] < -75) & (df["error_count"] > 5))
    )
    df["has_failed"] = failure_conditions.astype(int)

    print(f"[train] Generated {n_samples} synthetic samples "
          f"({df['has_failed'].sum()} failures, {df['has_failed'].mean():.1%})")
    return df


def engineer_features(df: pd.DataFrame, cfg: dict) -> pd.DataFrame:
    """
    Feature engineering: создаёт дополнительные признаки и
    приводит всё к числовому виду.
    """
    features = cfg["ml"]["feature_columns"]

    # Приводим все признаки к float, заполняем NaN нулями
    for col in features:
        if col in df.columns:
            df[col] = pd.to_numeric(df[col], errors="coerce").fillna(0).astype(float)
        else:
            df[col] = 0.0

    # ── Создаём дополнительный признак: failure score ──
    if all(f in df.columns for f in ["offline_ratio", "error_count", "age_days"]):
        df["failure_score"] = (
            df["offline_ratio"] * 3.0
            + np.log1p(df["error_count"]) * 0.5
            + np.log1p(df["age_days"]) * 0.01
        )
        if "failure_score" not in features:
            features.append("failure_score")

    # ── Interaction: offline × errors ──
    if "offline_ratio" in df.columns and "error_count" in df.columns:
        df["offline_error_interaction"] = df["offline_ratio"] * np.log1p(df["error_count"])
        if "offline_error_interaction" not in features:
            features.append("offline_error_interaction")

    # ── Биннинг возраста ──
    if "age_days" in df.columns:
        df["age_group"] = pd.cut(
            df["age_days"],
            bins=[0, 90, 365, 730, 3650],
            labels=[0, 1, 2, 3],
        ).astype(float).fillna(3)
        if "age_group" not in features:
            features.append("age_group")

    cfg["ml"]["feature_columns"] = features
    return df


def train_xgboost(X_train, y_train, X_val, y_val, cfg: dict):
    """
    Обучает XGBoost классификатор с гиперпараметрами из конфига.
    Использует early stopping для предотвращения переобучения.
    """
    import xgboost as xgb

    xgb_cfg = cfg["ml"]["xgboost"]

    model = xgb.XGBClassifier(
        n_estimators=xgb_cfg.get("n_estimators", 200),
        max_depth=xgb_cfg.get("max_depth", 6),
        learning_rate=xgb_cfg.get("learning_rate", 0.08),
        subsample=xgb_cfg.get("subsample", 0.8),
        colsample_bytree=xgb_cfg.get("colsample_bytree", 0.8),
        min_child_weight=xgb_cfg.get("min_child_weight", 3),
        gamma=xgb_cfg.get("gamma", 0.1),
        reg_alpha=xgb_cfg.get("reg_alpha", 0.1),
        reg_lambda=xgb_cfg.get("reg_lambda", 1.0),
        scale_pos_weight=xgb_cfg.get("scale_pos_weight", 3.0),
        random_state=RANDOM_STATE,
        eval_metric=xgb_cfg.get("eval_metric", ["auc", "logloss"]),
        early_stopping_rounds=xgb_cfg.get("early_stopping_rounds", 20),
        verbosity=1,
    )

    model.fit(
        X_train, y_train,
        eval_set=[(X_val, y_val)],
        verbose=False,
    )

    return model


def evaluate_model(model, X_test, y_test, features: list) -> dict:
    """
    Оценивает модель и возвращает метрики.
    Цель: accuracy > 75%, AUC > 0.8
    """
    y_pred = model.predict(X_test)
    y_proba = model.predict_proba(X_test)[:, 1]

    accuracy = accuracy_score(y_test, y_pred)
    auc = roc_auc_score(y_test, y_proba)

    tn, fp, fn, tp = confusion_matrix(y_test, y_pred).ravel()
    precision = tp / (tp + fp) if (tp + fp) > 0 else 0
    recall = tp / (tp + fn) if (tp + fn) > 0 else 0
    f1 = 2 * precision * recall / (precision + recall) if (precision + recall) > 0 else 0

    metrics = {
        "accuracy": round(float(accuracy), 4),
        "auc": round(float(auc), 4),
        "precision": round(float(precision), 4),
        "recall": round(float(recall), 4),
        "f1_score": round(float(f1), 4),
        "true_negatives": int(tn),
        "false_positives": int(fp),
        "false_negatives": int(fn),
        "true_positives": int(tp),
        "target_accuracy": TARGET_ACCURACY,
        "target_met": accuracy >= TARGET_ACCURACY,
    }

    print(f"\n[train] ═══ Model Evaluation ═══")
    print(f"[train]   Accuracy:  {metrics['accuracy']:.2%}  (target: >={TARGET_ACCURACY:.0%}) {'✓' if metrics['target_met'] else '✗'}")
    print(f"[train]   AUC:       {metrics['auc']:.4f}")
    print(f"[train]   Precision: {metrics['precision']:.2%}")
    print(f"[train]   Recall:    {metrics['recall']:.2%}")
    print(f"[train]   F1:        {metrics['f1_score']:.4f}")
    print(f"[train]   Confusion: TN={tn} FP={fp} FN={fn} TP={tp}")
    print(f"[train] ═══════════════════════════\n")

    return metrics


def extract_feature_importance(model, features: list) -> list:
    """Извлекает feature importance из обученной модели."""
    importance = model.feature_importances_
    total = importance.sum()
    if total > 0:
        importance = importance / total

    feat_imp = sorted(
        [
            {"feature": f, "importance": round(float(imp), 4)}
            for f, imp in zip(features, importance)
        ],
        key=lambda x: x["importance"],
        reverse=True,
    )
    return feat_imp


def save_model(model, metadata: dict, cfg: dict):
    """Сохраняет модель и метаданные."""
    os.makedirs(MODEL_DIR, exist_ok=True)

    model_path = os.path.join(BASE_DIR, cfg["ml"]["model_path"])
    joblib.dump(model, model_path)
    print(f"[train] Model saved: {model_path} ({os.path.getsize(model_path) / 1024:.1f} KB)")

    metadata_path = os.path.join(BASE_DIR, cfg["ml"]["model_metadata_path"])
    with open(metadata_path, "w") as f:
        json.dump(metadata, f, indent=2, ensure_ascii=False, default=str)
    print(f"[train] Metadata saved: {metadata_path}")


def main():
    cfg = load_config()
    print(f"[train] P2-1.1: XGBoost Device Failure Prediction — Training")
    print(f"[train] Target accuracy: >= {TARGET_ACCURACY:.0%}")

    # ── 1. Connect to TimescaleDB ──
    conn = psycopg2.connect(**cfg["db"])
    print(f"[train] Connected to {cfg['db']['host']}:{cfg['db']['port']}/{cfg['db']['database']}")

    # ── 2. Load and prepare data ──
    df = load_training_data(conn, cfg)
    df = engineer_features(df, cfg)

    features = cfg["ml"]["feature_columns"]
    label = cfg["ml"]["label_column"]

    # Фильтруем только колонки, которые есть в DataFrame
    available_features = [f for f in features if f in df.columns]
    missing = set(features) - set(available_features)
    if missing:
        print(f"[train] WARNING: Missing features in data: {missing}")

    print(f"[train] Using {len(available_features)} features: {available_features}")
    print(f"[train] Dataset shape: {df.shape}, failure rate: {df[label].mean():.2%}")

    X = df[available_features]
    y = df[label]

    # ── 3. Train/val/test split ──
    X_temp, X_test, y_temp, y_test = train_test_split(
        X, y, test_size=0.15, random_state=RANDOM_STATE, stratify=y,
    )
    X_train, X_val, y_train, y_val = train_test_split(
        X_temp, y_temp, test_size=0.15 / 0.85, random_state=RANDOM_STATE, stratify=y_temp,
    )

    print(f"[train] Split: train={len(X_train)} val={len(X_val)} test={len(X_test)}")

    # ── 4. Cross-validation ──
    print(f"[train] Running {N_FOLDS}-fold cross-validation...")
    skf = StratifiedKFold(n_splits=N_FOLDS, shuffle=True, random_state=RANDOM_STATE)
    cv_probas = cross_val_predict(
        _get_xgb_model(cfg), X, y, cv=skf, method="predict_proba", verbose=0,
    )[:, 1]
    cv_auc = roc_auc_score(y, cv_probas)
    print(f"[train] CV AUC: {cv_auc:.4f}")

    # ── 5. Train final model ──
    print(f"[train] Training final XGBoost model...")
    model = train_xgboost(X_train, y_train, X_val, y_val, cfg)
    best_iter = model.best_iteration + 1 if hasattr(model, "best_iteration") else "N/A"
    print(f"[train] Best iteration: {best_iter}")

    # ── 6. Evaluate ──
    metrics = evaluate_model(model, X_test, y_test, available_features)

    # ── 7. Feature importance ──
    feat_imp = extract_feature_importance(model, available_features)
    print(f"[train] Top-5 features:")
    for i, fi in enumerate(feat_imp[:5], 1):
        print(f"         {i}. {fi['feature']}: {fi['importance']:.4f}")

    # ── 8. Save model ──
    metadata = {
        "model_type": "xgboost",
        "model_variant": cfg["ml"].get("model_variant", "A"),
        "version": "xgboost_v1",
        "training_date": datetime.now(timezone.utc).isoformat(),
        "features": available_features,
        "feature_importance": feat_imp,
        "metrics": metrics,
        "cv_auc": round(float(cv_auc), 4),
        "config": {
            k: cfg["ml"]["xgboost"][k]
            for k in ["n_estimators", "max_depth", "learning_rate",
                       "subsample", "colsample_bytree", "scale_pos_weight"]
        },
        "dataset_size": len(df),
        "failure_rate": float(df[label].mean()),
        "n_features": len(available_features),
    }
    save_model(model, metadata, cfg)

    conn.close()
    print(f"[train] ✓ Training complete. Model variant: {metadata['model_variant']}")
    print(f"[train]   Accuracy: {metrics['accuracy']:.2%} — {'MEETS TARGET' if metrics['target_met'] else 'BELOW TARGET'}")


def _get_xgb_model(cfg: dict):
    """Возвращает XGBClassifier с параметрами из конфига (для CV)."""
    import xgboost as xgb

    xgb_cfg = cfg["ml"]["xgboost"]
    return xgb.XGBClassifier(
        n_estimators=xgb_cfg.get("n_estimators", 200),
        max_depth=xgb_cfg.get("max_depth", 6),
        learning_rate=xgb_cfg.get("learning_rate", 0.08),
        subsample=xgb_cfg.get("subsample", 0.8),
        colsample_bytree=xgb_cfg.get("colsample_bytree", 0.8),
        min_child_weight=xgb_cfg.get("min_child_weight", 3),
        gamma=xgb_cfg.get("gamma", 0.1),
        reg_alpha=xgb_cfg.get("reg_alpha", 0.1),
        reg_lambda=xgb_cfg.get("reg_lambda", 1.0),
        scale_pos_weight=xgb_cfg.get("scale_pos_weight", 3.0),
        random_state=RANDOM_STATE,
        eval_metric=xgb_cfg.get("eval_metric", ["auc", "logloss"]),
        verbosity=0,
    )


if __name__ == "__main__":
    main()
