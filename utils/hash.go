package utils

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"math/rand"
)

func MD5(raw []byte) []byte {
	h := md5.New()
	h.Write(raw)
	return h.Sum(nil)
}

func MD5Hex(raw string) string {
	h := md5.New()
	h.Write(*Str2Bytes(raw))
	return hex.EncodeToString(h.Sum(nil))
}

func SHA256(raw []byte) []byte {
	h := sha256.Sum256(raw)
	return h[:]
}

func SHA256Hex(raw string) string {
	h := sha256.Sum256(*Str2Bytes(raw))
	return hex.EncodeToString(h[:])
}

// 随机生成 [min, max] 的随机数
func RandomRange(r *rand.Rand, min, max int64) int64 {
	return r.Int63n(max-min+1) + min
}
