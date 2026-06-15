package hikp2p

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"fmt"
)

const V3Magic = 0xe2

func crc8(data []byte) byte {
	crc := byte(0)
	for _, b := range data {
		crc ^= b
		for i := 0; i < 8; i++ {
			if crc&0x80 != 0 {
				crc = (crc << 1) ^ 0x07
			} else {
				crc <<= 1
			}
		}
	}
	return crc
}

var V3IV = []byte{0x30, 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37, 0, 0, 0, 0, 0, 0, 0, 0}

func aes128CBCEncrypt(plain, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key[:16])
	if err != nil {
		return nil, err
	}
	padded := pkcs5Pad(plain, aes.BlockSize)
	ciphertext := make([]byte, len(padded))
	mode := cipher.NewCBCEncrypter(block, V3IV)
	mode.CryptBlocks(ciphertext, padded)
	return ciphertext, nil
}

func aes128CBCDecrypt(ciphertext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key[:16])
	if err != nil {
		return nil, err
	}
	if len(ciphertext)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("ciphertext not multiple of blocksize")
	}
	plain := make([]byte, len(ciphertext))
	mode := cipher.NewCBCDecrypter(block, V3IV)
	mode.CryptBlocks(plain, ciphertext)
	return pkcs5Unpad(plain)
}

func pkcs5Pad(src []byte, blockSize int) []byte {
	padding := blockSize - len(src)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(src, padtext...)
}

func pkcs5Unpad(src []byte) ([]byte, error) {
	length := len(src)
	unpad := int(src[length-1])
	if unpad > length {
		return nil, fmt.Errorf("invalid padding")
	}
	return src[:(length - unpad)], nil
}

type V3Message struct {
	MsgType    uint16
	Seq        uint32
	Reserved   uint16
	Encrypt    bool
	SaltVer    int
	SaltIdx    int
	ExpandHdr  bool
	Is2BLen    bool
	Attributes []V3Attribute
}

type V3Attribute struct {
	Tag   byte
	Value []byte
}

// EncodeV3Message кодирует V3 сообщение в байты.
func EncodeV3Message(msg V3Message, key []byte) ([]byte, error) {
	// Собираем тело атрибутов
	body, err := encodeAttributes(msg.Attributes, msg.Is2BLen)
	if err != nil {
		return nil, err
	}
	if msg.Encrypt && key != nil {
		body, err = aes128CBCEncrypt(body, key)
		if err != nil {
			return nil, err
		}
	}
	// Заголовок
	mask := byte(0)
	if msg.Encrypt {
		mask |= 0x80
	}
	mask |= byte((msg.SaltVer & 1) << 6)
	mask |= byte((msg.SaltIdx & 7) << 3)
	if msg.ExpandHdr {
		mask |= 0x04
	}
	if msg.Is2BLen {
		mask |= 0x02
	}
	header := make([]byte, 12)
	header[0] = V3Magic
	header[1] = mask
	binary.BigEndian.PutUint16(header[2:4], msg.MsgType)
	binary.BigEndian.PutUint32(header[4:8], msg.Seq)
	binary.BigEndian.PutUint16(header[8:10], msg.Reserved)
	header[10] = 12 // headerLen без expand
	if msg.ExpandHdr {
		// Для простоты не поддерживаем expand header (в P2P_SETUP не нужен)
		return nil, fmt.Errorf("expand header not implemented")
	}
	full := append(header, body...)
	full[11] = crc8(full[:len(full)])
	return full, nil
}

// DecodeV3Message декодирует V3 сообщение из сырых байт.
// Если передан ключ, расшифровывает тело.
func DecodeV3Message(data []byte, key []byte) (*V3Message, error) {
	if len(data) < 12 {
		return nil, fmt.Errorf("data too short")
	}
	// Проверка magic
	if data[0] != V3Magic {
		return nil, fmt.Errorf("invalid V3 magic: 0x%02x", data[0])
	}
	mask := data[1]
	msgType := binary.BigEndian.Uint16(data[2:4])
	seq := binary.BigEndian.Uint32(data[4:8])
	reserved := binary.BigEndian.Uint16(data[8:10])
	headerLen := data[10]
	if int(headerLen) > len(data) {
		return nil, fmt.Errorf("headerLen too big")
	}
	// Проверка CRC
	crcData := make([]byte, len(data))
	copy(crcData, data)
	crcData[11] = 0
	if crc8(crcData) != data[11] {
		return nil, fmt.Errorf("CRC mismatch")
	}
	// Парсим флаги
	encrypt := (mask & 0x80) != 0
	saltVer := int((mask >> 6) & 1)
	saltIdx := int((mask >> 3) & 7)
	expandHdr := (mask & 0x04) != 0
	is2BLen := (mask & 0x02) != 0

	bodyStart := int(headerLen)
	if bodyStart > len(data) {
		return nil, fmt.Errorf("headerLen exceeds data length")
	}
	body := data[bodyStart:]
	if encrypt && key != nil {
		dec, err := aes128CBCDecrypt(body, key)
		if err != nil {
			return nil, err
		}
		body = dec
	}
	attrs, err := decodeAttributes(body, is2BLen)
	if err != nil {
		return nil, err
	}
	return &V3Message{
		MsgType:    msgType,
		Seq:        seq,
		Reserved:   reserved,
		Encrypt:    encrypt,
		SaltVer:    saltVer,
		SaltIdx:    saltIdx,
		ExpandHdr:  expandHdr,
		Is2BLen:    is2BLen,
		Attributes: attrs,
	}, nil
}

// encodeAttributes кодирует список атрибутов в TLV формат.
func encodeAttributes(attrs []V3Attribute, is2BLen bool) ([]byte, error) {
	var buf []byte
	for _, a := range attrs {
		if a.Tag == 0x07 && is2BLen {
			buf = append(buf, a.Tag)
			lenBuf := make([]byte, 2)
			binary.BigEndian.PutUint16(lenBuf, uint16(len(a.Value)))
			buf = append(buf, lenBuf...)
			buf = append(buf, a.Value...)
		} else {
			buf = append(buf, a.Tag, byte(len(a.Value)))
			buf = append(buf, a.Value...)
		}
	}
	return buf, nil
}

// decodeAttributes разбирает TLV атрибуты из буфера.
// Если is2BLen true, для тега 0x07 используется 2‑байтовая длина.
func decodeAttributes(data []byte, is2BLen bool) ([]V3Attribute, error) {
	var attrs []V3Attribute
	offset := 0
	for offset < len(data) {
		if offset+1 > len(data) {
			return nil, fmt.Errorf("unexpected EOF reading tag")
		}
		tag := data[offset]
		offset++

		var valueLen int
		if tag == 0x07 && is2BLen {
			if offset+2 > len(data) {
				return nil, fmt.Errorf("unexpected EOF reading 2‑byte length")
			}
			valueLen = int(binary.BigEndian.Uint16(data[offset : offset+2]))
			offset += 2
		} else {
			if offset > len(data) {
				return nil, fmt.Errorf("unexpected EOF reading length")
			}
			valueLen = int(data[offset])
			offset++
		}
		if offset+valueLen > len(data) {
			return nil, fmt.Errorf("value exceeds buffer")
		}
		value := make([]byte, valueLen)
		copy(value, data[offset:offset+valueLen])
		offset += valueLen

		attrs = append(attrs, V3Attribute{Tag: tag, Value: value})
	}
	return attrs, nil
}
