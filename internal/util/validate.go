package util

import (
	"errors"
	"regexp"
)

// ValidateString 校验 input 是否为匹配正则 regex 的字符串，不匹配时返回 message
func ValidateString(input interface{}, regex *regexp.Regexp, message string) error {
	s, ok := input.(string)
	if !ok || (regex != nil && !regex.MatchString(s)) {
		return errors.New(message)
	}
	return nil
}
