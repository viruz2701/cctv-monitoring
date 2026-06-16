package jftech

import (
	"crypto/md5"
	"fmt"
	"sync"
	"time"
)

var (
	counter uint64 = 0
	mu      sync.Mutex
)

func getCounter() string {
	mu.Lock()
	defer mu.Unlock()
	counter++
	if counter >= 10000000 {
		counter = 1
	}
	return fmt.Sprintf("%07d", counter)
}

// GetTimeMillis генерирует timeMillis = 7-значный счётчик + 13-значный timestamp
func GetTimeMillis() string {
	return getCounter() + fmt.Sprintf("%013d", time.Now().UnixMilli())
}

// change реализует простой сдвиг байтов (аналог Java-метода)
func change(data string, moveCard int) []byte {
	b := []byte(data)
	n := len(b)
	for i := 0; i < n; i++ {
		var temp byte
		if i%moveCard > (n-i)%moveCard {
			temp = b[i]
		} else {
			temp = b[n-i-1]
		}
		b[i], b[n-i-1] = b[n-i-1], temp
	}
	return b
}

// mergeByte объединяет исходный и сдвинутый массивы
func mergeByte(original, changed []byte) []byte {
	n := len(original)
	result := make([]byte, n*2)
	for i := 0; i < n; i++ {
		result[i] = original[i]
		result[n*2-1-i] = changed[i]
	}
	return result
}

// GenerateSignature возвращает пару (timeMillis, signature)
func GenerateSignature(uuid, appKey, appSecret string, moveCard int) (string, string) {
	timeMillis := GetTimeMillis()
	data := uuid + appKey + appSecret + timeMillis
	original := []byte(data)
	changed := change(data, moveCard)
	merged := mergeByte(original, changed)
	hash := md5.Sum(merged)
	return timeMillis, fmt.Sprintf("%x", hash)
}
