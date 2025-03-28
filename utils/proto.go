package utils

import "google.golang.org/protobuf/proto"

type __Message_[T any] interface {
	*T
	proto.Message
}

func PbEncode[T any, U __Message_[T]](msg *T) []byte {
	data, _ := proto.Marshal(U(msg))
	return data
}
