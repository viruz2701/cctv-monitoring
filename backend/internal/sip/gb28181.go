package sip

import (
	"fmt"
	"regexp"
)

// GB28181 DeviceID структура (20 цифр)
// Пример: 34020000001310000001
// 34 - тип устройства (камера)
// 02 - регион
// 000000 - производитель
// 131 - модель
// 0000001 - серийный номер

type GB28181DeviceID struct {
	Raw          string
	TypeCode     string // 2 цифры
	RegionCode   string // 2 цифры
	Manufacturer string // 6 цифр
	ModelCode    string // 3 цифры
	SerialNumber string // 7 цифр
	IsValid      bool
}

var gb28181Regex = regexp.MustCompile(`^(\d{2})(\d{2})(\d{6})(\d{3})(\d{7})$`)

func ParseGB28181DeviceID(deviceID string) *GB28181DeviceID {
	result := &GB28181DeviceID{
		Raw:     deviceID,
		IsValid: false,
	}

	matches := gb28181Regex.FindStringSubmatch(deviceID)
	if matches == nil {
		return result
	}

	result.TypeCode = matches[1]
	result.RegionCode = matches[2]
	result.Manufacturer = matches[3]
	result.ModelCode = matches[4]
	result.SerialNumber = matches[5]
	result.IsValid = true

	return result
}

func (d *GB28181DeviceID) IsCamera() bool {
	return d.TypeCode == "34" || d.TypeCode == "35" || d.TypeCode == "36"
}

func (d *GB28181DeviceID) IsNVR() bool {
	return d.TypeCode == "11" || d.TypeCode == "12" || d.TypeCode == "13"
}

func (d *GB28181DeviceID) IsPlatform() bool {
	return d.TypeCode == "21" || d.TypeCode == "22"
}

func (d *GB28181DeviceID) String() string {
	if !d.IsValid {
		return fmt.Sprintf("Invalid GB28181 ID: %s", d.Raw)
	}
	return fmt.Sprintf("GB28181[type=%s,region=%s,mfr=%s,model=%s,serial=%s]",
		d.TypeCode, d.RegionCode, d.Manufacturer, d.ModelCode, d.SerialNumber)
}
