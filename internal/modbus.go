package internal

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/goburrow/modbus"
)

type ModbusServer struct {
	TCPHandler *modbus.TCPClientHandler
	RTUHandler *modbus.RTUClientHandler
	Client     modbus.Client
	LastAlive  time.Time
}

func (m *ModbusServer) Close() {
	if m == nil {
		return
	}
	if m.TCPHandler != nil {
		_ = m.TCPHandler.Close()
	}
	if m.RTUHandler != nil {
		_ = m.RTUHandler.Close()
	}
}

type TCPConfig struct {
	Host string `json:"host"`
	Port uint16 `json:"port"`
}

type RTUConfig struct {
	Port     string `json:"port"`
	BaudRate int    `json:"baud_rate"`
	DataBits int    `json:"data_bits"`
	Parity   string `json:"parity"`
	StopBits int    `json:"stop_bits"`
}

type ModbusConfig struct {
	Mode    string     `json:"mode"`
	SlaveID int        `json:"slave_id"`
	TCP     *TCPConfig `json:"tcp"`
	RTU     *RTUConfig `json:"rtu"`
	Host    string     `json:"host"` // legacy fallback
	Port    uint16     `json:"port"` // legacy fallback
}

func ConnModbus(config ModbusConfig) (*ModbusServer, error) {
	mode := normalizeMode(config.Mode)
	if mode == "" {
		if config.TCP != nil || config.Host != "" || config.Port != 0 {
			mode = "tcp"
		} else if config.RTU != nil {
			mode = "rtu"
		} else {
			mode = "tcp"
		}
	}

	if config.SlaveID < 0 || config.SlaveID > 255 {
		return nil, errors.New("slave_id must be between 0 and 255")
	}

	switch mode {
	case "tcp":
		return connectTCP(config)
	case "rtu":
		return connectRTU(config)
	default:
		return nil, fmt.Errorf("unsupported connection mode %q", mode)
	}
}

func normalizeMode(mode string) string {
	return strings.TrimSpace(strings.ToLower(mode))
}

func connectTCP(config ModbusConfig) (*ModbusServer, error) {
	tcpConfig := config.TCP
	if tcpConfig == nil {
		tcpConfig = &TCPConfig{
			Host: config.Host,
			Port: config.Port,
		}
	}
	if tcpConfig.Host == "" {
		return nil, errors.New("host is required for TCP connections")
	}
	if tcpConfig.Port == 0 {
		return nil, errors.New("port is required for TCP connections")
	}

	addr := fmt.Sprintf("%s:%d", tcpConfig.Host, tcpConfig.Port)
	log.Printf("Connecting to Modbus TCP server at %s\n", addr)
	handler := modbus.NewTCPClientHandler(addr)
	handler.SlaveId = byte(config.SlaveID)
	handler.Timeout = 2 * time.Second
	handler.IdleTimeout = 30 * time.Minute

	if err := handler.Connect(); err != nil {
		return nil, err
	}

	client := modbus.NewClient(handler)
	return &ModbusServer{
		TCPHandler: handler,
		Client:     client,
		LastAlive:  time.Now(),
	}, nil
}

func connectRTU(config ModbusConfig) (*ModbusServer, error) {
	if config.RTU == nil {
		return nil, errors.New("rtu configuration is required for RTU connections")
	}

	rtuCfg := *config.RTU
	if rtuCfg.Port == "" {
		return nil, errors.New("serial port is required for RTU connections")
	}

	if rtuCfg.BaudRate == 0 {
		rtuCfg.BaudRate = 9600
	}
	if rtuCfg.DataBits == 0 {
		rtuCfg.DataBits = 8
	}
	if rtuCfg.Parity == "" {
		rtuCfg.Parity = "N"
	}
	if rtuCfg.StopBits == 0 {
		rtuCfg.StopBits = 1
	}

	log.Printf("Connecting to Modbus RTU server on %s\n", rtuCfg.Port)
	handler := modbus.NewRTUClientHandler(rtuCfg.Port)
	handler.BaudRate = rtuCfg.BaudRate
	handler.DataBits = rtuCfg.DataBits
	handler.Parity = strings.ToUpper(rtuCfg.Parity)
	handler.StopBits = rtuCfg.StopBits
	handler.SlaveId = byte(config.SlaveID)
	handler.Timeout = 2 * time.Second
	handler.IdleTimeout = 30 * time.Minute

	if err := handler.Connect(); err != nil {
		return nil, err
	}

	client := modbus.NewClient(handler)
	return &ModbusServer{
		RTUHandler: handler,
		Client:     client,
		LastAlive:  time.Now(),
	}, nil
}
