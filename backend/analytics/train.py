import yaml
import psycopg2
import pandas as pd
import xgboost as xgb
from sklearn.model_selection import train_test_split
from sklearn.metrics import roc_auc_score
import joblib
import os

def train():
    # Определяем путь к config.yaml (находится в той же директории)
    base_dir = os.path.dirname(os.path.abspath(__file__))
    config_path = os.path.join(base_dir, 'config.yaml')
    
    with open(config_path) as f:
        cfg = yaml.safe_load(f)

    conn = psycopg2.connect(**cfg['db'])
    
    # Для примера создадим фейковые тренировочные данные,
    # так как реальной размеченной таблицы training_data пока нет.
    # В реальности вы должны создать view или таблицу с историческими отказами.
    # Здесь создадим случайные данные для демонстрации.
    import numpy as np
    np.random.seed(42)
    n_samples = 1000
    features = cfg['ml']['feature_columns']
    X = pd.DataFrame({
        'offline_ratio': np.random.uniform(0, 0.5, n_samples),
        'error_count': np.random.poisson(5, n_samples),
        'reboot_count': np.random.poisson(2, n_samples),
        'age_days': np.random.uniform(0, 1000, n_samples),
        'avg_alarm_priority': np.random.uniform(1, 3, n_samples),
        'last_error_code': np.random.choice([0, 100, 200, 404], n_samples)
    })
    y = (X['offline_ratio'] > 0.3) | (X['error_count'] > 10) | (X['reboot_count'] > 5)
    y = y.astype(int)
    
    X_train, X_test, y_train, y_test = train_test_split(X, y, test_size=0.2, random_state=42)

    model = xgb.XGBClassifier(
        n_estimators=100,
        max_depth=5,
        learning_rate=0.1,
        random_state=42
    )
    model.fit(X_train, y_train)

    y_pred_proba = model.predict_proba(X_test)[:, 1]
    auc = roc_auc_score(y_test, y_pred_proba)
    print(f"Model AUC: {auc:.4f}")

    # Сохраняем модель
    model_path = os.path.join(base_dir, cfg['ml']['model_path'])
    os.makedirs(os.path.dirname(model_path), exist_ok=True)
    joblib.dump(model, model_path)
    print(f"Model saved to {model_path}")

if __name__ == "__main__":
    train()