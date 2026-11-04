package domain

import (
	"net/mail"
	"regexp"
	"strings"
)

type Channel string

const (
	ChannelMpesa Channel = "mpesa"
	ChannelEmola Channel = "emola"
	ChannelCard  Channel = "card"
)

func (c Channel) Valid() bool    { return c == ChannelMpesa || c == ChannelEmola || c == ChannelCard }
func (c Channel) IsWallet() bool { return c == ChannelMpesa || c == ChannelEmola }

var nonDigit = regexp.MustCompile(`\D`)

// NormalizeMZPhone returns the 9-digit national number or "".
func NormalizeMZPhone(s string) string {
	d := nonDigit.ReplaceAllString(s, "")
	d = strings.TrimPrefix(d, "258")
	if len(d) != 9 || d[0] != '8' || d[1] < '2' || d[1] > '7' {
		return ""
	}
	return d
}

// WalletOf returns the mobile wallet that serves a national number.
func WalletOf(national string) Channel {
	if len(national) < 2 {
		return ""
	}
	switch national[:2] {
	case "84", "85":
		return ChannelMpesa
	case "86", "87":
		return ChannelEmola
	}
	return ""
}

func FormatMZPhone(national string) string {
	if len(national) != 9 {
		return national
	}
	return "+258 " + national[:2] + " " + national[2:5] + " " + national[5:]
}

func NormalizeEmail(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if len(s) > 254 {
		return ""
	}
	a, err := mail.ParseAddress(s)
	if err != nil || a.Address != s || !strings.Contains(s[strings.LastIndex(s, "@"):], ".") {
		return ""
	}
	return s
}

var handleRe = regexp.MustCompile(`^[a-z0-9.]{3,24}$`)

var reservedHandles = map[string]bool{
	"admin": true, "api": true, "refres": true, "entrar": true, "registar": true, "recuperar": true,
	"painel": true, "app": true, "static": true, "assets": true, "mpesa": true, "emola": true,
	"suporte": true, "ajuda": true, "termos": true, "privacidade": true, "aprovar": true, "www": true,
}

func ValidHandle(h string) bool {
	return handleRe.MatchString(h) && !strings.HasPrefix(h, ".") && !strings.HasSuffix(h, ".") && !strings.Contains(h, "..")
}

func ReservedHandle(h string) bool { return reservedHandles[h] }
