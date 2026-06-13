#!/usr/bin/env python3
import yaml
import psycopg2
import pandas as pd
import joblib
import requests
from datetime import datetime
from etl import extract_features

def generate_explanation(device_id, features, proba, api_key, model="deepseek-chat"):
    prompt = f"""
    Устройство видеонаблюдения ID: {device_id}
    Признаки за последние 30 дней:
    - Доля времени оффлайн: {features['offline_ratio']:.2%}
    - Количество ошибок: {int(features['error_count'])}
    - Количество перезагрузок: {int(features['reboot_count'])}
    - Возраст: {features['age_days']} дней
    - Средний приоритет тревог: {features['avg_alarm_priority']:.1f}
    - Последний код ошибки: {int(features['last_error_code'])}
    Вероятность отказа в ближайшие 30 дней по модели: {proba:.1%}
    Объясни кратко, какие факторы больше всего влияют на этот прогноз.
    """
    headers = {"Authorization": f"Bearer {api_key}", "Content-Type": "application/json"}
    data = {"model": model, "messages": [{"role": "user", "content": prompt}], "temperature": 0.3}
    try:
        resp = requests.post("https://api.deepseek.com/v1/chat/completions", json=data, headers=headers, timeout=5)
        explanation = resp.json()['choices'][0]['message']['content']
        return explanation
    except:
        return "Explanation unavailable"

def predict_all():
    with open('config.yaml') as f:
        cfg = yaml.safe_load(f)

    conn = psycopg2.connect(**cfg['db'])
    model = joblib.load(cfg['ml']['model_path'])

    df = extract_features(conn, device_id=None, days=30)
    if df.empty:
        print("No devices found, skipping predictions")
        return

    features = cfg['ml']['feature_columns']
    # Приведение всех признаков к числовым типам (float), заменяем NULL на 0
    for col in features:
        df[col] = pd.to_numeric(df[col], errors='coerce').fillna(0).astype(float)

    X = df[features]
    proba = model.predict_proba(X)[:, 1]

    cur = conn.cursor()
    for idx, row in df.iterrows():
        device_id = row['device_id']
        p = proba[idx]
        expl = None
        if cfg['deepseek']['enabled']:
            expl = generate_explanation(device_id, row[features].to_dict(), p, cfg['deepseek']['api_key'])
        cur.execute("""
            INSERT INTO predictions (device_id, prediction_date, failure_probability, explanation, model_version)
            VALUES (%s, %s, %s, %s, %s)
        """, (device_id, datetime.now(), p, expl, 'xgboost_v1'))
    conn.commit()
    cur.close()
    conn.close()
    print(f"Predictions saved for {len(df)} devices")

if __name__ == "__main__":
    predict_all()