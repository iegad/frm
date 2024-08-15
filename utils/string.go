package utils

import (
	"encoding/json"
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
