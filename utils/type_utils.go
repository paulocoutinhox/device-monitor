package utils

import (
	"strconv"
	"encoding/binary"
	"io"
)

func ConvertByteToInt(value byte) int {
	result, _ := strconv.Atoi(string(value))
	return result
}

func ConvertStringToInt(value string) int64 {
	result, _ := strconv.ParseInt(value, 10, 64)
	return result
}

func ConvertStringToFloat(value string) float64 {
	result, _ := strconv.ParseFloat(value, 64)
	return result
}

func ConvertByteToString(value byte) string {
	return string(value)
}

func BinaryRead(conn io.Reader, data interface{}) error {
	err := binary.Read(conn, binary.LittleEndian, data)
	return err
}