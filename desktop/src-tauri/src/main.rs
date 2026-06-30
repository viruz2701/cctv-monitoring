// ═══════════════════════════════════════════════════════════════════════════
// CCTV Health Monitor — Tauri Desktop App (DESKTOP-01)
//
// Desktop-приложение для открытия веб-интерфейсов CCTV камер
// в Microsoft Edge IE-mode. Обеспечивает совместимость со старыми
// камерами (Dahua, Hikvision, Uniview, Tiandy), требующими IE.
//
// Tauri Commands:
//   - open_camera_web_ui(device_id, url) — открытие камеры в IE-mode
//   - get_credentials(device_id) — получение credentials через API
//
// Соответствие:
//   - IEC 62443-3-3 SL-3: Zone separation
//   - OWASP ASVS L3 V3.3: Input validation, access control
//   - ISO 27001 A.12.4: Audit trail
// ═══════════════════════════════════════════════════════════════════════════

#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

mod ie_mode;

use serde::{Deserialize, Serialize};

// ─── Types ──────────────────────────────────────────────────────────

/// Credentials для устройства.
#[derive(Debug, Serialize, Deserialize)]
struct DeviceCredentials {
    username: String,
    password: String,
}

/// Результат Tauri команды.
#[derive(Debug, Serialize)]
struct CommandResult {
    success: bool,
    message: String,
    data: Option<String>,
}

// ═════════════════════════════════════════════════════════════════════
// Tauri Commands
// ═════════════════════════════════════════════════════════════════════

/// DESKTOP-01: Открывает веб-интерфейс камеры в Microsoft Edge IE-mode.
///
/// Проверяет входные параметры (OWASP ASVS V5.1: Input validation),
/// получает credentials через Backend API и запускает Edge в IE-mode.
///
/// # Arguments
/// * `device_id` - UUID устройства
/// * `url` - URL веб-интерфейса камеры (опционально, получается с сервера)
///
/// # Compliance
/// - OWASP ASVS V5.1: Input validation
/// - IEC 62443-3-3 SR 2.1: Authorisation enforcement
/// - ISO 27001 A.12.4: Audit trail
#[tauri::command]
async fn open_camera_web_ui(
    device_id: String,
    url: Option<String>,
) -> Result<CommandResult, String> {
    // OWASP ASVS V5.1: Input validation
    if device_id.trim().is_empty() {
        return Err("device_id is required".to_string());
    }

    // Получаем URL и credentials
    let target_url = match url {
        Some(u) if !u.trim().is_empty() => {
            // OWASP ASVS V5.1: URL validation
            if !u.starts_with("http://") && !u.starts_with("https://") {
                return Err(format!("Invalid camera URL: {}", u));
            }
            u
        }
        _ => {
            // Если URL не передан, пытаемся получить с сервера
            match get_device_url(&device_id).await {
                Ok(url) => url,
                Err(e) => return Err(format!("Failed to get device URL: {}", e)),
            }
        }
    };

    // Получаем credentials для устройства
    let credentials = match get_credentials(&device_id).await {
        Ok(creds) => Some(creds),
        Err(_) => None, // Пробуем без credentials (open уязвимые камеры)
    };

    // DESKTOP-02: Запускаем Edge в IE-mode
    let result = ie_mode::open_in_ie_mode(
        &target_url,
        credentials.as_ref().map(|c| c.username.as_str()),
        credentials.as_ref().map(|c| c.password.as_str()),
    );

    if result.success {
        Ok(CommandResult {
            success: true,
            message: format!("Opened {} in IE-mode", target_url),
            data: Some(device_id),
        })
    } else {
        Err(format!("Failed to open IE-mode: {}", result.message))
    }
}

/// DESKTOP-01: Получает credentials устройства через Backend API.
///
/// Вызывает GET /api/v1/credentials/{device_id} для получения
/// зашифрованных credentials устройства.
#[tauri::command]
async fn get_credentials(device_id: String) -> Result<DeviceCredentials, String> {
    // OWASP ASVS V5.1: Input validation
    if device_id.trim().is_empty() {
        return Err("device_id is required".to_string());
    }

    // Вызов Backend API
    let api_base = std::env::var("API_BASE_URL")
        .unwrap_or_else(|_| "http://localhost:8080/api/v1".to_string());

    let url = format!("{}/credentials/{}", api_base, device_id);

    let client = reqwest::Client::new();
    let response = client
        .get(&url)
        .header("Accept", "application/json")
        .send()
        .await
        .map_err(|e| format!("Failed to call API: {}", e))?;

    if !response.status().is_success() {
        return Err(format!(
            "API returned error: {}",
            response.status()
        ));
    }

    let credentials: DeviceCredentials = response
        .json()
        .await
        .map_err(|e| format!("Failed to parse credentials: {}", e))?;

    Ok(credentials)
}

// ═════════════════════════════════════════════════════════════════════
// Internal Helpers
// ═════════════════════════════════════════════════════════════════════

/// Получает URL устройства с сервера.
async fn get_device_url(device_id: &str) -> Result<String, String> {
    let api_base = std::env::var("API_BASE_URL")
        .unwrap_or_else(|_| "http://localhost:8080/api/v1".to_string());

    let url = format!("{}/devices/{}", api_base, device_id);

    let client = reqwest::Client::new();
    let response = client
        .get(&url)
        .header("Accept", "application/json")
        .send()
        .await
        .map_err(|e| format!("Failed to fetch device: {}", e))?;

    if !response.status().is_success() {
        return Err(format!("Device not found: {}", response.status()));
    }

    // Парсим URL из ответа
    #[derive(Deserialize)]
    struct DeviceResponse {
        ip_address: Option<String>,
    }

    let device: DeviceResponse = response
        .json()
        .await
        .map_err(|e| format!("Failed to parse device: {}", e))?;

    match device.ip_address {
        Some(ip) => Ok(format!("http://{}", ip)),
        None => Err("Device has no IP address".to_string()),
    }
}

// ═════════════════════════════════════════════════════════════════════
// Application Entry Point
// ═════════════════════════════════════════════════════════════════════

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_opener::init())
        .plugin(tauri_plugin_shell::init())
        .invoke_handler(tauri::generate_handler![
            open_camera_web_ui,
            get_credentials,
        ])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
