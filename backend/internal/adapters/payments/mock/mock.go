// Package mock is a sandbox payment gateway that mimics M-Pesa/e-Mola USSD push and card 3-D Secure.
//
// Wallet numbers ending in 0000 fail with insufficient funds, numbers ending in 9999 never confirm.
// Card OTP 000000 is rejected. Any other wallet payment confirms after AutoConfirm.
package mock

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	d "github.com/refresco/refresco-v2/backend/internal/domain"
	"github.com/refresco/refresco-v2/backend/internal/ports"
)

type payment struct {
	channel   d.Channel
	phone     string
	created   time.Time
	status    d.SupportStatus
	reason    string
	neverDone bool
}

type Gateway struct {
	mu          sync.Mutex
	payments    map[string]*payment
	AutoConfirm time.Duration
	TTL         time.Duration
	secret      []byte
	now         func() time.Time
}

func New(secret []byte, autoConfirm time.Duration) *Gateway {
	return &Gateway{payments: map[string]*payment{}, AutoConfirm: autoConfirm, TTL: 60 * time.Second, secret: secret, now: time.Now}
}

func (g *Gateway) Name() string { return "mock" }

func newRef() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return "mock_" + hex.EncodeToString(b)
}

func (g *Gateway) Initiate(_ context.Context, r ports.PaymentRequest) (ports.PaymentInit, error) {
	if r.Amount <= 0 {
		return ports.PaymentInit{}, errors.New("invalid amount")
	}
	ref := newRef()
	p := &payment{channel: r.Channel, phone: r.Phone, created: g.now(), status: d.SupportPending}
	next := "wallet_pin"
	switch {
	case r.Channel == d.ChannelCard:
		next = "otp"
		if strings.Contains(r.CardToken, "decline") {
			p.status, p.reason = d.SupportFailed, "Cartão recusado"
		}
	case strings.HasSuffix(r.Phone, "0000"):
		p.status, p.reason = d.SupportFailed, "Saldo insuficiente"
	case strings.HasSuffix(r.Phone, "9999"):
		p.neverDone = true
	}
	g.mu.Lock()
	g.gc()
	g.payments[ref] = p
	g.mu.Unlock()
	return ports.PaymentInit{ProviderRef: ref, NextAction: next, ExpiresAt: p.created.Add(g.TTL)}, nil
}

func (g *Gateway) gc() {
	cut := g.now().Add(-time.Hour)
	for k, p := range g.payments {
		if p.created.Before(cut) {
			delete(g.payments, k)
		}
	}
}

func (g *Gateway) Status(_ context.Context, ref string) (ports.PaymentState, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	p, ok := g.payments[ref]
	if !ok {
		return ports.PaymentState{Status: d.SupportPending}, nil
	}
	if p.status == d.SupportPending && p.channel.IsWallet() && !p.neverDone && g.AutoConfirm > 0 && g.now().Sub(p.created) >= g.AutoConfirm {
		p.status = d.SupportConfirmed
	}
	if p.status == d.SupportPending && g.now().Sub(p.created) > g.TTL {
		p.status, p.reason = d.SupportExpired, "O pedido expirou"
	}
	return ports.PaymentState{Status: p.status, FailReason: p.reason}, nil
}

func (g *Gateway) SubmitOTP(_ context.Context, ref, code string) (ports.PaymentState, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	p, ok := g.payments[ref]
	if !ok {
		return ports.PaymentState{}, errors.New("unknown payment")
	}
	if p.status == d.SupportPending {
		if code == "000000" {
			p.status, p.reason = d.SupportFailed, "Código do banco inválido"
		} else {
			p.status = d.SupportConfirmed
		}
	}
	return ports.PaymentState{Status: p.status, FailReason: p.reason}, nil
}

func (g *Gateway) Simulate(_ context.Context, ref string, success bool) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	p, ok := g.payments[ref]
	if !ok {
		return errors.New("unknown payment")
	}
	if p.status != d.SupportPending {
		return nil
	}
	if success {
		p.status = d.SupportConfirmed
	} else {
		p.status, p.reason = d.SupportFailed, "Pagamento recusado"
	}
	return nil
}

type webhook struct {
	ProviderRef string `json:"provider_ref"`
	Status      string `json:"status"`
	Reason      string `json:"reason"`
}

// Sign returns the signature for a webhook sent at unix time ts.
func (g *Gateway) Sign(ts string, body []byte) string {
	h := hmac.New(sha256.New, g.secret)
	h.Write([]byte(ts + "."))
	h.Write(body)
	return hex.EncodeToString(h.Sum(nil))
}

func (g *Gateway) ParseWebhook(h http.Header, body []byte) (string, ports.PaymentState, error) {
	ts := h.Get("X-Mock-Timestamp")
	sec, err := strconv.ParseInt(ts, 10, 64)
	if err != nil || g.now().Sub(time.Unix(sec, 0)).Abs() > 5*time.Minute {
		return "", ports.PaymentState{}, errors.New("stale or missing timestamp")
	}
	sig, err := hex.DecodeString(h.Get("X-Mock-Signature"))
	if err != nil || !hmac.Equal(sig, mustHex(g.Sign(ts, body))) {
		return "", ports.PaymentState{}, errors.New("bad signature")
	}
	var w webhook
	if err := json.Unmarshal(body, &w); err != nil || w.ProviderRef == "" {
		return "", ports.PaymentState{}, errors.New("bad payload")
	}
	st := d.SupportStatus(w.Status)
	if st != d.SupportConfirmed && st != d.SupportFailed && st != d.SupportExpired {
		return "", ports.PaymentState{}, errors.New("bad status")
	}
	return w.ProviderRef, ports.PaymentState{Status: st, FailReason: w.Reason}, nil
}

func mustHex(s string) []byte {
	b, _ := hex.DecodeString(s)
	return b
}
