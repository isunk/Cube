package module

import (
	"bytes"
	"testing"
)

func TestCipher(t *testing.T) {
	input, key := []byte("hello, world"), []byte("0123456789012345")
	options := map[string]interface{}{
		"padding": "pkcs7",
		"iv":      []byte("0123456789012345"),
		"nonce":   []byte("012345678901"),
	}

	// AES-ECB
	{
		cipher, _ := (&CryptoClient{}).CreateCipher("aes-ecb")
		a, err := cipher.Encrypt(input, key, options)
		if err != nil {
			t.Fatal(err)
		}

		h, _ := a.ToString("hex")
		if h != "e7bee814a4b9f3e7a5874e604302a32a" {
			t.Fatal("unexpected encryption")
		}

		b, err := cipher.Decrypt(a, key, options)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(b, input) {
			t.Fatal("unexpected decryption")
		}
	}

	// AES-CBC
	{
		cipher, _ := (&CryptoClient{}).CreateCipher("aes-cbc")
		a, err := cipher.Encrypt(input, key, options)
		if err != nil {
			t.Fatal(err)
		}

		h, _ := a.ToString("hex")
		if h != "d31c9075f9143908466fdcb2d11d3bfe" {
			t.Fatal("unexpected encryption")
		}

		b, err := cipher.Decrypt(a, key, options)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(b, input) {
			t.Fatal("unexpected decryption")
		}
	}

	// AES-GCM
	{
		cipher, _ := (&CryptoClient{}).CreateCipher("aes-gcm")
		a, err := cipher.Encrypt(input, key, options)
		if err != nil {
			t.Fatal(err)
		}

		h, _ := a.ToString("hex")
		if h != "73ccbd706b799a5494b7081c578c1052373694fba6210e4f8f7f5b65" {
			t.Fatal("unexpected encryption")
		}

		b, err := cipher.Decrypt(a, key, options)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(b, input) {
			t.Fatal("unexpected decryption")
		}
	}

	// SM4-ECB
	{
		cipher, _ := (&CryptoClient{}).CreateCipher("sm4-ecb")
		a, err := cipher.Encrypt(input, key, options)
		if err != nil {
			t.Fatal(err)
		}

		h, _ := a.ToString("hex")
		if h != "01be5a5117754b58b5bc57e79ad7ba02" {
			t.Fatal("unexpected encryption")
		}

		b, err := cipher.Decrypt(a, key, options)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(b, input) {
			t.Fatal("unexpected decryption")
		}
	}

	// SM4-CBC
	{
		cipher, _ := (&CryptoClient{}).CreateCipher("sm4-cbc")
		a, err := cipher.Encrypt(input, key, options)
		if err != nil {
			t.Fatal(err)
		}

		h, _ := a.ToString("hex")
		if h != "a708548e516e5fd54633e61e089a6e42" {
			t.Fatal("unexpected encryption")
		}

		b, err := cipher.Decrypt(a, key, options)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(b, input) {
			t.Fatal("unexpected decryption")
		}
	}
}

func TestHash(t *testing.T) {
	input := []byte("hello, world")

	// MD5
	{
		hash, _ := (&CryptoClient{}).CreateHash("md5")
		a := hash.(*CryptoHashClient).Sum(input)
		h, _ := a.ToString("hex")
		if h != "e4d7f1b4ed2e42d15898f4b27b019da4" {
			t.Fatal("unexpected sum")
		}
	}

	// SHA1
	{
		hash, _ := (&CryptoClient{}).CreateHash("sha1")
		a := hash.(*CryptoHashClient).Sum(input)
		h, _ := a.ToString("hex")
		if h != "b7e23ec29af22b0b4e41da31e868d57226121c84" {
			t.Fatal("unexpected sum")
		}
	}

	// SHA256
	{
		hash, _ := (&CryptoClient{}).CreateHash("sha256")
		a := hash.(*CryptoHashClient).Sum(input)
		h, _ := a.ToString("hex")
		if h != "09ca7e4eaa6e8ae9c7d261167129184883644d07dfba7cbfbc4c8a2e08360d5b" {
			t.Fatal("unexpected sum")
		}
	}

	// SHA512
	{
		hash, _ := (&CryptoClient{}).CreateHash("sha512")
		a := hash.(*CryptoHashClient).Sum(input)
		h, _ := a.ToString("hex")
		if h != "8710339dcb6814d0d9d2290ef422285c9322b7163951f9a0ca8f883d3305286f44139aa374848e4174f5aada663027e4548637b6d19894aec4fb6c46a139fbf9" {
			t.Fatal("unexpected sum")
		}
	}

	// SM3
	{
		hash, _ := (&CryptoClient{}).CreateHash("sm3")
		a := hash.(*CryptoSm3HashClient).Sum(input)
		h, _ := a.ToString("hex")
		if h != "02df30dff15f2ccb72bffdcb44e68d4d09974036dc7a6927e556fbef421c7f34" {
			t.Fatal("unexpected sum")
		}
	}
}

func TestHmac(t *testing.T) {
	input, key := []byte("hello, world"), []byte("0123456789012345")

	// MD5
	{
		hash, _ := (&CryptoClient{}).CreateHmac("md5")
		a := hash.(*CryptoHmacClient).Sum(input, key)
		h, _ := a.ToString("hex")
		if h != "e83943f62bcb19e1d8ac0e7885250189" {
			t.Fatal("unexpected sum")
		}
	}

	// SHA1
	{
		hash, _ := (&CryptoClient{}).CreateHmac("sha1")
		a := hash.(*CryptoHmacClient).Sum(input, key)
		h, _ := a.ToString("hex")
		if h != "a8f37017b38a816f35c5577a4069453b1375a259" {
			t.Fatal("unexpected sum")
		}
	}

	// SHA256
	{
		hash, _ := (&CryptoClient{}).CreateHmac("sha256")
		a := hash.(*CryptoHmacClient).Sum(input, key)
		h, _ := a.ToString("hex")
		if h != "d37d0d896f2b787298c4a700612d7f40a8cfe0146cfc4710739b829bb5ac1dcf" {
			t.Fatal("unexpected sum")
		}
	}

	// SHA512
	{
		hash, _ := (&CryptoClient{}).CreateHmac("sha512")
		a := hash.(*CryptoHmacClient).Sum(input, key)
		h, _ := a.ToString("hex")
		if h != "90026d53574b7436b80b7a0009561496bf110e31f96a2e021e75b24a9d7c05d69b6b3ecffb5f1ee78cb4afd3ac7ba61a2531d1f5a0dbba5d274e03ac1b94b82e" {
			t.Fatal("unexpected sum")
		}
	}

	// SM3
	{
		hash, _ := (&CryptoClient{}).CreateHmac("sm3")
		a := hash.(*CryptoSm3HmacClient).Sum(input, key)
		h, _ := a.ToString("hex")
		if h != "4cc0ca38c6d3824977ffba1243f1986f4fd44bea5be94c8720f0b5c0081a72df" {
			t.Fatal("unexpected sum")
		}
	}
}

func BenchmarkHash(b *testing.B) {
	hash, _ := (&CryptoClient{}).CreateHash("sha256")
	h := hash.(*CryptoHashClient)

	for n := 0; n < b.N; n++ {
		h.Sum([]byte("hello, world"))
	}
}

func TestRsa(t *testing.T) {
	input := []byte("hello, world")

	client := (&CryptoClient{}).CreateRsa()

	keys, err := client.GenerateKey(2048)
	if err != nil {
		t.Fatal(err)
	}

	a, err := client.Encrypt(input, (*keys)["publicKey"], "pkcs1")
	if err != nil {
		t.Fatal(err)
	}

	b, err := client.Decrypt(a, (*keys)["privateKey"], "pkcs1")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b, input) {
		t.Fatal("unexpected decryption")
	}

	c, err := client.Sign(input, (*keys)["privateKey"], "sha256", "pkcs1")
	if err != nil {
		t.Fatal(err)
	}

	d, err := client.Verify(input, c, (*keys)["publicKey"], "sha256", "pkcs1")
	if err != nil {
		t.Fatal(err)
	}
	if !d {
		t.Fatal("unexpected verify")
	}
}

func TestSm2(t *testing.T) {
	input := []byte("hello, world")

	client := (&CryptoClient{}).CreateSm2()

	keys, err := client.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}

	a, err := client.Encrypt(input, (*keys)["publicKey"])
	if err != nil {
		t.Fatal(err)
	}

	b, err := client.Decrypt(a, (*keys)["privateKey"])
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b, input) {
		t.Fatal("unexpected decryption")
	}

	c, err := client.Sign(input, (*keys)["privateKey"])
	if err != nil {
		t.Fatal(err)
	}

	d, err := client.Verify(input, c, (*keys)["publicKey"])
	if err != nil {
		t.Fatal(err)
	}
	if !d {
		t.Fatal("unexpected verify")
	}
}
