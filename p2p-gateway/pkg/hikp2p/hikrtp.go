package hikp2p

import "encoding/binary"

const (
	HikRTPHeaderLen = 12
)

func ExtractNALU(payload []byte) []byte {
	if len(payload) < HikRTPHeaderLen {
		return nil
	}
	// Тип 0x8060 / 0x8050 / 0x8051 – видео
	typ := binary.BigEndian.Uint16(payload[0:2])
	if typ != 0x8060 && typ != 0x8050 && typ != 0x8051 {
		return nil
	}
	// Пропускаем Hik‑RTP заголовок
	rtpPayload := payload[HikRTPHeaderLen:]
	if len(rtpPayload) < 13 || rtpPayload[0] != 0x0d {
		// нет суб‑заголовка – просто возвращаем остаток
		return rtpPayload
	}
	// Суб‑заголовок 13 байт
	subHeader := rtpPayload[1]
	if subHeader&0xf0 == 0x80 || subHeader&0xf0 == 0x90 {
		// аудио – пропускаем
		return nil
	}
	nalData := rtpPayload[13:]
	return nalData
}
