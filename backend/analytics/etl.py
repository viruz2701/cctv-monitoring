import pandas as pd

def extract_features(conn, device_id=None, days=30):
    """
    Извлекает агрегированные признаки для устройства(в) за последние days дней.
    Если device_id указан – только для него, иначе для всех.
    """
    # Условия фильтрации по device_id
    telemetry_cond = ""
    errors_cond = ""
    reboots_cond = ""
    alarms_cond = ""
    device_cond = ""
    if device_id:
        telemetry_cond = f"AND device_id = '{device_id}'"
        errors_cond = f"AND device_id = '{device_id}'"
        reboots_cond = f"AND device_id = '{device_id}'"
        alarms_cond = f"AND device_id = '{device_id}'"
        device_cond = f"WHERE device_id = '{device_id}'"

    # Формируем запрос с прямой подстановкой days (безопасно, т.к. days - int)
    query = f"""
    WITH telemetry_last_days AS (
        SELECT device_id, status, time,
               LAG(status) OVER (PARTITION BY device_id ORDER BY time) as prev_status
        FROM telemetry
        WHERE time > NOW() - {days} * INTERVAL '1 day'
        {telemetry_cond}
    ),
    offline_calc AS (
        SELECT device_id,
               COUNT(*) FILTER (WHERE status = 'OFFLINE') * 1.0 / COUNT(*) AS offline_ratio
        FROM telemetry_last_days
        GROUP BY device_id
    ),
    errors AS (
        SELECT device_id, COUNT(*) AS error_count,
               COUNT(DISTINCT event_code) AS unique_errors
        FROM parsed_logs
        WHERE log_level = 'ERROR' AND time > NOW() - {days} * INTERVAL '1 day'
        {errors_cond}
        GROUP BY device_id
    ),
    reboots AS (
        SELECT device_id, COUNT(*) AS reboot_count
        FROM alarms
        WHERE method = 6 AND time > NOW() - {days} * INTERVAL '1 day'
        {reboots_cond}
        GROUP BY device_id
    ),
    alarms_agg AS (
        SELECT device_id, AVG(priority) AS avg_alarm_priority
        FROM alarms
        WHERE time > NOW() - {days} * INTERVAL '1 day'
        {alarms_cond}
        GROUP BY device_id
    ),
    device_info AS (
        SELECT device_id,
               EXTRACT(DAY FROM (NOW() - registered_at)) AS age_days,
               (SELECT event_code FROM parsed_logs WHERE device_id = d.device_id ORDER BY time DESC LIMIT 1) AS last_error_code
        FROM devices d
        {device_cond}
    )
    SELECT
        d.device_id,
        COALESCE(o.offline_ratio, 0) AS offline_ratio,
        COALESCE(e.error_count, 0) AS error_count,
        COALESCE(r.reboot_count, 0) AS reboot_count,
        COALESCE(di.age_days, 0) AS age_days,
        COALESCE(a.avg_alarm_priority, 2) AS avg_alarm_priority,
        COALESCE(di.last_error_code, 0) AS last_error_code
    FROM devices d
    LEFT JOIN offline_calc o USING(device_id)
    LEFT JOIN errors e USING(device_id)
    LEFT JOIN reboots r USING(device_id)
    LEFT JOIN alarms_agg a USING(device_id)
    LEFT JOIN device_info di USING(device_id)
    """
    # Выполняем запрос без параметров (подстановка уже сделана)
    df = pd.read_sql(query, conn)
    return df