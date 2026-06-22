package module

import (
	"bytes"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"

	"cube/internal/builtin"

	"github.com/emmansun/gmsm/sm2"
	"github.com/emmansun/gmsm/sm3"
	"github.com/emmansun/gmsm/sm4"
)

func init() {
	register("crypto", func(ctx Context) interface{} {
		return &CryptoClient{}
	})
}

//#region Cipher

type CryptoCipherClient interface {
	Encrypt(input []byte, key []byte, options map[string]interface{}) (builtin.Buffer, error)
	Decrypt(input []byte, key []byte, options map[string]interface{}) (builtin.Buffer, error)
}

type BaseCipherClient struct{}

func (c *BaseCipherClient) pad(input []byte, blockSize int, padType string) ([]byte, error) {
	switch strings.ToLower(padType) {
	case "none":
		return input, nil
	case "pkcs5": // pkcs5 填充模式：为 pkcs7 的子集，方式与 pkcs7 相同，不同的是 pkcs5 的 blockSize 固定为 8，而 pkcs7 的 blockSize 为 1 - 255
		fallthrough
	case "pkcs7": // pkcs7 填充模式：在原文末尾填充 padSize（其中 1 ≤ padSize ≤ blockSize）个字节 padByte（值为 padSize），使得总长度为 blockSize 的整数倍
		padSize := blockSize - (len(input) % blockSize)                      // 需要填充的长度
		padByte := byte(padSize)                                             // 需要填充的字节
		return append(input, bytes.Repeat([]byte{padByte}, padSize)...), nil // 在原文末尾填充 padSize 个字节 padByte
	case "zero": // zero padding：在原文末尾填充 0x00，使得总长度为 blockSize 的整数倍
		padSize := blockSize - (len(input) % blockSize)
		if padSize == blockSize {
			padSize = 0 // 已经是 blockSize 的整数倍，不需要填充
		}
		return append(input, bytes.Repeat([]byte{0x00}, padSize)...), nil
	default:
		return nil, fmt.Errorf("padding %s is not supported", padType)
	}
}

func (c *BaseCipherClient) unpad(input []byte, blockSize int, padType string) ([]byte, error) {
	switch strings.ToLower(padType) {
	case "none":
		return input, nil
	case "pkcs5":
		fallthrough // 同 pkcs7
	case "pkcs7":
		padByte := input[len(input)-1]         // 最后一个字节，即为填充所使用的字节
		padSize := int(padByte)                // 填充的字节值，也是所填充字节的长度
		return input[:len(input)-padSize], nil // 去除末尾 padSize 个字节
	case "zero": // zero padding：去除末尾所有的 0x00 字节
		for len(input) > 0 && input[len(input)-1] == 0x00 {
			input = input[:len(input)-1]
		}
		return input, nil
	default:
		return nil, fmt.Errorf("padding %s is not supported", padType)
	}
}

type AesEcbCipherClient struct{ BaseCipherClient }

func (c *AesEcbCipherClient) Encrypt(input []byte, key []byte, options map[string]interface{}) (builtin.Buffer, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()

	padding, err := read[string](options, "padding", "pkcs7")
	if err != nil {
		return nil, err
	}
	padded, err := c.pad(input, blockSize, padding)
	if err != nil {
		return nil, err
	}

	output, buffer := make([]byte, 0), make([]byte, blockSize)
	for i, j := 0, len(padded); i < j; i += blockSize {
		block.Encrypt(buffer, padded[i:i+blockSize])
		output = append(output, buffer...)
	}
	return output, nil
}

func (c *AesEcbCipherClient) Decrypt(input []byte, key []byte, options map[string]interface{}) (builtin.Buffer, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()

	padding, err := read[string](options, "padding", "pkcs7")
	if err != nil {
		return nil, err
	}

	output, buffer := make([]byte, 0), make([]byte, blockSize)
	for i, j := 0, len(input); i < j; i += blockSize {
		block.Decrypt(buffer, input[i:i+blockSize])
		output = append(output, buffer...)
	}
	return c.unpad(output, blockSize, padding)
}

type AesCbcCipherClient struct{ BaseCipherClient }

func (c *AesCbcCipherClient) Encrypt(input []byte, key []byte, options map[string]interface{}) (builtin.Buffer, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()

	iv, err := read[[]byte](options, "iv", nil)
	if err != nil {
		return nil, err
	}
	if len(iv) != blockSize {
		return nil, fmt.Errorf("iv length must be %d bytes, got %d", blockSize, len(iv))
	}

	padding, err := read[string](options, "padding", "pkcs7")
	if err != nil {
		return nil, err
	}
	padded, err := c.pad(input, blockSize, padding)
	if err != nil {
		return nil, err
	}

	output := make([]byte, len(padded))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(output, padded)
	return output, nil
}

func (c *AesCbcCipherClient) Decrypt(input []byte, key []byte, options map[string]interface{}) (builtin.Buffer, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()

	iv, err := read[[]byte](options, "iv", nil)
	if err != nil {
		return nil, err
	}
	if len(iv) != blockSize {
		return nil, fmt.Errorf("iv length must be %d bytes, got %d", blockSize, len(iv))
	}
	if len(input)%blockSize != 0 {
		return nil, fmt.Errorf("ciphertext is not a multiple of the block size")
	}

	padding, err := read[string](options, "padding", "pkcs7")
	if err != nil {
		return nil, err
	}

	output := make([]byte, len(input))
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(output, input)
	return c.unpad(output, blockSize, padding)
}

type AesGcmCipherClient struct{ BaseCipherClient }

func (c *AesGcmCipherClient) Encrypt(input []byte, key []byte, options map[string]interface{}) (builtin.Buffer, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce, err := read[[]byte](options, "nonce", nil)
	if err != nil {
		return nil, err
	}
	if nonce == nil {
		return nil, fmt.Errorf("nonce is required")
	}
	if len(nonce) != gcm.NonceSize() {
		return nil, fmt.Errorf("nonce length must be %d bytes, got %d", gcm.NonceSize(), len(nonce))
	}

	output := gcm.Seal(nil, nonce, input, nil)
	return output, nil
}

func (c *AesGcmCipherClient) Decrypt(input []byte, key []byte, options map[string]interface{}) (builtin.Buffer, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce, err := read[[]byte](options, "nonce", nil)
	if err != nil {
		return nil, err
	}
	if nonce == nil {
		return nil, fmt.Errorf("nonce is required")
	}
	if len(nonce) != gcm.NonceSize() {
		return nil, fmt.Errorf("nonce length must be %d bytes, got %d", gcm.NonceSize(), len(nonce))
	}

	return gcm.Open(nil, nonce, input, nil)
}

type Sm4EcbCipherClient struct{ BaseCipherClient }

func (c *Sm4EcbCipherClient) Encrypt(input []byte, key []byte, options map[string]interface{}) (builtin.Buffer, error) {
	block, err := sm4.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()

	padding, err := read[string](options, "padding", "pkcs7")
	if err != nil {
		return nil, err
	}
	padded, err := c.pad(input, blockSize, padding)
	if err != nil {
		return nil, err
	}

	output := make([]byte, len(padded))
	buf := make([]byte, blockSize)
	for i := 0; i < len(padded); i += blockSize {
		block.Encrypt(buf, padded[i:i+blockSize])
		copy(output[i:], buf)
	}
	return output, nil
}

func (c *Sm4EcbCipherClient) Decrypt(input []byte, key []byte, options map[string]interface{}) (builtin.Buffer, error) {
	block, err := sm4.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()

	padding, err := read[string](options, "padding", "pkcs7")
	if err != nil {
		return nil, err
	}
	if len(input)%blockSize != 0 {
		return nil, fmt.Errorf("ciphertext is not a multiple of the block size")
	}

	output := make([]byte, len(input))
	buf := make([]byte, blockSize)
	for i := 0; i < len(input); i += blockSize {
		block.Decrypt(buf, input[i:i+blockSize])
		copy(output[i:], buf)
	}
	return c.unpad(output, blockSize, padding)
}

type Sm4CbcCipherClient struct{ BaseCipherClient }

func (c *Sm4CbcCipherClient) Encrypt(input []byte, key []byte, options map[string]interface{}) (builtin.Buffer, error) {
	block, err := sm4.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()

	padding, err := read[string](options, "padding", "pkcs7")
	if err != nil {
		return nil, err
	}
	padded, err := c.pad(input, blockSize, padding)
	if err != nil {
		return nil, err
	}

	iv, err := read[[]byte](options, "iv", nil)
	if err != nil {
		return nil, err
	}
	if len(iv) != blockSize {
		return nil, fmt.Errorf("iv length must be %d bytes, got %d", blockSize, len(iv))
	}

	output := make([]byte, len(padded))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(output, padded)
	return output, nil
}

func (c *Sm4CbcCipherClient) Decrypt(input []byte, key []byte, options map[string]interface{}) (builtin.Buffer, error) {
	block, err := sm4.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()

	padding, err := read[string](options, "padding", "pkcs7")
	if err != nil {
		return nil, err
	}

	iv, err := read[[]byte](options, "iv", nil)
	if err != nil {
		return nil, err
	}
	if len(iv) != blockSize {
		return nil, fmt.Errorf("iv length must be %d bytes, got %d", blockSize, len(iv))
	}
	if len(input)%blockSize != 0 {
		return nil, fmt.Errorf("ciphertext is not a multiple of the block size")
	}

	output := make([]byte, len(input))
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(output, input)
	return c.unpad(output, blockSize, padding)
}

//#endregion

//#region Hash & Hmac

type CryptoHashClient struct {
	hash crypto.Hash
}

func (c *CryptoHashClient) Sum(input []byte) builtin.Buffer {
	h := c.hash.New()
	h.Write(input)
	return h.Sum(nil)
}

type CryptoSm3HashClient struct{}

func (c *CryptoSm3HashClient) Sum(input []byte) builtin.Buffer {
	h := sm3.New()
	h.Write(input)
	return h.Sum(nil)
}

type CryptoHmacClient struct {
	hash crypto.Hash
}

func (c *CryptoHmacClient) Sum(input []byte, key []byte) builtin.Buffer {
	h := hmac.New(c.hash.New, key)
	h.Write(input)
	return h.Sum(nil)
}

type CryptoSm3HmacClient struct{}

func (c *CryptoSm3HmacClient) Sum(input []byte, key []byte) builtin.Buffer {
	h := hmac.New(sm3.New, key)
	h.Write(input)
	return h.Sum(nil)
}

//#endregion

//#region RSA

type CryptoRsaClient struct{}

func (c *CryptoRsaClient) GenerateKey(length int) (*map[string]builtin.Buffer, error) {
	if length == 0 {
		length = 2048
	}
	privateKey, err := rsa.GenerateKey(rand.Reader, length)
	if err != nil {
		return nil, err
	}
	derStream := x509.MarshalPKCS1PrivateKey(privateKey)
	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: derStream,
	}
	prvkey := pem.EncodeToMemory(block)
	publicKey := &privateKey.PublicKey
	derPubStream := x509.MarshalPKCS1PublicKey(publicKey)
	pubKey := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: derPubStream,
	})
	return &map[string]builtin.Buffer{
		"privateKey": prvkey,
		"publicKey":  pubKey,
	}, nil
}

func (c *CryptoRsaClient) Encrypt(input []byte, key []byte, padding string) (builtin.Buffer, error) {
	block, _ := pem.Decode(key)
	if block == nil {
		return nil, fmt.Errorf("public key is invalid")
	}
	publicKey, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	if padding == "oaep" {
		return rsa.EncryptOAEP(sha256.New(), rand.Reader, publicKey, input, nil)
	}
	return rsa.EncryptPKCS1v15(rand.Reader, publicKey, input)
}

func (c *CryptoRsaClient) Decrypt(input []byte, key []byte, padding string) (builtin.Buffer, error) {
	block, _ := pem.Decode(key)
	if block == nil {
		return nil, fmt.Errorf("private key is invalid")
	}
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	if padding == "oaep" {
		return rsa.DecryptOAEP(sha256.New(), rand.Reader, privateKey, input, nil)
	}
	return rsa.DecryptPKCS1v15(rand.Reader, privateKey, input)
}

func (c *CryptoRsaClient) Sign(input []byte, key []byte, algorithm string, padding string) (builtin.Buffer, error) {
	hash, err := toHash(algorithm)
	if err != nil {
		return nil, err
	}
	h := hash.New()
	h.Write(input)
	digest := h.Sum(nil)
	block, _ := pem.Decode(key)
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	if padding == "pss" {
		return rsa.SignPSS(rand.Reader, privateKey, hash, digest, &rsa.PSSOptions{
			SaltLength: rsa.PSSSaltLengthEqualsHash,
		})
	}
	return rsa.SignPKCS1v15(nil, privateKey, hash, digest)
}

func (c *CryptoRsaClient) Verify(input []byte, sign []byte, key []byte, algorithm string, padding string) (bool, error) {
	block, _ := pem.Decode(key)
	if block == nil {
		return false, fmt.Errorf("public key is invalid")
	}
	publicKey, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return false, err
	}
	hash, err := toHash(algorithm)
	if err != nil {
		return false, err
	}
	h := hash.New()
	h.Write(input)
	digest := h.Sum(nil)
	if padding == "pss" {
		if err = rsa.VerifyPSS(publicKey, hash, digest[:], sign, nil); err != nil {
			return false, nil
		}
	} else {
		if err = rsa.VerifyPKCS1v15(publicKey, hash, digest[:], sign); err != nil {
			return false, nil
		}
	}
	return true, nil
}

//#endregion

//#region SM2

type CryptoSm2Client struct{}

func (c *CryptoSm2Client) GenerateKey() (*map[string]builtin.Buffer, error) {
	privateKey, err := sm2.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	prvkey := privateKey.D.Bytes()
	if len(prvkey) < 32 {
		padded := make([]byte, 32)
		copy(padded[32-len(prvkey):], prvkey)
		prvkey = padded
	}
	pubkey := elliptic.MarshalCompressed(sm2.P256(), privateKey.PublicKey.X, privateKey.PublicKey.Y)
	return &map[string]builtin.Buffer{
		"privateKey": prvkey,
		"publicKey":  pubkey,
	}, nil
}

func (c *CryptoSm2Client) Encrypt(input []byte, key []byte) (builtin.Buffer, error) {
	publicKey, err := c.toPublicKey(key)
	if err != nil {
		return nil, err
	}
	return sm2.EncryptASN1(rand.Reader, publicKey, input)
}

func (c *CryptoSm2Client) Decrypt(input []byte, key []byte) (builtin.Buffer, error) {
	privateKey, err := c.toPrivateKey(key)
	if err != nil {
		return nil, err
	}
	return sm2.Decrypt(privateKey, input)
}

func (c *CryptoSm2Client) Sign(input []byte, key []byte) (builtin.Buffer, error) {
	privateKey, err := c.toPrivateKey(key)
	if err != nil {
		return nil, err
	}
	return sm2.SignASN1(rand.Reader, privateKey, input, &sm2.SM2SignerOption{})
}

func (c *CryptoSm2Client) Verify(input []byte, sign []byte, key []byte) (bool, error) {
	publicKey, err := c.toPublicKey(key)
	if err != nil {
		return false, err
	}
	if !sm2.VerifyASN1(publicKey, input, sign) {
		return false, nil
	}
	return true, nil
}

func (c *CryptoSm2Client) toPrivateKey(key []byte) (*sm2.PrivateKey, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("private key must be 32 bytes")
	}
	return sm2.NewPrivateKey(key)
}

func (c *CryptoSm2Client) toPublicKey(key []byte) (*ecdsa.PublicKey, error) {
	if len(key) == 33 {
		x, y := elliptic.UnmarshalCompressed(sm2.P256(), key)
		if x == nil {
			return nil, fmt.Errorf("invalid compressed public key")
		}
		return &ecdsa.PublicKey{
			Curve: sm2.P256(),
			X:     x,
			Y:     y,
		}, nil
	}
	if len(key) == 65 {
		return sm2.ParseUncompressedPublicKey(key)
	}
	return nil, fmt.Errorf("public key must be 33 bytes(compressed) or 65 bytes(uncompressed)")
}

//#endregion

func toHash(algorithm string) (crypto.Hash, error) {
	switch strings.ToLower(algorithm) {
	case "md5":
		return crypto.MD5, nil
	case "sha1":
		return crypto.SHA1, nil
	case "sha256":
		return crypto.SHA256, nil
	case "sha512":
		return crypto.SHA512, nil
	default:
		return crypto.SHA256, fmt.Errorf("hash algorithm %s is not supported", algorithm)
	}
}

func read[T any](options map[string]interface{}, key string, dvalue T) (T, error) {
	if options == nil {
		return dvalue, nil
	}
	option, ok := options[key]
	if !ok {
		return dvalue, nil
	}

	var n T

	switch any(dvalue).(type) {
	case string:
		switch v := option.(type) {
		case string:
			return any(v).(T), nil
		case *builtin.Buffer:
			return any(string(*v)).(T), nil
		default:
			return n, fmt.Errorf("invalid option %s", key)
		}
	case []byte:
		switch v := option.(type) {
		case string:
			return any([]byte(v)).(T), nil
		case []byte:
			return any(v).(T), nil
		case builtin.Buffer:
			return any([]byte(v)).(T), nil
		case *builtin.Buffer:
			return any([]byte(*v)).(T), nil
		default:
			return n, fmt.Errorf("invalid option %s", key)
		}
	default:
		return n, fmt.Errorf("invalid type of option %s", key)
	}
}

type CryptoClient struct{}

func (c *CryptoClient) CreateCipher(algorithm string) (CryptoCipherClient, error) {
	switch strings.ToLower(algorithm) {
	case "aes-ecb":
		return &AesEcbCipherClient{}, nil
	case "aes-cbc":
		return &AesCbcCipherClient{}, nil
	case "aes-gcm":
		return &AesGcmCipherClient{}, nil
	case "sm4-ecb":
		return &Sm4EcbCipherClient{}, nil
	case "sm4-cbc":
		return &Sm4CbcCipherClient{}, nil
	default:
		return nil, fmt.Errorf("cipher algorithm %s is not supported", algorithm)
	}
}

func (c *CryptoClient) CreateHash(algorithm string) (interface{}, error) {
	switch strings.ToLower(algorithm) {
	case "sm3":
		return &CryptoSm3HashClient{}, nil
	default:
		hash, err := toHash(algorithm)
		if err != nil {
			return nil, err
		}
		return &CryptoHashClient{hash: hash}, nil
	}
}

func (c *CryptoClient) CreateHmac(algorithm string) (interface{}, error) {
	switch strings.ToLower(algorithm) {
	case "sm3":
		return &CryptoSm3HmacClient{}, nil
	default:
		hash, err := toHash(algorithm)
		if err != nil {
			return nil, err
		}
		return &CryptoHmacClient{hash: hash}, nil
	}
}

func (c *CryptoClient) CreateRsa() *CryptoRsaClient {
	return &CryptoRsaClient{}
}

func (c *CryptoClient) CreateSm2() *CryptoSm2Client {
	return &CryptoSm2Client{}
}
