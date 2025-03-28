package utils

import (
	"encoding/json"
	"strings"
	"unsafe"
)

func Str2Bytes(s string) *[]byte {
	return (*[]byte)(unsafe.Pointer(&s))
}

func Bytes2Str(b []byte) *string {
	return (*string)(unsafe.Pointer(&b))
}

func JSON(v interface{}) string {
	jstr, _ := json.Marshal(v)
	return string(jstr)
}

func GetFileSuffix(fname string) string {
	idx := strings.LastIndex(fname, ".")
	if idx < 0 {
		return ""
	}

	return strings.ToLower(fname[idx+1:])
}

func StartWith(raw, start string) bool {
	return strings.HasPrefix(raw, start)
}

func EndWith(raw, end string) bool {
	return strings.HasSuffix(raw, end)
}
