package common

import (
	"crypto/aes"
	"crypto/cipher"
	"bytes"
	"compress/flate"
	"strings"
	"io/ioutil"
	"fmt"
)


//aes encrpty alg with mode of CBC
func AesCBCEncrypt(origData, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()
	origData = PKCS7Padding(origData, blockSize)
	blockMode := cipher.NewCBCEncrypter(block, []byte{0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0})
	crypted := make([]byte, len(origData))
	blockMode.CryptBlocks(crypted, origData)
	return crypted, nil
}

//aes decrpty alg with mode of CBC
func AesCBCDecrypt(crypted, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	//blockSize := block.BlockSize()
	blockMode := cipher.NewCBCDecrypter(block, []byte{0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0})
	origData := make([]byte, len(crypted))
	blockMode.CryptBlocks(origData, crypted)
	origData = PKCS7UnPadding(origData)
	return origData, nil
}

//padding deal
func PKCS7Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext) % blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

//unpadding deal
func PKCS7UnPadding(origData []byte) []byte {
	length := len(origData)
	unpadding := int(origData[length-1])
	return origData[:(length - unpadding)]
}

//gzflate decode
func Gzdecode(data string) string  {
	if data == "" {
		return ""
	}
	r :=flate.NewReader(strings.NewReader(data))
	defer r.Close()
	out, err := ioutil.ReadAll(r)
	if err !=nil {
		fmt.Printf("%v\n",err)
		return ""
	}
	return string(out)
}

//gzflate encode
func Gzencode(data string,level int) []byte  {
	if data == "" {
		return []byte{}
	}
	var bufs bytes.Buffer
	w,_ :=flate.NewWriter(&bufs,level)
	w.Write([]byte(data))
	w.Flush()
	w.Close()
	return bufs.Bytes()
}

