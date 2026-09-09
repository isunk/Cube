package util

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

var REGEX_DIGEST_PARAM = regexp.MustCompile(`(\w+)="?([^",]*)`) // Digest 认证参数正则

type QueryParams struct {
	url.Values
}

func (p *QueryParams) GetOrDefault(key string, defaultValue string) string {
	if value := p.Get(key); value != "" {
		return value
	}
	return defaultValue
}

func (p *QueryParams) GetIntOrDefault(key string, defaultValue int) int {
	if !p.Has(key) {
		return defaultValue
	}
	if value, err := strconv.Atoi(p.Get(key)); err == nil {
		return value
	}
	return defaultValue
}

type DigestAuth struct{}

func (a *DigestAuth) parse(input string) map[string]string {
	matches := REGEX_DIGEST_PARAM.FindAllStringSubmatch(input, -1)
	if len(matches) == 0 {
		return nil
	}

	params := make(map[string]string)
	for _, match := range matches {
		params[match[1]] = match[2]
	}

	return params
}

func (a *DigestAuth) md5(input string) string {
	h := md5.Sum([]byte(input))
	return hex.EncodeToString(h[:])
}

func (a *DigestAuth) VerifyWithMd5(input string, method string, userpass string) bool {
	p := a.parse(input)
	if p == nil {
		return false
	}

	// 用 SplitN 限制为 2 段，避免密码中含 ":" 被截断，避免缺 ":" 时越界
	parts := strings.SplitN(userpass, ":", 2)
	if len(parts) < 2 {
		return false
	}
	username, password := parts[0], parts[1]

	if p["username"] != username {
		return false
	}

	h1 := a.md5(p["username"] + ":" + p["realm"] + ":" + password)
	h2 := a.md5(method + ":" + p["uri"])
	re := a.md5(h1 + ":" + p["nonce"] + ":" + p["nc"] + ":" + p["cnonce"] + ":" + p["qop"] + ":" + h2)

	return p["response"] == re
}

func (a *DigestAuth) Random(size int) string {
	b := make([]byte, size/2+1)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)[:size]
}

func UnmarshalWithIoReader(r io.Reader, v interface{}) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

func StringWithIoReader(r io.Reader) (string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
