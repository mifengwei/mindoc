package routers

import (
	"bytes"
	"html"
	"regexp"
	"strings"
)

// HTML2Str 将 HTML 转换为纯文本
func HTML2Str(s string) string {
	re := regexp.MustCompile(`<[\s\S]*?>`)
	s = re.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	s = strings.ReplaceAll(s, "&#160;", " ")
	return strings.TrimSpace(s)
}

// Htmlquote HTML 转义
func Htmlquote(s string) string {
	return html.EscapeString(s)
}

// Htmlunquote HTML 反转义
func Htmlunquote(s string) string {
	return html.UnescapeString(s)
}

// Htmlfilter 过滤 HTML 标签，保留安全内容
func Htmlfilter(s string) string {
	re := regexp.MustCompile(`<script[\s\S]*?</script>`)
	s = re.ReplaceAllString(s, "")
	re = regexp.MustCompile(`<style[\s\S]*?</style>`)
	s = re.ReplaceAllString(s, "")
	return s
}

// HTML2StrBytes HTML2Str 的 bytes 版本
func HTML2StrBytes(b []byte) string {
	return HTML2Str(string(b))
}

// BytesToString 安全转换
func BytesToString(b []byte) string {
	return string(b)
}

// StringToBytes 安全转换
func StringToBytes(s string) []byte {
	return bytes.TrimSpace([]byte(s))
}
