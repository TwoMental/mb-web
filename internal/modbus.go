package internal

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/goburrow/modbus"
)

type ModbusServer struct {
	clientMu sync.Mutex
	statusMu sync.Mutex

	TCPHandler *modbus.TCPClientHandler
	RTUHandler *modbus.RTUClientHandler
	Client     modbus.Client

	LastAlive time.Time
	Mode      string
	Config    ModbusConfig

	Connected            bool
	LastError            string
	lastReconnectAttempt time.Time
}

var ErrReadOnly = errors.New("register type is read-only")

func (m *ModbusServer) Close() {
	if m == nil {
		return
	}
	m.clientMu.Lock()
	defer m.clientMu.Unlock()

	if m.TCPHandler != nil {
		_ = m.TCPHandler.Close()
	}
	if m.RTUHandler != nil {
		_ = m.RTUHandler.Close()
	}
}

func (m *ModbusServer) markSuccess() {
	m.statusMu.Lock()
	m.Connected = true
	m.LastAlive = time.Now()
	m.LastError = ""
	m.statusMu.Unlock()
}

func (m *ModbusServer) markFailure(err error) {
	m.statusMu.Lock()
	m.Connected = false
	if err != nil {
		m.LastError = err.Error()
	}
	m.statusMu.Unlock()
}

func (m *ModbusServer) MarkSuccess() {
	m.markSuccess()
}

func (m *ModbusServer) MarkFailure(err error) {
	m.markFailure(err)
}

type ModbusStatus struct {
	Connected bool      `json:"connected"`
	Mode      string    `json:"mode"`
	LastAlive time.Time `json:"last_alive"`
	LastError string    `json:"last_error"`
}

func (m *ModbusServer) Status() ModbusStatus {
	m.statusMu.Lock()
	defer m.statusMu.Unlock()
	return ModbusStatus{
		Connected: m.Connected,
		Mode:      m.Mode,
		LastAlive: m.LastAlive,
		LastError: m.LastError,
	}
}

func (m *ModbusServer) Reconnect() error {
	if m == nil {
		return errors.New("no active connection")
	}

	m.clientMu.Lock()
	defer m.clientMu.Unlock()

	newServer, err := ConnModbus(m.Config)

	m.statusMu.Lock()
	m.lastReconnectAttempt = time.Now()
	m.statusMu.Unlock()

	if err != nil {
		m.markFailure(err)
		return err
	}

	if m.TCPHandler != nil {
		_ = m.TCPHandler.Close()
	}
	if m.RTUHandler != nil {
		_ = m.RTUHandler.Close()
	}

	m.TCPHandler = newServer.TCPHandler
	m.RTUHandler = newServer.RTUHandler
	m.Client = newServer.Client
	m.Mode = newServer.Mode
	m.Config = newServer.Config
	m.markSuccess()
	return nil
}

func (m *ModbusServer) EnsureConnection(maxIdle time.Duration) error {
	if m == nil {
		return errors.New("no active connection")
	}

	m.statusMu.Lock()
	connected := m.Connected
	lastAlive := m.LastAlive
	lastAttempt := m.lastReconnectAttempt
	m.statusMu.Unlock()

	if connected && (maxIdle <= 0 || time.Since(lastAlive) < maxIdle) {
		return nil
	}

	if time.Since(lastAttempt) < 5*time.Second {
		return nil
	}

	return m.Reconnect()
}

func (m *ModbusServer) ReadRegister(registerType RegisterType, address uint16) ([]byte, error) {
	m.clientMu.Lock()
	defer m.clientMu.Unlock()

	if m.Client == nil {
		return nil, errors.New("modbus client not initialized")
	}

	switch registerType {
	case RegisterTypeCoil:
		return m.Client.ReadCoils(address, 1)
	case RegisterTypeDiscreteInput:
		return m.Client.ReadDiscreteInputs(address, 1)
	case RegisterTypeInputRegister:
		return m.Client.ReadInputRegisters(address, 1)
	case RegisterTypeHoldingRegister, RegisterTypeDefault:
		return m.Client.ReadHoldingRegisters(address, 1)
	default:
		return nil, fmt.Errorf("invalid register type %d", registerType)
	}
}

func (m *ModbusServer) WriteSingle(registerType RegisterType, address uint16, value uint16) error {
	m.clientMu.Lock()
	defer m.clientMu.Unlock()

	if m.Client == nil {
		return errors.New("modbus client not initialized")
	}

	switch registerType {
	case RegisterTypeCoil:
		var coilValue uint16
		if value != 0 {
			coilValue = 0xFF00
		}
		_, err := m.Client.WriteSingleCoil(address, coilValue)
		return err
	case RegisterTypeHoldingRegister, RegisterTypeDefault:
		_, err := m.Client.WriteSingleRegister(address, value)
		return err
	default:
		return ErrReadOnly
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

	config.Mode = mode

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
	if config.TCP == nil {
		config.TCP = tcpConfig
	}
	server := &ModbusServer{
		TCPHandler: handler,
		Client:     client,
		Mode:       "tcp",
		Config:     config,
		Connected:  true,
	}
	server.markSuccess()
	return server, nil
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
	updatedCfg := rtuCfg
	config.RTU = &updatedCfg
	server := &ModbusServer{
		RTUHandler: handler,
		Client:     client,
		Mode:       "rtu",
		Config:     config,
		Connected:  true,
	}
	server.markSuccess()
	return server, nil
}
