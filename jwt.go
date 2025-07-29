package log2fuse

import (
	"encoding/base64"
	"strings"
)

func decodeJWTHeader(value string) string {
	withBearer := strings.HasPrefix(value, "Bearer ")
	var token string
	if withBearer {
		token = strings.TrimPrefix(value, "Bearer ")
	} else {
		token = value
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return value
	}
	decodedParts, err := decodeEach(parts[0:2], base64Decode)
	if err != nil {
		return value
	}
	if withBearer {
		return "Bearer " + strings.Join(decodedParts, ".")
	}
	return strings.Join(decodedParts, ".")
}

func base64Decode(encodedString string) (string, error) {
	decodedBytes, err := base64.RawURLEncoding.DecodeString(encodedString)
	if err != nil {
		return "", err
	}
	return string(decodedBytes), nil
}
