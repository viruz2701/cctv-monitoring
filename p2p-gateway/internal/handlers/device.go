package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yourorg/p2p-gateway/internal/adapters/hikvision"
	"github.com/yourorg/p2p-gateway/internal/db"
	"github.com/yourorg/p2p-gateway/internal/models"
	"golang.org/x/crypto/bcrypt"
)

type DeviceHandler struct {
	db         *db.DB
	hikAdapter *hikvision.Adapter
}

func NewDeviceHandler(db *db.DB, hikAdapter *hikvision.Adapter) *DeviceHandler {
	return &DeviceHandler{db: db, hikAdapter: hikAdapter}
}

// POST /p2p/register
func (h *DeviceHandler) RegisterDevice(c *gin.Context) {
	var req struct {
		Serial       string  `json:"serial" binding:"required"`
		Brand        string  `json:"brand" binding:"required"`
		SecurityCode string  `json:"security_code" binding:"required"`
		CloudUser    *string `json:"cloud_user"`
		CloudPass    *string `json:"cloud_pass"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Проверка доступности через соответствующий адаптер
	var online bool
	var err error
	switch req.Brand {
	case "hikvision":
		online, err = h.hikAdapter.CheckDevice(req.Serial, req.SecurityCode)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported brand"})
		return
	}
	if err != nil || !online {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device not reachable via P2P"})
		return
	}

	// Шифрование облачных учетных данных
	var encryptedPass *string
	if req.CloudPass != nil {
		hash, _ := bcrypt.GenerateFromPassword([]byte(*req.CloudPass), bcrypt.DefaultCost)
		encrypted := string(hash)
		encryptedPass = &encrypted
	}

	dev := &models.P2PDevice{
		Serial:       req.Serial,
		Brand:        req.Brand,
		SecurityCode: req.SecurityCode,
		CloudUser:    req.CloudUser,
		CloudPass:    encryptedPass,
		Status:       "online",
	}

	if err := h.db.CreateP2PDevice(dev); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "device registered", "id": dev.ID})
}

// GET /p2p/status/:serial
func (h *DeviceHandler) GetStatus(c *gin.Context) {
	serial := c.Param("serial")
	dev, err := h.db.GetP2PDeviceBySerial(serial)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "device not found"})
		return
	}
	// Проксируем запрос актуального статуса через адаптер
	status, _ := h.hikAdapter.GetStatus(serial, dev.SecurityCode)
	c.JSON(http.StatusOK, gin.H{"serial": serial, "status": status})
}

// GET /p2p/snapshot/:serial
func (h *DeviceHandler) GetSnapshot(c *gin.Context) {
	serial := c.Param("serial")
	dev, err := h.db.GetP2PDeviceBySerial(serial)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "device not found"})
		return
	}
	imageData, err := h.hikAdapter.GetSnapshot(serial, dev.SecurityCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "snapshot failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"serial": serial, "image_base64": imageData})
}

// POST /p2p/command/:serial
func (h *DeviceHandler) SendCommand(c *gin.Context) {
	serial := c.Param("serial")
	var req models.CommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	dev, err := h.db.GetP2PDeviceBySerial(serial)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "device not found"})
		return
	}
	// Отправка команды через адаптер
	err = h.hikAdapter.SendPTZCommand(serial, dev.SecurityCode, req.Command, req.Params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "command failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "command sent"})
}
