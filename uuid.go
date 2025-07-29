package log2fuse

import (
	"crypto/rand"
	"encoding/hex"
)

// UUID representation compliant with specification
// described in RFC 4122.
type UUID [16]byte

// String returns canonical string representation of UUID.
func (u UUID) String() string {
	buf := make([]byte, 36)

	hex.Encode(buf[0:8], u[0:4])
	buf[8] = '-'
	hex.Encode(buf[9:13], u[4:6])
	buf[13] = '-'
	hex.Encode(buf[14:18], u[6:8])
	buf[18] = '-'
	hex.Encode(buf[19:23], u[8:10])
	buf[23] = '-'
	hex.Encode(buf[24:], u[10:])

	return string(buf)
}

// Must is a helper that wraps a call to a function returning (UUID, error)
// and panics if the error is non-nil.
func Must(u UUID, err error) UUID {
	if err != nil {
		panic(err)
	}
	return u
}

// GenerateUUID4 generates RFC 4122 version 4 UUID.
func GenerateUUID4() (UUID, error) {
	u := UUID{}
	_, err := rand.Read(u[:])
	if err != nil {
		return UUID{}, err
	}
	u[6] = (u[6] & 0x0f) | 0x40 // Version 4
	u[8] = (u[8] & 0x3f) | 0x80 // Variant is 10
	return u, nil
}
