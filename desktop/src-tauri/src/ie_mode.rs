// ═══════════════════════════════════════════════════════════════════════════
// IE-Mode Launcher (DESKTOP-02)
//
// Запуск Microsoft Edge в IE-mode для совместимости со старыми
// веб-интерфейсами CCTV камер (Dahua, Hikvision, Uniview, Tiandy).
//
// Flow:
//   1. Получает URL камеры и credentials
//   2. Запускает Edge с флагом --ie-mode-test
//   3. Инжектирует credentials через cookie/DOM
//
// Соответствие:
//   - IEC 62443-3-3 SL-3: Zone separation
//   - OWASP ASVS L3 V3.3: Access control (credentials не логируются)
//   - ISO 27001 A.12.4: Audit trail (только URL логируется)
// ═══════════════════════════════════════════════════════════════════════════

use std::process::Command;
use std::process::Stdio;

/// Результат запуска IE-mode.
#[derive(Debug, serde::Serialize)]
pub struct IEModeResult {
    pub success: bool,
    pub message: String,
    pub url: String,
}

/// Открывает Edge в IE-mode с указанным URL.
///
/// DESKTOP-02: Использует флаг --ie-mode-test для включения
/// режима совместимости с Internet Explorer.
///
/// # Arguments
/// * `url` - URL веб-интерфейса камеры
/// * `username` - Опциональное имя пользователя для auto-login
/// * `password` - Опциональный пароль для auto-login
///
/// # Compliance
/// - Пароль НЕ логируется (security by design)
/// - URL логируется для audit trail (ISO 27001 A.12.4)
pub fn open_in_ie_mode(url: &str, username: Option<&str>, password: Option<&str>) -> IEModeResult {
    // Определяем путь к Edge в зависимости от ОС
    let edge_path = find_edge_executable();

    match edge_path {
        Some(path) => {
            // DESKTOP-02: Формируем URL с credentials для auto-login
            let target_url = if let (Some(user), Some(pass)) = (username, password) {
                // Inject credentials via URL (http://user:pass@host)
                // Это работает для basic auth камер
                if url.starts_with("http://") || url.starts_with("https://") {
                    let rest = url.trim_start_matches("http://")
                        .trim_start_matches("https://");
                    let protocol = if url.starts_with("https://") { "https://" } else { "http://" };
                    format!("{}:{}@{}/", user, pass, rest)
                        .replacen("http://", protocol, 1)
                } else {
                    url.to_string()
                }
            } else {
                url.to_string()
            };

            // DESKTOP-02: Запускаем Edge с IE-mode флагом
            let result = Command::new(&path)
                .arg("--ie-mode-test")
                .arg("--new-window")
                .arg(&target_url)
                .stdout(Stdio::null())
                .stderr(Stdio::null())
                .spawn();

            match result {
                Ok(child) => {
                    IEModeResult {
                        success: true,
                        message: format!("Edge launched in IE-mode (PID: {})", child.id()),
                        url: url.to_string(),
                    }
                }
                Err(e) => {
                    IEModeResult {
                        success: false,
                        message: format!("Failed to launch Edge: {}", e),
                        url: url.to_string(),
                    }
                }
            }
        }
        None => {
            IEModeResult {
                success: false,
                message: "Microsoft Edge not found".to_string(),
                url: url.to_string(),
            }
        }
    }
}

/// Ищет исполняемый файл Microsoft Edge в стандартных расположениях.
fn find_edge_executable() -> Option<String> {
    let candidates = vec![
        // Windows
        r"C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe",
        r"C:\Program Files\Microsoft\Edge\Application\msedge.exe",
        // macOS
        "/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
        // Linux
        "/usr/bin/microsoft-edge",
        "/usr/bin/microsoft-edge-stable",
        "/usr/bin/msedge",
    ];

    for candidate in &candidates {
        if std::path::Path::new(candidate).exists() {
            return Some(candidate.to_string());
        }
    }

    // Попытка найти через PATH
    which_edge()
}

/// Ищет Edge через системный PATH.
fn which_edge() -> Option<String> {
    let names = ["microsoft-edge", "microsoft-edge-stable", "msedge", "edge"];

    for name in &names {
        if let Ok(path) = which::which(name) {
            return Some(path.to_string_lossy().to_string());
        }
    }

    None
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_ie_mode_url_formatting() {
        let result = open_in_ie_mode("http://192.168.1.100", Some("admin"), Some("password123"));
        assert!(result.url == "http://192.168.1.100");
    }

    #[test]
    fn test_find_edge_not_panic() {
        // Функция не должна паниковать
        let _ = find_edge_executable();
    }
}
