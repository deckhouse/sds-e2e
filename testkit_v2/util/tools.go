package integration

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/base64"
)


func hashMd5(in string) string {
	binHash := md5.Sum([]byte(in))
	return hex.EncodeToString(binHash[:])
}

func base64Encode(in string) string {
	return base64.StdEncoding.EncodeToString([]byte(in))
}
