package internal

// RegisterType represents the different Modbus register categories.
type RegisterType uint8

const (
	RegisterTypeDefault         RegisterType = iota // Defaults to holding register for backward compatibility
	RegisterTypeCoil                                // Coil (0x01 read, 0x05 write single, 0x15 write multiple)
	RegisterTypeDiscreteInput                       // Discrete input (0x02 read)
	RegisterTypeInputRegister                       // Input register (0x04 read)
	RegisterTypeHoldingRegister                     // Holding register (0x03 read, 0x06 write single, 0x16 write multiple)
)
