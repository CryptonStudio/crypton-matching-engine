package itch

import (
	"encoding/binary"
	"time"
)

func readByte(data []byte) (byte, []byte) {
	return data[0], data[1:]
}

func readBytes2(data []byte) ([2]byte, []byte) {
	return [2]byte{data[0], data[1]}, data[2:]
}

func readBytes4(data []byte) ([4]byte, []byte) {
	return [4]byte{data[0], data[1], data[2], data[3]}, data[4:]
}

func readBytes8(data []byte) ([8]byte, []byte) {
	return [8]byte{data[0], data[1], data[2], data[3], data[4], data[5], data[6], data[7]}, data[8:]
}

// func readString2(data []byte) (string, []byte) {
// 	return string(data[:2]), data[2:]
// }

// func readString4(data []byte) (string, []byte) {
// 	return string(data[:4]), data[4:]
// }

// func readString8(data []byte) (string, []byte) {
// 	return string(data[:8]), data[8:]
// }

func readUint16(data []byte) (uint16, []byte) {
	return binary.BigEndian.Uint16(data), data[2:]
}

func readUint32(data []byte) (uint32, []byte) {
	return binary.BigEndian.Uint32(data), data[4:]
}

func readUint64(data []byte) (uint64, []byte) {
	return binary.BigEndian.Uint64(data), data[8:]
}

func readTime(data []byte) (time.Time, []byte) {
	ns := (int64(data[0]) << 40) + (int64(data[1]) << 32) + (int64(data[2]) << 24) + (int64(data[3]) << 16) + (int64(data[4]) << 8) + int64(data[5])
	seconds := ns / int64(time.Second)
	ns = ns % int64(time.Second)
	return time.Unix(seconds, ns), data[6:]
}
