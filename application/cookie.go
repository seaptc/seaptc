package application

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func DecodeStringsFromCookie(values string) ([]string, error) {
	parts := strings.Split(values, "|")
	for i := range parts {
		var err error
		parts[i], err = url.QueryUnescape(parts[i])
		if err != nil {
			return nil, err
		}
	}
	return parts, nil
}

func EncodeStringsForCookie(values ...string) string {
	var buf []byte
	for i, value := range values {
		if i > 0 {
			buf = append(buf, '|')
		}
		for i := 0; i < len(value); i++ {
			b := value[i]
			switch {
			case b == ' ':
				buf = append(buf, '+')
			case // byte values not allowed in cookie value
				b <= ' ' ||
					b >= 127 ||
					b == '"' ||
					b == ',' ||
					b == ';' ||
					b == '\\' ||
					// byte values with special meaning in percent encoding
					b == '+' ||
					b == '%' ||
					// delimiter for separating values
					b == '|':
				buf = append(buf, '%', "0123456789ABCDEF"[b>>4], "0123456789ABCDEF"[b&15])
			default:
				buf = append(buf, b)
			}
		}
	}
	return string(buf)
}

func SignValue(secret string, maxAge int64, value string) string {
	buf := make([]byte, hex.EncodedLen(sha1.Size))
	buf = append(buf, '|')
	buf = strconv.AppendInt(buf, time.Now().Add(time.Second*time.Duration(maxAge)).Unix(), 36)
	buf = append(buf, '|')
	buf = append(buf, value...)
	h := hmac.New(sha1.New, []byte(secret))
	h.Write(buf[hex.EncodedLen(sha1.Size)+1:])
	sum := h.Sum(nil)
	hex.Encode(buf, sum)
	return string(buf)
}

func VerifySignature(secret, value string) (string, bool) {
	sig, value := split(value)

	h := hmac.New(sha1.New, []byte(secret))
	h.Write([]byte(value))
	want := h.Sum(nil)

	got, err := hex.DecodeString(sig)

	if err != nil {
		return "", false
	}
	if !hmac.Equal(got, want) {
		return "", false
	}

	s, value := split(value)
	t, err := strconv.ParseInt(s, 36, 64)
	if err != nil {
		return "", false
	}

	if time.Unix(t, 0).Before(time.Now()) {
		return "", false
	}

	return value, true
}

func split(s string) (string, string) {
	if i := strings.IndexByte(s, '|'); i >= 0 {
		return s[:i], s[i+1:]
	}
	return s, ""
}
