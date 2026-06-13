// Package xiongmai реализует P2P-адаптер для устройств Xiongmai с использованием NetSDK.
package xiongmai

/*
#cgo CFLAGS: -I./third_party/xiongmai/include
#cgo LDFLAGS: -L/usr/local/lib -lh264net -lh264play -lstdc++

#include <stdlib.h>
#include <string.h>
#include "h264_net_sdk.h"
#include "h264_play.h"

// Экспортируемый Go callback для видеоданных
extern void goVideoCallback(int handle, char* data, int len, int type, void* user);

// Статическая функция-прокси для передачи в SDK
static void cVideoCallback(int handle, char* data, int len, int type, void* user) {
    goVideoCallback(handle, data, len, type, user);
}
*/
import "C"
import (
	"errors"
	"fmt"
	"sync"
	"unsafe"
)

// Adapter представляет P2P-подключение к одному устройству Xiongmai.
type Adapter struct {
	serial   string
	user     string
	pass     string
	loginID  C.long
	playPort C.int
	callback func([]byte) // внешний callback для получения видео/H.264 кадров
	mu       sync.Mutex
	isActive bool
}

// NewAdapter создаёт новый экземпляр адаптера (без подключения).
func NewAdapter() *Adapter {
	return &Adapter{}
}

// Connect устанавливает P2P-соединение с устройством по серийному номеру.
// Параметры: serial - серийный номер (например, "ABCD1234EFGH5678"),
//
//	user   - имя пользователя устройства (обычно "admin"),
//	pass   - пароль устройства.
func (a *Adapter) Connect(serial, user, pass string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.isActive {
		return errors.New("already connected")
	}

	a.serial = serial
	a.user = user
	a.pass = pass

	cSerial := C.CString(serial)
	cUser := C.CString(user)
	cPass := C.CString(pass)
	defer C.free(unsafe.Pointer(cSerial))
	defer C.free(unsafe.Pointer(cUser))
	defer C.free(unsafe.Pointer(cPass))

	var devInfo C.DEVICEINFO
	var errCode C.int
	// Тип подключения 2 = P2P/Cloud режим
	loginID := C.H264_DVR_Login(cSerial, 34567, cUser, cPass, &devInfo, &errCode, 2)
	if loginID <= 0 {
		return fmt.Errorf("login failed with code %d", errCode)
	}
	a.loginID = loginID

	// Получить свободный порт для плеера
	var port C.int
	if C.H264_PLAY_GetPort(&port) != 0 {
		C.H264_DVR_Logout(loginID)
		return errors.New("failed to get player port")
	}
	a.playPort = port

	// Открыть поток и запустить плеер (без окна)
	if C.H264_PLAY_OpenStream(port, nil, 0, 1024*1024) != 0 {
		C.H264_PLAY_FreePort(port)
		C.H264_DVR_Logout(loginID)
		return errors.New("failed to open stream")
	}
	C.H264_PLAY_Play(port, nil)

	// Установить callback для получения сырых видеоданных
	C.H264_DVR_SetRealDataCallBack(loginID, (C.H264_RealDataCallBack)(C.cVideoCallback), unsafe.Pointer(a))

	// Запустить реальный просмотр (канал 0, основной поток)
	var playStru C.H264_DVR_PLAY_INFO
	playStru.nChannel = 0
	playStru.nStreamType = 0 // 0 - main, 1 - sub
	if C.H264_DVR_RealPlay(loginID, &playStru) != 0 {
		a.Disconnect()
		return errors.New("failed to start real play")
	}

	a.isActive = true
	return nil
}

// StartStream запускает передачу видеопотока в указанный callback.
// callback будет вызываться для каждого полученного кадра (с приватным заголовком).
func (a *Adapter) StartStream(callback func([]byte)) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.isActive {
		return errors.New("not connected")
	}
	a.callback = callback
	return nil
}

// PTZControl отправляет команду управления PTZ.
// cmd - команда (например, 0x21 = влево, 0x22 = вправо, 0x23 = вверх, 0x24 = вниз,
//
//	0x25 = Zoom+, 0x26 = Zoom-).
//
// speed - скорость (1-7).
// stop - true для остановки движения, false для начала.
func (a *Adapter) PTZControl(cmd int, speed int, stop bool) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.isActive {
		return errors.New("not connected")
	}
	bStop := 0
	if stop {
		bStop = 1
	}
	ret := C.H264_DVR_PTZControl(a.loginID, 0, C.long(cmd), 0, C.long(bStop), C.long(speed))
	if ret != 0 {
		return fmt.Errorf("PTZ control failed with code %d", ret)
	}
	return nil
}

// Disconnect закрывает соединение с устройством и освобождает ресурсы.
func (a *Adapter) Disconnect() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.isActive {
		return nil
	}

	if a.playPort != 0 {
		C.H264_PLAY_Stop(a.playPort)
		C.H264_PLAY_CloseStream(a.playPort)
		C.H264_PLAY_FreePort(a.playPort)
		a.playPort = 0
	}
	if a.loginID != 0 {
		C.H264_DVR_StopRealPlay(a.loginID)
		C.H264_DVR_Logout(a.loginID)
		a.loginID = 0
	}
	a.callback = nil
	a.isActive = false
	return nil
}

// export goVideoCallback – обратный вызов из C, передающий видеоданные.
//
//export goVideoCallback
func goVideoCallback(handle C.int, data *C.char, length C.int, dataType C.int, user unsafe.Pointer) {
	a := (*Adapter)(user)
	if a == nil || a.callback == nil {
		return
	}
	// Копируем данные в слайс байт; dataType может указывать на тип кадра (I/P/B)
	bytes := C.GoBytes(unsafe.Pointer(data), length)
	a.callback(bytes)
}

// InitSDK глобально инициализирует NetSDK (должна быть вызвана один раз при старте сервиса).
func InitSDK() error {
	ret := C.H264_DVR_Init()
	if ret != 0 {
		return errors.New("H264_DVR_Init failed")
	}
	return nil
}

// CleanupSDK освобождает глобальные ресурсы SDK (вызывать при завершении сервиса).
func CleanupSDK() {
	C.H264_DVR_Cleanup()
}
