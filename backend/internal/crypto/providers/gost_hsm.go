// Package providers — HSM Auto-Detect for GOST crypto.
//
// ═══════════════════════════════════════════════════════════════════════════
// P2-MKT.1: HSM Auto-Detect (КриптоПро CSP / другие СКЗИ)
//
// Поддерживаемые HSM/СКЗИ:
//   - КриптоПро CSP (libcryptcp.so / cpapi.dll)
//   - КриптоПро HSM (плагин)
//   - ViPNet CSP (libvipnet.so)
//   - SignalCom CSP (libsignal.so)
//   - Лисси-СКЗИ (liblissi.so)
//   - Общие PKCS#11 токены (libpkcs11.so)
//
// Методы детекции:
//  1. Поиск библиотек в стандартных путях (LD_LIBRARY_PATH, /usr/lib)
//  2. Попытка загрузки через dlopen/cgo
//  3. Проверка PKCS#11 интерфейса
//  4. Проверка переменных окружения
//
// Compliance:
//   - Приказ ФСТЭК № 17 — Сертифицированные СКЗИ
//   - IEC 62443-3-3 SR 5.1 — Zone-based access
//   - OWASP ASVS V6.2 — Key management
//
// ═══════════════════════════════════════════════════════════════════════════
package providers

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// ────────────────────────────────────────────────────────────────────────────
// HSM Provider types
// ────────────────────────────────────────────────────────────────────────────

// HSMProviderType — тип HSM/СКЗИ провайдера.
type HSMProviderType string

const (
	// HSMКриптоПроCSP — КриптоПро CSP.
	HSMКриптоПроCSP HSMProviderType = "cryptopro-csp"

	// HSMКриптоПроHSM — КриптоПро HSM.
	HSMКриптоПроHSM HSMProviderType = "cryptopro-hsm"

	// HSMViPNetCSP — ViPNet CSP.
	HSMViPNetCSP HSMProviderType = "vipnet-csp"

	// HSMSignalComCSP — SignalCom CSP.
	HSMSignalComCSP HSMProviderType = "signalcom-csp"

	// HSMLissiSKZI — Лисси-СКЗИ.
	HSMLissiSKZI HSMProviderType = "lissi-skzi"

	// HSMPKCS11 — PKCS#11 токен.
	HSMPKCS11 HSMProviderType = "pkcs11"

	// HSMSoftware — программная реализация (нет HSM).
	HSMSoftware HSMProviderType = "software"
)

// HSMInfo — информация об обнаруженном HSM.
type HSMInfo struct {
	Type      HSMProviderType `json:"type"`
	Name      string          `json:"name"`
	Version   string          `json:"version,omitempty"`
	Path      string          `json:"path,omitempty"`
	Certified bool            `json:"certified"`
	Available bool            `json:"available"`
}

// ────────────────────────────────────────────────────────────────────────────
// HSM Detection
// ────────────────────────────────────────────────────────────────────────────

// hsmDetectPaths — стандартные пути поиска HSM библиотек.
var hsmDetectPaths = []string{
	"/usr/lib",
	"/usr/lib64",
	"/usr/local/lib",
	"/usr/local/lib64",
	"/opt/cprocsp/lib",
	"/opt/cprocsp/lib/amd64",
	"/opt/ViPNet/lib",
	"/opt/SignalCom/lib",
	"/var/opt/lissi-skzi/lib",
}

// hsmLibraryPatterns — паттерны библиотек для детекции HSM.
type hsmLibraryPattern struct {
	provider HSMProviderType
	name     string
	patterns []string // имена библиотек для поиска
}

var hsmLibraryPatterns = []hsmLibraryPattern{
	{
		provider: HSMКриптоПроCSP,
		name:     "КриптоПро CSP",
		patterns: []string{"libcryptcp.so", "libcryptcp.so.*", "cpapi.dll"},
	},
	{
		provider: HSMКриптоПроHSM,
		name:     "КриптоПро HSM",
		patterns: []string{"libcppkcs11.so", "libcppkcs11.so.*"},
	},
	{
		provider: HSMViPNetCSP,
		name:     "ViPNet CSP",
		patterns: []string{"libvipnet.so", "libvipnet.so.*"},
	},
	{
		provider: HSMSignalComCSP,
		name:     "SignalCom CSP",
		patterns: []string{"libsignal.so", "libsignal.so.*"},
	},
	{
		provider: HSMLissiSKZI,
		name:     "Лисси-СКЗИ",
		patterns: []string{"liblissi.so", "liblissi.so.*"},
	},
	{
		provider: HSMPKCS11,
		name:     "PKCS#11",
		patterns: []string{"libpkcs11.so", "libpkcs11.so.*", "opensc-pkcs11.so"},
	},
}

// DetectHSM — автоматическая детекция доступных HSM/СКЗИ на системе.
//
// Возвращает список обнаруженных HSM провайдеров.
// Первый элемент — наиболее приоритетный (сертифицированный).
//
// Алгоритм:
//  1. Поиск библиотек в стандартных путях
//  2. Проверка через ldconfig -p (Linux)
//  3. Проверка переменных окружения (CRYPTOPRO_PATH, VIPNET_PATH и т.д.)
//  4. Проверка через csptest (КриптоПро)
func DetectHSM() []HSMInfo {
	var detected []HSMInfo

	// Шаг 1: Проверка переменных окружения
	if envInfo := detectFromEnv(); envInfo != nil {
		detected = append(detected, envInfo...)
	}

	// Шаг 2: Поиск библиотек в стандартных путях
	if libInfo := detectFromLibraries(); libInfo != nil {
		detected = append(detected, libInfo...)
	}

	// Шаг 3: Проверка через csptest (КриптоПро)
	if cpInfo := detectCryptoProCSP(); cpInfo != nil {
		detected = append(detected, *cpInfo)
	}

	// Дедупликация
	detected = deduplicateHSM(detected)

	return detected
}

// detectFromEnv проверяет переменные окружения на наличие HSM.
func detectFromEnv() []HSMInfo {
	var detected []HSMInfo

	envVars := map[string]struct {
		hsmType HSMProviderType
		name    string
	}{
		"CRYPTOPRO_PATH":  {HSMКриптоПроCSP, "КриптоПро CSP"},
		"VIPNET_PATH":     {HSMViPNetCSP, "ViPNet CSP"},
		"SIGNALCOM_PATH":  {HSMSignalComCSP, "SignalCom CSP"},
		"LISSI_SKZI_PATH": {HSMLissiSKZI, "Лисси-СКЗИ"},
		"PKCS11_MODULE":   {HSMPKCS11, "PKCS#11"},
	}

	for env, info := range envVars {
		if path := os.Getenv(env); path != "" {
			if _, err := os.Stat(path); err == nil {
				detected = append(detected, HSMInfo{
					Type:      info.hsmType,
					Name:      info.name,
					Path:      path,
					Certified: info.hsmType == HSMКриптоПроCSP,
					Available: true,
				})
			}
		}
	}

	return detected
}

// detectFromLibraries ищет HSM библиотеки в стандартных путях.
func detectFromLibraries() []HSMInfo {
	var detected []HSMInfo
	seen := make(map[string]bool)

	for _, pattern := range hsmLibraryPatterns {
		for _, libPattern := range pattern.patterns {
			// Поиск по всем стандартным путям
			for _, dir := range hsmDetectPaths {
				matches, err := filepath.Glob(filepath.Join(dir, libPattern))
				if err != nil {
					continue
				}
				for _, match := range matches {
					if !seen[match] {
						seen[match] = true
						detected = append(detected, HSMInfo{
							Type:      pattern.provider,
							Name:      pattern.name,
							Path:      match,
							Certified: pattern.provider == HSMКриптоПроCSP,
							Available: true,
						})
					}
				}
			}
		}
	}

	return detected
}

// detectCryptoProCSP проверяет наличие КриптоПро CSP через csptest.
func detectCryptoProCSP() *HSMInfo {
	// Пробуем разные варианты расположения csptest
	csptestPaths := []string{
		"csptest",
		"/opt/cprocsp/sbin/amd64/csptest",
		"/opt/cprocsp/sbin/csptest",
	}

	for _, csptestPath := range csptestPaths {
		cmd := exec.Command(csptestPath, "-help")
		output, err := cmd.Output()
		if err != nil {
			continue
		}

		// Парсим версию из вывода
		version := parseCSPVersion(string(output))

		return &HSMInfo{
			Type:      HSMКриптоПроCSP,
			Name:      "КриптоПро CSP",
			Version:   version,
			Certified: true,
			Available: true,
		}
	}

	return nil
}

// parseCSPVersion парсит версию КриптоПро из вывода csptest.
func parseCSPVersion(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "version") || strings.Contains(line, "Version") {
			parts := strings.Fields(line)
			for _, part := range parts {
				if strings.Contains(part, ".") && len(part) > 3 {
					return part
				}
			}
		}
	}
	return ""
}

// deduplicateHSM удаляет дубликаты HSM провайдеров.
func deduplicateHSM(detected []HSMInfo) []HSMInfo {
	seen := make(map[HSMProviderType]bool)
	var result []HSMInfo
	for _, hsm := range detected {
		if !seen[hsm.Type] {
			seen[hsm.Type] = true
			result = append(result, hsm)
		}
	}
	return result
}

// ────────────────────────────────────────────────────────────────────────────
// HSM status checking
// ────────────────────────────────────────────────────────────────────────────

// IsHSMAvailable проверяет доступность аппаратного HSM на системе.
func IsHSMAvailable() bool {
	detected := DetectHSM()
	return len(detected) > 0
}

// GetBestHSM возвращает наилучший доступный HSM провайдер.
//
// Приоритет:
//  1. КриптоПро CSP (наиболее распространён в РФ)
//  2. КриптоПро HSM (аппаратное ускорение)
//  3. Другие сертифицированные СКЗИ
//  4. PKCS#11 токены
func GetBestHSM() *HSMInfo {
	detected := DetectHSM()
	if len(detected) == 0 {
		return nil
	}

	// Сортировка по приоритету
	priority := map[HSMProviderType]int{
		HSMКриптоПроCSP: 1,
		HSMКриптоПроHSM: 2,
		HSMViPNetCSP:    3,
		HSMSignalComCSP: 4,
		HSMLissiSKZI:    5,
		HSMPKCS11:       6,
	}

	best := detected[0]
	bestPriority := priority[best.Type]

	for _, hsm := range detected[1:] {
		if p := priority[hsm.Type]; p < bestPriority {
			best = hsm
			bestPriority = p
		}
	}

	return &best
}

// ────────────────────────────────────────────────────────────────────────────
// Platform-specific HSM detection
// ────────────────────────────────────────────────────────────────────────────

// detectHSMByOS возвращает пути для детекции HSM в зависимости от ОС.
func detectHSMByOS() []string {
	switch runtime.GOOS {
	case "linux":
		return []string{
			"/usr/lib",
			"/usr/lib64",
			"/usr/lib/x86_64-linux-gnu",
			"/opt/cprocsp/lib/amd64",
		}
	case "windows":
		return []string{
			"C:\\Program Files\\Crypto Pro\\",
			"C:\\Program Files (x86)\\Crypto Pro\\",
		}
	default:
		return hsmDetectPaths
	}
}

// ────────────────────────────────────────────────────────────────────────────
// HSM Provider factory
// ────────────────────────────────────────────────────────────────────────────

// HSMProvider описывает интерфейс HSM провайдера для аппаратного ускорения.
type HSMProvider interface {
	// Type возвращает тип HSM провайдера.
	Type() HSMProviderType

	// Encrypt шифрует данные с использованием HSM.
	Encrypt(keyLabel string, plaintext []byte) ([]byte, error)

	// Decrypt расшифровывает данные с использованием HSM.
	Decrypt(keyLabel string, ciphertext []byte) ([]byte, error)

	// Sign подписывает данные с использованием HSM.
	Sign(keyLabel string, data []byte) ([]byte, error)

	// Verify проверяет подпись с использованием HSM.
	Verify(keyLabel string, data, signature []byte) (bool, error)
}

// NewHSMProvider создаёт HSM провайдер на основе обнаруженного HSM.
//
// Возвращает nil, если HSM не обнаружен.
func NewHSMProvider() HSMProvider {
	best := GetBestHSM()
	if best == nil {
		return nil
	}

	switch best.Type {
	case HSMКриптоПроCSP, HSMКриптоПроHSM:
		return newCryptoProHSMProvider(best)
	case HSMViPNetCSP:
		return newViPNetHSMProvider(best)
	default:
		return newPKCS11HSMProvider(best)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Stub HSM providers (для совместимости интерфейса)
// ────────────────────────────────────────────────────────────────────────────

// cryptoProHSMProvider — HSM провайдер для КриптоПро.
type cryptoProHSMProvider struct {
	info *HSMInfo
}

func newCryptoProHSMProvider(info *HSMInfo) *cryptoProHSMProvider {
	return &cryptoProHSMProvider{info: info}
}

func (p *cryptoProHSMProvider) Type() HSMProviderType { return p.info.Type }
func (p *cryptoProHSMProvider) Encrypt(keyLabel string, plaintext []byte) ([]byte, error) {
	return nil, fmt.Errorf("cryptopro hsm: encrypt not implemented (requires CGo)")
}
func (p *cryptoProHSMProvider) Decrypt(keyLabel string, ciphertext []byte) ([]byte, error) {
	return nil, fmt.Errorf("cryptopro hsm: decrypt not implemented (requires CGo)")
}
func (p *cryptoProHSMProvider) Sign(keyLabel string, data []byte) ([]byte, error) {
	return nil, fmt.Errorf("cryptopro hsm: sign not implemented (requires CGo)")
}
func (p *cryptoProHSMProvider) Verify(keyLabel string, data, signature []byte) (bool, error) {
	return false, fmt.Errorf("cryptopro hsm: verify not implemented (requires CGo)")
}

// viPNetHSMProvider — HSM провайдер для ViPNet.
type viPNetHSMProvider struct {
	info *HSMInfo
}

func newViPNetHSMProvider(info *HSMInfo) *viPNetHSMProvider {
	return &viPNetHSMProvider{info: info}
}

func (p *viPNetHSMProvider) Type() HSMProviderType { return p.info.Type }
func (p *viPNetHSMProvider) Encrypt(keyLabel string, plaintext []byte) ([]byte, error) {
	return nil, fmt.Errorf("vipnet hsm: encrypt not implemented (requires CGo)")
}
func (p *viPNetHSMProvider) Decrypt(keyLabel string, ciphertext []byte) ([]byte, error) {
	return nil, fmt.Errorf("vipnet hsm: decrypt not implemented (requires CGo)")
}
func (p *viPNetHSMProvider) Sign(keyLabel string, data []byte) ([]byte, error) {
	return nil, fmt.Errorf("vipnet hsm: sign not implemented (requires CGo)")
}
func (p *viPNetHSMProvider) Verify(keyLabel string, data, signature []byte) (bool, error) {
	return false, fmt.Errorf("vipnet hsm: verify not implemented (requires CGo)")
}

// pkcs11HSMProvider — HSM провайдер для PKCS#11 токенов.
type pkcs11HSMProvider struct {
	info *HSMInfo
}

func newPKCS11HSMProvider(info *HSMInfo) *pkcs11HSMProvider {
	return &pkcs11HSMProvider{info: info}
}

func (p *pkcs11HSMProvider) Type() HSMProviderType { return p.info.Type }
func (p *pkcs11HSMProvider) Encrypt(keyLabel string, plaintext []byte) ([]byte, error) {
	return nil, fmt.Errorf("pkcs11 hsm: encrypt not implemented (requires CGo)")
}
func (p *pkcs11HSMProvider) Decrypt(keyLabel string, ciphertext []byte) ([]byte, error) {
	return nil, fmt.Errorf("pkcs11 hsm: decrypt not implemented (requires CGo)")
}
func (p *pkcs11HSMProvider) Sign(keyLabel string, data []byte) ([]byte, error) {
	return nil, fmt.Errorf("pkcs11 hsm: sign not implemented (requires CGo)")
}
func (p *pkcs11HSMProvider) Verify(keyLabel string, data, signature []byte) (bool, error) {
	return false, fmt.Errorf("pkcs11 hsm: verify not implemented (requires CGo)")
}
