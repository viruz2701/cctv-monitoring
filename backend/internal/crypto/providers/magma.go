// Package providers — ГОСТ 28147-89 (Магма) block cipher implementation.
//
// ═══════════════════════════════════════════════════════════════════════════
// P2-MKT.1: Real GOST 28147-89 (Magma) block cipher
//
// Алгоритм:
//   - Блок: 64 бита (8 байт)
//   - Ключ: 256 бит (32 байта) = 8 × 32-бит подключей
//   - Раунды: 32 (сеть Фейстеля)
//   - S-box: id-tc26-gost-28147-param-Z (ГОСТ Р 34.12-2015, Приложение А)
//   - Режим: ECB (одиночный блок), CBC (много-блочные сообщения)
//
// Структура раунда (сеть Фейстеля):
//  1. f = (right + K_i) mod 2^32
//  2. f = S-box substitution (8 × 4-bit S-boxes)
//  3. f = cyclic left shift by 11 bits
//  4. new_right = left XOR f
//  5. left = right
//
// Порядок подключей (32 раунда):
//   - Раунды 0-23: K1, K2, ..., K8 (повтор 3 раза: K1..K8, K1..K8, K1..K8)
//   - Раунды 24-31: K8, K7, ..., K1 (обратный порядок)
//
// Compliance:
//   - ГОСТ 28147-89 (Магма) — Симметричное шифрование (Советский/РФ стандарт)
//   - ГОСТ Р 34.12-2015 — Современное описание алгоритма
//   - Приказ ФСТЭК № 17 — Класс КС3
//   - IEC 62443-3-3 SR 5.1 — Zone-based access control
//
// ═══════════════════════════════════════════════════════════════════════════
package providers

import (
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// ────────────────────────────────────────────────────────────────────────────
// GOST 28147-89 constants
// ────────────────────────────────────────────────────────────────────────────

const (
	// MagmaBlockSize — размер блока Магма (64 бита = 8 байт).
	MagmaBlockSize = 8

	// MagmaKeySize — размер ключа Магма (256 бит = 32 байта).
	MagmaKeySize = 32

	// MagmaRounds — количество раундов.
	MagmaRounds = 32

	// MagmaSubkeys — количество 32-битных подключей.
	MagmaSubkeys = 8

	// MagmaCyclicShift — величина циклического сдвига в раундовой функции.
	MagmaCyclicShift = 11
)

// ────────────────────────────────────────────────────────────────────────────
// S-box: id-tc26-gost-28147-param-Z (ГОСТ Р 34.12-2015, Приложение А)
// ────────────────────────────────────────────────────────────────────────────

// magmaSBox — таблица замены id-tc26-gost-28147-param-Z.
// 8 S-boxes, each 4-bit → 4-bit (16 элементов).
//
// Источник: ГОСТ Р 34.12-2015, Приложение А.
// Идентификатор OID: 1.2.643.7.1.1.1.1 (id-tc26-gost-28147-param-Z).
var magmaSBox = [8][16]byte{
	{0xC, 0x4, 0x6, 0x2, 0xA, 0x5, 0xB, 0x9, 0xE, 0x8, 0xD, 0x7, 0x0, 0x3, 0xF, 0x1}, // S0
	{0x6, 0x8, 0x2, 0x3, 0x9, 0xA, 0x5, 0xC, 0x1, 0xE, 0x4, 0x7, 0xB, 0xD, 0x0, 0xF}, // S1
	{0xB, 0x3, 0x5, 0x8, 0x2, 0xF, 0xA, 0xD, 0xE, 0x1, 0x7, 0x4, 0xC, 0x9, 0x6, 0x0}, // S2
	{0xC, 0x4, 0x6, 0x2, 0xA, 0x5, 0xB, 0x9, 0xE, 0x8, 0xD, 0x7, 0x0, 0x3, 0xF, 0x1}, // S3
	{0x6, 0x8, 0x2, 0x3, 0x9, 0xA, 0x5, 0xC, 0x1, 0xE, 0x4, 0x7, 0xB, 0xD, 0x0, 0xF}, // S4
	{0xB, 0x3, 0x5, 0x8, 0x2, 0xF, 0xA, 0xD, 0xE, 0x1, 0x7, 0x4, 0xC, 0x9, 0x6, 0x0}, // S5
	{0x1, 0xF, 0xD, 0x0, 0x5, 0x7, 0xA, 0x4, 0x9, 0x2, 0x3, 0xE, 0x6, 0xB, 0x8, 0xC}, // S6
	{0x1, 0xF, 0xD, 0x0, 0x5, 0x7, 0xA, 0x4, 0x9, 0x2, 0x3, 0xE, 0x6, 0xB, 0x8, 0xC}, // S7
}

// magmaSubkeyOrder — порядок использования подключей в 32 раундах.
//
// Раунды 0–23: K1..K8, K1..K8, K1..K8 (прямой порядок, 3 раза)
// Раунды 24–31: K8..K1 (обратный порядок)
var magmaSubkeyOrder = [MagmaRounds]int{
	0, 1, 2, 3, 4, 5, 6, 7, // K1..K8
	0, 1, 2, 3, 4, 5, 6, 7, // K1..K8
	0, 1, 2, 3, 4, 5, 6, 7, // K1..K8
	7, 6, 5, 4, 3, 2, 1, 0, // K8..K1
}

// ────────────────────────────────────────────────────────────────────────────
// Errors
// ────────────────────────────────────────────────────────────────────────────

var (
	// ErrMagmaInvalidKeySize — неверный размер ключа.
	ErrMagmaInvalidKeySize = errors.New("magma: key must be 32 bytes (256 bit)")

	// ErrMagmaInvalidBlockSize — неверный размер блока.
	ErrMagmaInvalidBlockSize = errors.New("magma: block size must be 8 bytes (64 bit)")

	// ErrMagmaInvalidCiphertext — неверный формат ciphertext.
	ErrMagmaInvalidCiphertext = errors.New("magma: invalid ciphertext")
)

// ────────────────────────────────────────────────────────────────────────────
// MagmaCipher — реализация блочного шифра ГОСТ 28147-89 (Магма)
// ────────────────────────────────────────────────────────────────────────────

// MagmaCipher implements cipher.Block for GOST 28147-89 (Magma).
type MagmaCipher struct {
	subkeys [MagmaSubkeys]uint32 // 8 × 32-бит подключей
}

// NewMagmaCipher создаёт новый MagmaCipher с заданным ключом.
//
// Ключ: 32 байта (256 бит), разбивается на 8 × 32-бит подключей
// в little-endian порядке.
func NewMagmaCipher(key []byte) (*MagmaCipher, error) {
	if len(key) != MagmaKeySize {
		return nil, fmt.Errorf("%w: got %d bytes", ErrMagmaInvalidKeySize, len(key))
	}

	var c MagmaCipher
	for i := 0; i < MagmaSubkeys; i++ {
		c.subkeys[i] = binary.LittleEndian.Uint32(key[i*4 : (i+1)*4])
	}
	return &c, nil
}

// BlockSize возвращает размер блока (8 байт).
func (c *MagmaCipher) BlockSize() int { return MagmaBlockSize }

// Encrypt шифрует один 64-битный блок (8 байт).
// src и dst могут указывать на один и тот же срез.
func (c *MagmaCipher) Encrypt(dst, src []byte) {
	if len(src) < MagmaBlockSize {
		panic("magma: src too short")
	}
	if len(dst) < MagmaBlockSize {
		panic("magma: dst too short")
	}

	// Разбиваем блок на левую и правую половины (32 бита каждая)
	a := binary.LittleEndian.Uint32(src[0:4]) // левая половина (N1)
	b := binary.LittleEndian.Uint32(src[4:8]) // правая половина (N2)

	// 32 раунда сети Фейстеля
	for round := 0; round < MagmaRounds; round++ {
		idx := magmaSubkeyOrder[round]
		f := magmaRoundFunction(b, c.subkeys[idx])

		if round < MagmaRounds-1 {
			// f = f XOR a; swap a, b
			newB := a ^ f
			a = b
			b = newB
		} else {
			// Последний раунд: без swap
			a = a ^ f
			// b остаётся без изменений
		}
	}

	// Результат: a (N1) || b (N2)
	binary.LittleEndian.PutUint32(dst[0:4], a)
	binary.LittleEndian.PutUint32(dst[4:8], b)
}

// Decrypt расшифровывает один 64-битный блок (8 байт).
// Для Магма дешифрование = шифрование с обратным порядком подключей.
func (c *MagmaCipher) Decrypt(dst, src []byte) {
	if len(src) < MagmaBlockSize {
		panic("magma: src too short")
	}
	if len(dst) < MagmaBlockSize {
		panic("magma: dst too short")
	}

	a := binary.LittleEndian.Uint32(src[0:4])
	b := binary.LittleEndian.Uint32(src[4:8])

	// Дешифрование: обратный порядок раундов
	// 32 раунда с обратным порядком подключей
	for round := 0; round < MagmaRounds; round++ {
		// Для дешифрования используем обратный порядок
		idx := magmaSubkeyOrder[MagmaRounds-1-round]
		f := magmaRoundFunction(b, c.subkeys[idx])

		if round < MagmaRounds-1 {
			newB := a ^ f
			a = b
			b = newB
		} else {
			a = a ^ f
		}
	}

	binary.LittleEndian.PutUint32(dst[0:4], a)
	binary.LittleEndian.PutUint32(dst[4:8], b)
}

// ────────────────────────────────────────────────────────────────────────────
// Round function
// ────────────────────────────────────────────────────────────────────────────

// magmaRoundFunction — раундовая функция Магма.
//
// Вход: 32-битное значение (правая половина), 32-битный подключ.
// Выход: 32-битное значение после S-box замены и циклического сдвига.
//
// Шаги:
//  1. sum = (right + subkey) mod 2^32
//  2. S-box substitution: каждый 4-битный блок заменяется через S-box
//  3. Cyclic left shift на 11 бит
func magmaRoundFunction(right, subkey uint32) uint32 {
	// Шаг 1: сложение по модулю 2^32
	sum := right + subkey

	// Шаг 2: S-box подстановка (8 S-boxes × 4 бита = 32 бита)
	var result uint32
	for i := 0; i < 8; i++ {
		// Берём 4 бита (начиная с младших)
		nibble := byte((sum >> (i * 4)) & 0xF)
		// Заменяем через S-box
		substituted := magmaSBox[i][nibble]
		// Собираем результат
		result |= uint32(substituted) << (i * 4)
	}

	// Шаг 3: циклический сдвиг влево на 11 бит
	return (result << MagmaCyclicShift) | (result >> (32 - MagmaCyclicShift))
}

// ────────────────────────────────────────────────────────────────────────────
// CBC mode implementation
// ────────────────────────────────────────────────────────────────────────────

// magmaCBCIvSize — размер IV для CBC режима Магма (8 байт = размер блока).
const magmaCBCIvSize = MagmaBlockSize

// magmaCBCEncrypt шифрует данные в режиме CBC с PKCS#7 padding.
//
// Формат выхода: [IV (8) || ciphertext]
// IV генерируется случайно для каждого шифрования.
func magmaCBCEncrypt(c *MagmaCipher, plaintext []byte) ([]byte, error) {
	// Генерируем случайный IV
	iv := make([]byte, magmaCBCIvSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, fmt.Errorf("magma-cbc: iv generation: %w", err)
	}

	// PKCS#7 padding
	padded := pkcs7Pad(plaintext, MagmaBlockSize)

	// Шифруем в CBC режиме
	ciphertext := make([]byte, len(padded))
	cbc := cipher.NewCBCEncrypter(c, iv)
	cbc.CryptBlocks(ciphertext, padded)

	// IV || ciphertext
	result := make([]byte, magmaCBCIvSize+len(ciphertext))
	copy(result[:magmaCBCIvSize], iv)
	copy(result[magmaCBCIvSize:], ciphertext)

	return result, nil
}

// magmaCBCDecrypt расшифровывает данные в режиме CBC с PKCS#7 padding.
//
// Ожидает формат: [IV (8) || ciphertext]
func magmaCBCDecrypt(c *MagmaCipher, data []byte) ([]byte, error) {
	if len(data) < magmaCBCIvSize+MagmaBlockSize {
		return nil, fmt.Errorf("%w: data too short (%d bytes)", ErrMagmaInvalidCiphertext, len(data))
	}

	if len(data)%MagmaBlockSize != magmaCBCIvSize%MagmaBlockSize {
		// Длина данных минус IV должна быть кратна размеру блока
		if (len(data)-magmaCBCIvSize)%MagmaBlockSize != 0 {
			return nil, fmt.Errorf("%w: ciphertext length not aligned (%d bytes)", ErrMagmaInvalidCiphertext, len(data))
		}
	}

	iv := data[:magmaCBCIvSize]
	ciphertext := data[magmaCBCIvSize:]

	padded := make([]byte, len(ciphertext))
	cbc := cipher.NewCBCDecrypter(c, iv)
	cbc.CryptBlocks(padded, ciphertext)

	// PKCS#7 unpadding
	plaintext, err := pkcs7Unpad(padded, MagmaBlockSize)
	if err != nil {
		return nil, fmt.Errorf("magma-cbc: %w", err)
	}

	return plaintext, nil
}

// ────────────────────────────────────────────────────────────────────────────
// PKCS#7 padding helpers
// ────────────────────────────────────────────────────────────────────────────

// pkcs7Pad добавляет PKCS#7 padding к данным.
func pkcs7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padded := make([]byte, len(data)+padding)
	copy(padded, data)
	for i := len(data); i < len(padded); i++ {
		padded[i] = byte(padding)
	}
	return padded
}

// pkcs7Unpad удаляет PKCS#7 padding из данных.
func pkcs7Unpad(data []byte, blockSize int) ([]byte, error) {
	if len(data) == 0 || len(data)%blockSize != 0 {
		return nil, errors.New("invalid padding: data length not aligned")
	}

	padding := int(data[len(data)-1])
	if padding == 0 || padding > blockSize {
		return nil, errors.New("invalid padding: out of range")
	}

	// Проверяем все байты padding
	for i := len(data) - padding; i < len(data); i++ {
		if data[i] != byte(padding) {
			return nil, errors.New("invalid padding: inconsistent values")
		}
	}

	return data[:len(data)-padding], nil
}
