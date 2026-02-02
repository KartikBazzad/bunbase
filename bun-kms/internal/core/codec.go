package core

import (
	"encoding/binary"
	"errors"
)

func EncodeCiphertext(version int, nonce, data []byte) ([]byte, error) {
	if version <= 0 {
		return nil, errors.New("invalid version")
	}
	if len(nonce) == 0 {
		return nil, errors.New("nonce required")
	}
	out := make([]byte, 4+len(nonce)+len(data))
	binary.BigEndian.PutUint32(out[:4], uint32(version))
	copy(out[4:], nonce)
	copy(out[4+len(nonce):], data)
	return out, nil
}

func DecodeCiphertext(blob []byte, nonceSize int) (int, []byte, []byte, error) {
	if len(blob) < 4+nonceSize {
		return 0, nil, nil, errors.New("ciphertext too short")
	}
	version := int(binary.BigEndian.Uint32(blob[:4]))
	nonce := make([]byte, nonceSize)
	copy(nonce, blob[4:4+nonceSize])
	data := make([]byte, len(blob)-(4+nonceSize))
	copy(data, blob[4+nonceSize:])
	return version, nonce, data, nil
}
