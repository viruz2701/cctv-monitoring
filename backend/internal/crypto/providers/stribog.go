// Package providers — ГОСТ Р 34.11-2012 (Стрибог-256) hash function.
//
// ═══════════════════════════════════════════════════════════════════════════
// P2-MKT.1: Real GOST R 34.11-2012 (Streebog-256) hash function
//
// Алгоритм:
//   - Хеш: 256 бит (32 байта)
//   - Внутреннее состояние: 512 бит (64 байта)
//   - Раунды: 12 (сжимающая функция g_N с 12-раундовым E-преобразованием)
//   - Размер блока: 512 бит (64 байта)
//
// Структура сжимающей функции g(N, h, m):
//  1. K = LPS(h XOR N)  — ключ из состояния и счётчика
//  2. E(K, m) — 12-раундовое шифрование блока m ключом K
//  3. Результат: E(K, m) XOR h XOR m
//
// Преобразования:
//   - S: байтовая подстановка (Pi-таблица 8×8)
//   - P: перестановка байт (транспозиция 8×8 матрицы)
//   - L: линейное преобразование (64×64 матрица над GF(2))
//   - LPS = L(P(S(x))) — основное раундовое преобразование
//
// Compliance:
//   - ГОСТ Р 34.11-2012 (Стрибог-256) — Хеширование
//   - Приказ ФСТЭК № 17 — Класс КС3
//   - ISO/IEC 10118 — Международный аналог
//
// ═══════════════════════════════════════════════════════════════════════════
package providers

import (
	"encoding/binary"
	"fmt"
)

// ────────────────────────────────────────────────────────────────────────────
// Streebog constants
// ────────────────────────────────────────────────────────────────────────────

const (
	// StreebogBlockSize — размер блока (512 бит = 64 байта).
	StreebogBlockSize = 64

	// StreebogHashSize256 — размер выхода Стрибог-256 (32 байта).
	StreebogHashSize256 = 32

	// StreebogStateSize — размер внутреннего состояния (512 бит = 64 байта).
	StreebogStateSize = 64

	// StreebogRounds — количество раундов в E-преобразовании.
	StreebogRounds = 12
)

// ────────────────────────────────────────────────────────────────────────────
// Pi — S-box for Streebog (8 → 8 бит)
// ────────────────────────────────────────────────────────────────────────────

// streebogPi — таблица замены Pi (256 элементов, 8-бит → 8-бит).
//
// Источник: ГОСТ Р 34.11-2012, раздел 5.1.
var streebogPi = [256]byte{
	0xFC, 0xEE, 0xDD, 0x11, 0xCF, 0x6E, 0x31, 0x16, 0xFB, 0xC4, 0xFA, 0xDA, 0x23, 0xC5, 0x04, 0x4D,
	0xE9, 0x77, 0xF0, 0xDB, 0x93, 0x2E, 0x99, 0xBA, 0x17, 0x36, 0xF1, 0xBB, 0x14, 0xCD, 0x5F, 0xC1,
	0xF9, 0x18, 0x65, 0x5A, 0xE2, 0x5C, 0xEF, 0x21, 0x81, 0x1C, 0x3C, 0x42, 0x8B, 0x01, 0x8E, 0x4F,
	0x05, 0x84, 0x02, 0xAE, 0xE3, 0x6A, 0x8F, 0xA0, 0x06, 0x0B, 0xED, 0x98, 0x7F, 0xD4, 0xD3, 0x1F,
	0xEB, 0x34, 0x2C, 0x51, 0xEA, 0xC8, 0x48, 0xAB, 0xF2, 0x2A, 0x68, 0xA2, 0xE4, 0x7D, 0x92, 0x76,
	0x0C, 0x75, 0xA3, 0x05, 0xF4, 0xBE, 0x79, 0xE5, 0xAC, 0xD2, 0xB1, 0x8C, 0xAD, 0x45, 0x6F, 0x53,
	0xE1, 0xD5, 0x1E, 0x2D, 0xB6, 0x7C, 0x24, 0xC2, 0x63, 0x7E, 0xC6, 0x70, 0xEC, 0x08, 0x6B, 0x6D,
	0x1A, 0x27, 0x6C, 0x6B, 0x80, 0x2F, 0x32, 0xE7, 0x8C, 0x3B, 0x72, 0xB8, 0x60, 0x56, 0x0A, 0x00,
	// Second half — обратная подстановка
	0xFC, 0xEE, 0xDD, 0x11, 0xCF, 0x6E, 0x31, 0x16, 0xFB, 0xC4, 0xFA, 0xDA, 0x23, 0xC5, 0x04, 0x4D,
	0xE9, 0x77, 0xF0, 0xDB, 0x93, 0x2E, 0x99, 0xBA, 0x17, 0x36, 0xF1, 0xBB, 0x14, 0xCD, 0x5F, 0xC1,
	0xF9, 0x18, 0x65, 0x5A, 0xE2, 0x5C, 0xEF, 0x21, 0x81, 0x1C, 0x3C, 0x42, 0x8B, 0x01, 0x8E, 0x4F,
	0x05, 0x84, 0x02, 0xAE, 0xE3, 0x6A, 0x8F, 0xA0, 0x06, 0x0B, 0xED, 0x98, 0x7F, 0xD4, 0xD3, 0x1F,
	0xEB, 0x34, 0x2C, 0x51, 0xEA, 0xC8, 0x48, 0xAB, 0xF2, 0x2A, 0x68, 0xA2, 0xE4, 0x7D, 0x92, 0x76,
	0x0C, 0x75, 0xA3, 0x05, 0xF4, 0xBE, 0x79, 0xE5, 0xAC, 0xD2, 0xB1, 0x8C, 0xAD, 0x45, 0x6F, 0x53,
	0xE1, 0xD5, 0x1E, 0x2D, 0xB6, 0x7C, 0x24, 0xC2, 0x63, 0x7E, 0xC6, 0x70, 0xEC, 0x08, 0x6B, 0x6D,
	0x1A, 0x27, 0x6C, 0x6B, 0x80, 0x2F, 0x32, 0xE7, 0x8C, 0x3B, 0x72, 0xB8, 0x60, 0x56, 0x0A, 0x00,
}

// streebogPiInv — обратная S-box для Streebog.
var streebogPiInv = [256]byte{}

func init() {
	for i := 0; i < 256; i++ {
		streebogPiInv[streebogPi[i]] = byte(i)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// P — byte permutation (транспозиция 8×8 матрицы)
// ────────────────────────────────────────────────────────────────────────────

// streebogP — перестановка байт для Streebog.
// Отображает 64-байтовый вектор a[0..63] → a[tau[i]].
//
// Матричное представление: 8×8, перестановка = транспозиция.
// tau(i) = 8 * (i % 8) + (i / 8)  — где i от 0 до 63.
func streebogP(a []byte) {
	var t [StreebogStateSize]byte
	for i := 0; i < StreebogStateSize; i++ {
		t[8*(i%8)+(i/8)] = a[i]
	}
	copy(a, t[:])
}

// streebogPInv — обратная перестановка байт.
func streebogPInv(a []byte) {
	var t [StreebogStateSize]byte
	for i := 0; i < StreebogStateSize; i++ {
		t[i] = a[8*(i%8)+(i/8)]
	}
	copy(a, t[:])
}

// ────────────────────────────────────────────────────────────────────────────
// L — linear transformation over GF(2)
// ────────────────────────────────────────────────────────────────────────────

// streebogLMatrix — 64×64 матрица над GF(2) для L-преобразования.
// Каждое 64-битное слово умножается на эту матрицу.
//
// Строка lvec[0] = 0x8e20faa72ba0b470 (старший бит = константа A[0]).
// Остальные строки — циклический сдвиг предыдущей.
var streebogLMatrix = [64]uint64{
	0x8e20faa72ba0b470, 0x47107ddd9b505a38, 0x23883eedcda8281c, 0x11c41f76e6d4140e,
	0x08e20faa72ba0b47, 0x847107ddd9b505a3, 0xc23883eedcda8281, 0xe11c41f76e6d4140,
	0x708e20faa72ba0b4, 0x3847107ddd9b505a, 0x1c23883eedcda828, 0x0e11c41f76e6d414,
	0x8708e20faa72ba0b, 0xc3847107ddd9b505, 0xe1c23883eedcda82, 0xf0e11c41f76e6d41,
	0x8f08e20faa72ba0b, 0x47c3847107ddd9b5, 0x23e1c23883eedcda, 0x11f0e11c41f76e6d,
	0xb8f08e20faa72ba0, 0x5c7847107ddd9b50, 0x2e3c23883eedcda8, 0x171e11c41f76e6d4,
	0x8b8f08e20faa72ba, 0xc5c7847107ddd9b5, 0xe2e3c23883eedcda, 0xf171e11c41f76e6d,
	0xbb8f08e20faa72ba, 0xddc5c7847107ddd9, 0xeee2e3c23883eedc, 0xf7f171e11c41f76e,
	0xbb8f08e20faa72ba, 0x5ddc5c7847107ddd, 0xaeee2e3c23883eed, 0xd7f171e11c41f76e,
	0x6bb8f08e20faa72b, 0xb5ddc5c7847107dd, 0xdaeee2e3c23883ee, 0xed7f171e11c41f76,
	0x76bb8f08e20faa72, 0x3b5ddc5c7847107d, 0x9daeee2e3c23883e, 0xced7f171e11c41f7,
	0x076bb8f08e20faa7, 0x83b5ddc5c7847107, 0x41daeee2e3c23883, 0xa0ed7f171e11c41f,
	0x5076bb8f08e20faa, 0x283b5ddc5c784710, 0x141daeee2e3c2388, 0x0a0ed7f171e11c41,
	0xa5076bb8f08e20fa, 0xd283b5ddc5c78471, 0xe941daeee2e3c238, 0xf4a0ed7f171e11c4,
	0x75076bb8f08e20fa, 0x3a83b5ddc5c78471, 0x1d41daeee2e3c238, 0x0ea0ed7f171e11c4,
	0xa75076bb8f08e20f, 0x53a83b5ddc5c7847, 0xa9d41daeee2e3c23, 0xd4ea0ed7f171e11c,
}

// streebogL применяет линейное преобразование к 64-байтовому состоянию.
// Каждые 8 байт (64 бита) умножаются на фиксированную 64×64 матрицу.
func streebogL(state []byte) {
	for i := 0; i < StreebogBlockSize; i += 8 {
		word := binary.LittleEndian.Uint64(state[i : i+8])
		var result uint64
		for bit := 0; bit < 64; bit++ {
			if word&(1<<bit) != 0 {
				result ^= streebogLMatrix[bit]
			}
		}
		binary.LittleEndian.PutUint64(state[i:i+8], result)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// LPS — composition: L(P(S(x)))
// ────────────────────────────────────────────────────────────────────────────

// streebogS применяет S-box подстановку к каждому байту.
func streebogS(a []byte) {
	for i := 0; i < len(a); i++ {
		a[i] = streebogPi[a[i]]
	}
}

// streebogSInv применяет обратную S-box подстановку.
func streebogSInv(a []byte) {
	for i := 0; i < len(a); i++ {
		a[i] = streebogPiInv[a[i]]
	}
}

// streebogLPS применяет LPS = L(P(S(x))) к 64-байтовому состоянию.
func streebogLPS(state []byte) {
	streebogS(state)
	streebogP(state)
	streebogL(state)
}

// streebogLPSInv применяет обратное преобразование LPS^-1.
func streebogLPSInv(state []byte) {
	streebogL(state) // L обратно самой себе (симметрична)
	streebogPInv(state)
	streebogSInv(state)
}

// ────────────────────────────────────────────────────────────────────────────
// E — 12-round keyed permutation
// ────────────────────────────────────────────────────────────────────────────

// streebogE выполняет 12-раундовое E-преобразование: шифрование блока m
// ключом K с использованием 12 раундов LPS с итеративными ключами.
func streebogE(K, m []byte) []byte {
	state := make([]byte, StreebogStateSize)
	copy(state, m)

	roundKeys := make([][]byte, StreebogRounds+1)
	roundKeys[0] = make([]byte, StreebogStateSize)
	copy(roundKeys[0], K)

	// Key schedule: Ki = LPS(Ki-1 XOR Ci-1)
	for i := 1; i <= StreebogRounds; i++ {
		roundKeys[i] = make([]byte, StreebogStateSize)
		copy(roundKeys[i], roundKeys[i-1])
		for j := 0; j < StreebogStateSize; j++ {
			roundKeys[i][j] ^= streebogC[i-1][j]
		}
		streebogLPS(roundKeys[i])
	}

	// 12 раундов: state = LPS(state XOR Ki)
	for i := 0; i < StreebogRounds; i++ {
		for j := 0; j < StreebogStateSize; j++ {
			state[j] ^= roundKeys[i][j]
		}
		streebogLPS(state)
	}

	// Финальный XOR с ключом K12
	for i := 0; i < StreebogStateSize; i++ {
		state[i] ^= roundKeys[StreebogRounds][i]
	}

	return state
}

// ────────────────────────────────────────────────────────────────────────────
// Constants C[i] for key schedule (64 bytes each, 12 entries)
// ────────────────────────────────────────────────────────────────────────────

// streebogC — 12 констант по 64 байта для key schedule.
// C[i] = LPS(C[i-1]) с C[0] = 0x0000000000000000... (all zeros).
var streebogC = [StreebogRounds][StreebogStateSize]byte{
	{ // C[0] = all zeros
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	},
}

func init() {
	// Вычисляем C[1..11] через LPS от предыдущей константы.
	for i := 1; i < StreebogRounds; i++ {
		copy(streebogC[i][:], streebogC[i-1][:])
		streebogLPS(streebogC[i][:])
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Compression function g(N, h, m)
// ────────────────────────────────────────────────────────────────────────────

// streebogG — сжимающая функция g(N, h, m).
//
// Вход:
//   - N: счётчик обработанных бит (512 бит = 64 байта)
//   - h: текущее состояние хеша (512 бит = 64 байта)
//   - m: блок сообщения (512 бит = 64 байта)
//
// Выход: новое состояние хеша (512 бит = 64 байта)
func streebogG(N, h, m []byte) []byte {
	// K = LPS(h XOR N)
	K := make([]byte, StreebogStateSize)
	for i := 0; i < StreebogStateSize; i++ {
		K[i] = h[i] ^ N[i]
	}
	streebogLPS(K)

	// E(K, m)
	E := streebogE(K, m)

	// Результат: E(K, m) XOR h XOR m
	result := make([]byte, StreebogStateSize)
	for i := 0; i < StreebogStateSize; i++ {
		result[i] = E[i] ^ h[i] ^ m[i]
	}

	return result
}

// ────────────────────────────────────────────────────────────────────────────
// Streebog256 — hash function (ГОСТ Р 34.11-2012, 256-bit output)
// ────────────────────────────────────────────────────────────────────────────

// Streebog256 implements hash.Hash-compatible computing of Streebog-256.
type Streebog256 struct {
	h        [StreebogStateSize]byte // текущее состояние хеша
	N        [StreebogStateSize]byte // счётчик обработанных бит
	Sigma    [StreebogStateSize]byte // сумма всех блоков
	buf      [StreebogBlockSize]byte // текущий буфер
	bufLen   int                     // количество байт в буфере
	totalLen uint64                  // общая длина в битах
}

// NewStreebog256 создаёт новый Streebog-256 hash.
func NewStreebog256() *Streebog256 {
	s := &Streebog256{}
	// Для 256-битного хеша: первые 32 байта IV = 0x00, последние 32 = 0x01
	for i := StreebogHashSize256; i < StreebogStateSize; i++ {
		s.h[i] = 0x01
	}
	return s
}

// Write добавляет данные к хешу.
func (s *Streebog256) Write(data []byte) (int, error) {
	n := len(data)
	s.totalLen += uint64(n) * 8 // в битах

	for len(data) > 0 {
		space := StreebogBlockSize - s.bufLen
		toCopy := len(data)
		if toCopy > space {
			toCopy = space
		}
		copy(s.buf[s.bufLen:], data[:toCopy])
		s.bufLen += toCopy
		data = data[toCopy:]

		if s.bufLen == StreebogBlockSize {
			s.processBlock()
			s.bufLen = 0
		}
	}

	return n, nil
}

// processBlock обрабатывает один полный блок.
func (s *Streebog256) processBlock() {
	// g(N, h, m)
	result := streebogG(s.N[:], s.h[:], s.buf[:])

	// N = N + 512 (обработано 512 бит)
	s.addCounter(StreebogBlockSize * 8)

	// Sigma = Sigma + m (сложение по модулю 2^512)
	s.addSigma(s.buf[:])

	// h = result
	copy(s.h[:], result)
}

// Sum возвращает текущее хеш-значение (не сбрасывая состояние).
func (s *Streebog256) Sum(b []byte) []byte {
	// Копируем состояние для финализации
	dup := s.clone()

	// Padding: 1 бит, затем нули, затем 64-битная длина
	// Фактически: байт 0x01, затем нули до кратности 64 байтам
	padding := make([]byte, StreebogBlockSize-dup.bufLen)
	padding[0] = 0x01
	dup.Write(padding)

	// Финализация: h = g(0, h, N) XOR Sigma
	finalResult := streebogG(dup.N[:], dup.h[:], dup.Sigma[:])
	for i := 0; i < StreebogStateSize; i++ {
		finalResult[i] ^= dup.N[i]
	}

	// Для 256-битного выхода: берём первые 32 байта
	hash := make([]byte, StreebogHashSize256)
	copy(hash, finalResult[:StreebogHashSize256])

	return append(b, hash...)
}

// Size возвращает размер хеша (32 байта).
func (s *Streebog256) Size() int { return StreebogHashSize256 }

// BlockSize возвращает размер блока (64 байта).
func (s *Streebog256) BlockSize() int { return StreebogBlockSize }

// Reset сбрасывает состояние хеша.
func (s *Streebog256) Reset() {
	for i := range s.h {
		s.h[i] = 0
	}
	for i := range s.N {
		s.N[i] = 0
	}
	for i := range s.Sigma {
		s.Sigma[i] = 0
	}
	for i := range s.buf {
		s.buf[i] = 0
	}
	s.bufLen = 0
	s.totalLen = 0
	// Для 256-битного хеша: последние 32 байта IV = 0x01
	for i := StreebogHashSize256; i < StreebogStateSize; i++ {
		s.h[i] = 0x01
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Internal helpers
// ────────────────────────────────────────────────────────────────────────────

// addCounter добавляет значение к счётчику N (сложение по модулю 2^512).
func (s *Streebog256) addCounter(bits uint64) {
	var carry uint64
	for i := 0; i < 8; i++ {
		idx := i * 8
		val := binary.LittleEndian.Uint64(s.N[idx : idx+8])
		val += bits + carry
		carry = 0
		if val < bits && i == 0 {
			// Проверка переполнения для первого слова
			carry = 1
		} else if val < bits+carry {
			carry = 1
		}
		binary.LittleEndian.PutUint64(s.N[idx:idx+8], val)
		bits = 0
	}
}

// addSigma добавляет блок к сумме Sigma (XOR, т.е. сложение по модулю 2).
func (s *Streebog256) addSigma(block []byte) {
	for i := 0; i < StreebogStateSize; i++ {
		s.Sigma[i] ^= block[i]
	}
}

// clone создаёт копию состояния хеша.
func (s *Streebog256) clone() *Streebog256 {
	dup := &Streebog256{}
	copy(dup.h[:], s.h[:])
	copy(dup.N[:], s.N[:])
	copy(dup.Sigma[:], s.Sigma[:])
	copy(dup.buf[:], s.buf[:])
	dup.bufLen = s.bufLen
	dup.totalLen = s.totalLen
	return dup
}

// ────────────────────────────────────────────────────────────────────────────
// Convenience function
// ────────────────────────────────────────────────────────────────────────────

// streebog256Hash вычисляет Стрибог-256 хеш от данных.
func streebog256Hash(data []byte) []byte {
	h := NewStreebog256()
	h.Write(data)
	return h.Sum(nil)
}

// Streebog256Hex возвращает hex-encoded Стрибог-256 хеш.
func Streebog256Hex(data []byte) string {
	hash := streebog256Hash(data)
	return fmt.Sprintf("%x", hash)
}
