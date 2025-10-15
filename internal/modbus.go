package internal

import (
	"fmt"
	"log"
	"time"

	"github.com/goburrow/modbus"
)

type ModbusServer struct {
	H         *modbus.TCPClientHandler
	C         modbus.Client
	LastAlive time.Time
}

type ModbusConfig struct {
	Host    string `json:"host" binding:"required"`
	Port    uint16 `json:"port" binding:"required" validate:"gte=1,lte=65535"`
	SlaveID int    `json:"slave_id" binding:"required" validate:"gte=1,lte=255"`
}

func ConnModbus(config ModbusConfig) (*ModbusServer, error) {
	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	log.Printf("Connecting to Modbus server at %s\n", addr)
	handler := modbus.NewTCPClientHandler(addr)
	handler.SlaveId = byte(config.SlaveID)
	handler.Timeout = 2 * time.Second
	handler.IdleTimeout = 30 * time.Minute

	err := handler.Connect()
	if err != nil {
		return nil, err
	}

	client := modbus.NewClient(handler)
	return &ModbusServer{H: handler, C: client, LastAlive: time.Now()}, nil
}
