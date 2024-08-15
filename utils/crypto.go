package utils

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"fmt"
)

func AesCBCEncrypt(origData, key, iv []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	origData = pkcs5Padding(origData, len(key))
	blockMode := cipher.NewCBCEncrypter(block, iv)
	crypted := make([]byte, len(origData))
	blockMode.CryptBlocks(crypted, origData)
	return base64.StdEncoding.EncodeToString(crypted), nil
}

func AesCBCDecrypt(encrypted, key, iv []byte) ([]byte, error) {
	encrypted, _ = base64.StdEncoding.DecodeString(string(encrypted))
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockMode := cipher.NewCBCDecrypter(block, iv)
	origData := make([]byte, len(encrypted))
	blockMode.CryptBlocks(origData, encrypted)
	origData, err = pkcs5Unpadding(origData)
	return origData, err
}

const (
	Pkcs5Padding = "PKCS5"
	Pkcs7Padding = "PKCS7"
	ZEROSPadding = "ZEROS"
)

type ecb struct {
	b         cipher.Block
	blockSize int
}

func newECB(b cipher.Block) *ecb {
	return &ecb{
		b:         b,
		blockSize: b.BlockSize(),
	}
}

type ecbEncrypter ecb
type ecbDecrypter ecb

func NewECBDecrypter(b cipher.Block) cipher.BlockMode {
	return (*ecbDecrypter)(newECB(b))
}

func (x *ecbDecrypter) BlockSize() int { return x.blockSize }

func (x *ecbDecrypter) CryptBlocks(dst, src []byte) {
	if len(src)%x.blockSize != 0 {
		panic("crypto/cipher: input not full blocks")
	}
	if len(dst) < len(src) {
		panic("crypto/cipher: output smaller than input")
	}
	for len(src) > 0 {
		x.b.Decrypt(dst, src[:x.blockSize])
		src = src[x.blockSize:]
		dst = dst[x.blockSize:]
	}
}

func newECBEncrypter(b cipher.Block) cipher.BlockMode {
	return (*ecbEncrypter)(newECB(b))
}

func (x *ecbEncrypter) BlockSize() int { return x.blockSize }

func (x *ecbEncrypter) CryptBlocks(dst, src []byte) {
	if len(src)%x.blockSize != 0 {
		panic("crypto/cipher: input not full blocks")
	}
	if len(dst) < len(src) {
		panic("crypto/cipher: output smaller than input")
	}
	for len(src) > 0 {
		x.b.Encrypt(dst, src[:x.blockSize])
		src = src[x.blockSize:]
		dst = dst[x.blockSize:]
	}
}

func pkcs5Padding(src []byte, blockSize int) []byte {
	return pkcs7Padding(src, blockSize)
}

func pkcs5Unpadding(src []byte) ([]byte, error) {
	return pkcs7UnPadding(src)
}

func pkcs7Padding(src []byte, blockSize int) []byte {
	padding := blockSize - len(src)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(src, padtext...)
}

func pkcs7UnPadding(src []byte) ([]byte, error) {
	length := len(src)
	if length == 0 {
		return src, fmt.Errorf("UnPadding error")
	}
	unpadding := int(src[length-1])
	if length < unpadding {
		return src, fmt.Errorf("UnPadding error")
	}
	return src[:(length - unpadding)], nil
}

func zerosPadding(src []byte, blockSize int) []byte {
	paddingCount := blockSize - len(src)%blockSize
	if paddingCount == 0 {
		return src
	} else {
		return append(src, bytes.Repeat([]byte{byte(0)}, paddingCount)...)
	}
}

func zerosUnPadding(src []byte) ([]byte, error) {
	for i := len(src) - 1; ; i-- {
		if src[i] != 0 {
			return src[:i+1], nil
		}
	}
}

func Padding(padding string, src []byte, blockSize int) []byte {
	switch padding {
	case Pkcs5Padding:
		src = pkcs5Padding(src, blockSize)
	case Pkcs7Padding:
		src = pkcs7Padding(src, blockSize)
	case ZEROSPadding:
		src = zerosPadding(src, blockSize)
	}
	return src
}

func UnPadding(padding string, src []byte) ([]byte, error) {
	switch padding {
	case Pkcs5Padding:
		return pkcs5Unpadding(src)
	case Pkcs7Padding:
		return pkcs7UnPadding(src)
	case ZEROSPadding:
		return zerosUnPadding(src)
	}
	return src, nil
}

// AesECBEncrypt aes加密算法，ECB模式，可以指定三种填充模式
func AesECBEncrypt(plaintext, key []byte, padding string) []byte {
	// 要加密的内容长度必须是快长度的整数倍，首先进行填充
	plaintext = Padding(padding, plaintext, aes.BlockSize)
	if len(plaintext)%aes.BlockSize != 0 {
		return nil
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}

	ciphertext := make([]byte, len(plaintext))
	mode := newECBEncrypter(block)
	mode.CryptBlocks(ciphertext, plaintext)

	return ciphertext
}

// AesECBDecrypt aes解密算法，ECB模式，可以指定三种填充模式
func AesECBDecrypt(ciphertext, key []byte, padding string) []byte {

	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}

	if len(ciphertext) < aes.BlockSize {
		panic("ciphertext too short")
	}

	// ECB mode always works in whole blocks.
	if len(ciphertext)%aes.BlockSize != 0 {
		panic("ciphertext is not a multiple of the block size")
	}

	mode := NewECBDecrypter(block)

	// CryptBlocks can work in-place if the two arguments are the same.
	mode.CryptBlocks(ciphertext, ciphertext)

	ciphertext, err = UnPadding(padding, ciphertext)
	if err != nil {
		return nil
	}

	return ciphertext
}
