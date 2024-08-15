package nw

import "unsafe"

const (
	DEFAULT_TIMEOUT int32 = 300
	MAX_TIMEOUT     int32 = 600
	UINT32_SIZE           = int(unsafe.Sizeof(uint32(0)))
	MAX_BUF_SIZE          = 1024 * 1024 * 2
)
