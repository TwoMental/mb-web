package internal

import "fmt"

// BytesToHexStrings converts a byte slice into a slice of hex string values.
func BytesToHexStrings(b []byte) []string {
	hexStrings := make([]string, len(b))
	for i, byteValue := range b {
		hexStrings[i] = fmt.Sprintf("0x%02x", byteValue)
	}
	return hexStrings
}
