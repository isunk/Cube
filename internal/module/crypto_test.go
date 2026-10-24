package module

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestCipher(t *testing.T) {
	input, key := []byte("hello, world"), []byte("0123456789012345")
	options := map[string]interface{}{
		"padding": "pkcs7",
		"iv":      []byte("0123456789012345"),
		"nonce":   []byte("012345678901"),
	}

	t.Run("AES-ECB", func(t *testing.T) {
		cipher, _ := (&CryptoClient{}).CreateCipher("aes-ecb")
		a, err := cipher.Encrypt(input, key, options)
		if err != nil {
			t.Fatal(err)
		}
		h, _ := a.ToString("hex")
		if h != "e7bee814a4b9f3e7a5874e604302a32a" {
			t.Fatalf("encryption failed: %s", h)
		}
		b, err := cipher.Decrypt(a, key, options)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(b, input) {
			t.Fatal("decryption failed")
		}
	})

	t.Run("AES-CBC", func(t *testing.T) {
		cipher, _ := (&CryptoClient{}).CreateCipher("aes-cbc")
		a, err := cipher.Encrypt(input, key, options)
		if err != nil {
			t.Fatal(err)
		}
		h, _ := a.ToString("hex")
		if h != "d31c9075f9143908466fdcb2d11d3bfe" {
			t.Fatalf("encryption failed: %s", h)
		}
		b, err := cipher.Decrypt(a, key, options)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(b, input) {
			t.Fatal("decryption failed")
		}
	})

	t.Run("AES-GCM", func(t *testing.T) {
		cipher, _ := (&CryptoClient{}).CreateCipher("aes-gcm")
		a, err := cipher.Encrypt(input, key, options)
		if err != nil {
			t.Fatal(err)
		}
		h, _ := a.ToString("hex")
		if h != "73ccbd706b799a5494b7081c578c1052373694fba6210e4f8f7f5b65" {
			t.Fatalf("encryption failed: %s", h)
		}
		b, err := cipher.Decrypt(a, key, options)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(b, input) {
			t.Fatal("decryption failed")
		}
	})

	t.Run("SM4-ECB", func(t *testing.T) {
		cipher, _ := (&CryptoClient{}).CreateCipher("sm4-ecb")
		a, err := cipher.Encrypt(input, key, options)
		if err != nil {
			t.Fatal(err)
		}
		h, _ := a.ToString("hex")
		if h != "01be5a5117754b58b5bc57e79ad7ba02" {
			t.Fatalf("encryption failed: %s", h)
		}
		b, err := cipher.Decrypt(a, key, options)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(b, input) {
			t.Fatal("decryption failed")
		}
	})

	t.Run("SM4-CBC", func(t *testing.T) {
		cipher, _ := (&CryptoClient{}).CreateCipher("sm4-cbc")
		a, err := cipher.Encrypt(input, key, options)
		if err != nil {
			t.Fatal(err)
		}
		h, _ := a.ToString("hex")
		if h != "a708548e516e5fd54633e61e089a6e42" {
			t.Fatalf("encryption failed: %s", h)
		}
		b, err := cipher.Decrypt(a, key, options)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(b, input) {
			t.Fatal("decryption failed")
		}
	})
}

func TestHash(t *testing.T) {
	input := []byte("hello, world")

	t.Run("MD5", func(t *testing.T) {
		hash, _ := (&CryptoClient{}).CreateHash("md5")
		buf := hash.(*CryptoHashClient).Sum(input)
		h, _ := buf.ToString("hex")
		if h != "e4d7f1b4ed2e42d15898f4b27b019da4" {
			t.Fatalf("hash failed: %s", h)
		}
	})

	t.Run("SHA1", func(t *testing.T) {
		hash, _ := (&CryptoClient{}).CreateHash("sha1")
		buf := hash.(*CryptoHashClient).Sum(input)
		h, _ := buf.ToString("hex")
		if h != "b7e23ec29af22b0b4e41da31e868d57226121c84" {
			t.Fatalf("hash failed: %s", h)
		}
	})

	t.Run("SHA256", func(t *testing.T) {
		hash, _ := (&CryptoClient{}).CreateHash("sha256")
		buf := hash.(*CryptoHashClient).Sum(input)
		h, _ := buf.ToString("hex")
		if h != "09ca7e4eaa6e8ae9c7d261167129184883644d07dfba7cbfbc4c8a2e08360d5b" {
			t.Fatalf("hash failed: %s", h)
		}
	})

	t.Run("SHA512", func(t *testing.T) {
		hash, _ := (&CryptoClient{}).CreateHash("sha512")
		buf := hash.(*CryptoHashClient).Sum(input)
		h, _ := buf.ToString("hex")
		if h != "8710339dcb6814d0d9d2290ef422285c9322b7163951f9a0ca8f883d3305286f44139aa374848e4174f5aada663027e4548637b6d19894aec4fb6c46a139fbf9" {
			t.Fatalf("hash failed: %s", h)
		}
	})

	t.Run("SM3", func(t *testing.T) {
		hash, _ := (&CryptoClient{}).CreateHash("sm3")
		buf := hash.(*CryptoSm3HashClient).Sum(input)
		h, _ := buf.ToString("hex")
		if h != "02df30dff15f2ccb72bffdcb44e68d4d09974036dc7a6927e556fbef421c7f34" {
			t.Fatalf("hash failed: %s", h)
		}
	})
}

func TestHmac(t *testing.T) {
	input, key := []byte("hello, world"), []byte("0123456789012345")

	t.Run("MD5", func(t *testing.T) {
		hash, _ := (&CryptoClient{}).CreateHmac("md5")
		buf := hash.(*CryptoHmacClient).Sum(input, key)
		h, _ := buf.ToString("hex")
		if h != "e83943f62bcb19e1d8ac0e7885250189" {
			t.Fatalf("hmac failed: %s", h)
		}
	})

	t.Run("SHA1", func(t *testing.T) {
		hash, _ := (&CryptoClient{}).CreateHmac("sha1")
		buf := hash.(*CryptoHmacClient).Sum(input, key)
		h, _ := buf.ToString("hex")
		if h != "a8f37017b38a816f35c5577a4069453b1375a259" {
			t.Fatalf("hmac failed: %s", h)
		}
	})

	t.Run("SHA256", func(t *testing.T) {
		hash, _ := (&CryptoClient{}).CreateHmac("sha256")
		buf := hash.(*CryptoHmacClient).Sum(input, key)
		h, _ := buf.ToString("hex")
		if h != "d37d0d896f2b787298c4a700612d7f40a8cfe0146cfc4710739b829bb5ac1dcf" {
			t.Fatalf("hmac failed: %s", h)
		}
	})

	t.Run("SHA512", func(t *testing.T) {
		hash, _ := (&CryptoClient{}).CreateHmac("sha512")
		buf := hash.(*CryptoHmacClient).Sum(input, key)
		h, _ := buf.ToString("hex")
		if h != "90026d53574b7436b80b7a0009561496bf110e31f96a2e021e75b24a9d7c05d69b6b3ecffb5f1ee78cb4afd3ac7ba61a2531d1f5a0dbba5d274e03ac1b94b82e" {
			t.Fatalf("hmac failed: %s", h)
		}
	})

	t.Run("SM3", func(t *testing.T) {
		hash, _ := (&CryptoClient{}).CreateHmac("sm3")
		buf := hash.(*CryptoSm3HmacClient).Sum(input, key)
		h, _ := buf.ToString("hex")
		if h != "4cc0ca38c6d3824977ffba1243f1986f4fd44bea5be94c8720f0b5c0081a72df" {
			t.Fatalf("hmac failed: %s", h)
		}
	})
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
	prvkey, pubkey := (*keys)["privateKey"], (*keys)["publicKey"]

	t.Run("加密解密", func(t *testing.T) {
		a, err := client.Encrypt(input, pubkey, "pkcs1")
		if err != nil {
			t.Fatal(err)
		}
		b, err := client.Decrypt(a, prvkey, "pkcs1")
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(b, input) {
			t.Fatal("decryption failed")
		}
	})

	t.Run("签名验证", func(t *testing.T) {
		c, err := client.Sign(input, prvkey, "sha256", "pkcs1")
		if err != nil {
			t.Fatal(err)
		}
		d, err := client.Verify(input, c, pubkey, "sha256", "pkcs1")
		if err != nil {
			t.Fatal(err)
		}
		if !d {
			t.Fatal("verify failed")
		}
	})
}

func TestSm2(t *testing.T) {
	input := []byte("hello, world")
	client := (&CryptoClient{}).CreateSm2()

	t.Run("生成密钥加解密", func(t *testing.T) {
		keys, err := client.GenerateKey()
		if err != nil {
			t.Fatal(err)
		}
		prvkey, pubkey := (*keys)["privateKey"], (*keys)["publicKey"]
		a, err := client.Encrypt(input, pubkey, nil)
		if err != nil {
			t.Fatal(err)
		}
		b, err := client.Decrypt(a, prvkey, nil)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(b, input) {
			t.Fatal("decryption failed")
		}
	})

	t.Run("解密（C1C2C3）", func(t *testing.T) {
		prvkey, _ := hex.DecodeString("49601eb539c8e72dd31a2e19622a8e4e70b7879f35eb5d37cf08aac7a7996220")
		a, _ := hex.DecodeString("8fe38c33f0ece39e55740ae822657fb3b058eba67ca2e44f2bd87d3876e67ee5780bd1ab5ebd06f06b7bfecf930ae59aafb1e78ae776c2946dc0b9d17a50aef6813674dd756ffbd324040b9f5f31f907650c2eb0f1711ef7cf7f1e84c4255763f43687c01530c11240f02fc2")
		b, err := client.Decrypt(a, prvkey, map[string]interface{}{"encoding": "c1c2c3"})
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(b, input) {
			t.Fatal("decryption failed")
		}
	})

	t.Run("解密（C1C3C2）", func(t *testing.T) {
		prvkey, _ := hex.DecodeString("49601eb539c8e72dd31a2e19622a8e4e70b7879f35eb5d37cf08aac7a7996220")
		a, _ := hex.DecodeString("347641bb934912e8fe68ac4d0a1e69c16aa206c9547a47042c7c2d36222723ab6ad53f48b3c820f405b00a4f0be2a9b57e2d847be6d6e6fbd92d05b308c1262c6479c866f6508730764cb1822f17061fe47a5bfe2b5e714ba51ca78dafe6b86f2ef1c892eed8c6496da359b9")
		b, err := client.Decrypt(a, prvkey, nil)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(b, input) {
			t.Fatal("decryption failed")
		}
	})

	t.Run("验签（默认 Raw，无杂凑）", func(t *testing.T) {
		pubkey, _ := hex.DecodeString("04b4b578c390043086d3039f910c8718e3ee6525bfec083563ee1784b7a9d849b168470ec465f0c5b8827f60d1b78e68d33b884fcc2294ae408911c48c35d2510e")
		c, _ := hex.DecodeString("1f09fcfe63aaf0c5bfe73fdb9209ea570c0e69b3e5d68a2428447430060ec48da457dc21d0d81731fea8570d2715b16f83a16f8dd7aa3b91c2447d8592dd12be")
		d, err := client.Verify(input, c, pubkey, nil)
		if err != nil {
			t.Fatal(err)
		}
		if !d {
			t.Fatal("verify failed")
		}
	})

	t.Run("验签（ASN1 编码 + SM3 杂凑 & UID=0123456789012345）", func(t *testing.T) {
		pubkey, _ := hex.DecodeString("04b4b578c390043086d3039f910c8718e3ee6525bfec083563ee1784b7a9d849b168470ec465f0c5b8827f60d1b78e68d33b884fcc2294ae408911c48c35d2510e")
		c, _ := hex.DecodeString("304602210082e779ea88fb411d395975f57b7fad985c64c95f78eaa1088e0eb70e9d06e40e022100b78642f810f011340252833beb5c882d18e7997be59b8e9453065a69e7d2aebc")
		d, err := client.Verify(input, c, pubkey, map[string]interface{}{"format": "asn1", "hash": "sm3", "uid": []byte("0123456789012345")})
		if err != nil {
			t.Fatal(err)
		}
		if !d {
			t.Fatal("verify failed")
		}
	})
}
