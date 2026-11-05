// Package mtoca is the gateway for mobile wallets through RIHA's own mtoca service,
// which passes payments on to Millennium bim's broker.
//
// Its create call only answers once bim does, so Initiate sends it on a goroutine.
// Mtoca keeps each outcome, so the status can be asked by operation number alone,
// including after a restart here. A payment only counts as failed when bim says so:
// anything else leaves the outcome unknown, because the customer may have paid.
package mtoca

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	d "github.com/refresco/refresco-v2/backend/internal/domain"
	"github.com/refresco/refresco-v2/backend/internal/ports"
)

const (
	// Wait is how long the customer is shown. The prompt alone can take a minute to arrive.
	Wait          = 3 * time.Minute
	createTimeout = 4 * time.Minute
	statusTimeout = 15 * time.Second
	// askEvery stops a busy status page from reaching bim on every poll.
	askEvery    = 3 * time.Second
	forgetAfter = 2 * time.Hour
)

var channels = map[d.Channel]string{d.ChannelMpesa: "MPESA", d.ChannelEmola: "EMOLA", d.ChannelBim: "BIM"}

type Config struct {
	BaseURL     string
	CallbackURL string
	HTTP        *http.Client
	Log         *slog.Logger
	Now         func() time.Time
}

type payment struct {
	started  time.Time
	creating bool
	id       string
	asked    time.Time
	last     ports.PaymentState
	final    bool
}

type Gateway struct {
	cfg      Config
	http     *http.Client
	mu       sync.Mutex
	payments map[string]*payment
}

func New(cfg Config) (*Gateway, error) {
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	if cfg.BaseURL == "" {
		return nil, errors.New("mtoca: the base url is required")
	}
	if cfg.Log == nil {
		cfg.Log = slog.Default()
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	h := cfg.HTTP
	if h == nil {
		h = &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	return &Gateway{cfg: cfg, http: h, payments: map[string]*payment{}}, nil
}

func (g *Gateway) Name() string { return "mtoca" }

// newOperation makes the 12 digit number bim knows the payment by.
func newOperation() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(900_000_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%012d", n.Int64()+100_000_000_000), nil
}

type createBody struct {
	OperationNumber string          `json:"operationNumber"`
	Channel         string          `json:"channel"`
	Amount          json.RawMessage `json:"amount"`
	PhoneNumber     string          `json:"phoneNumber"`
	Description     string          `json:"description"`
	CallbackURL     string          `json:"callbackUrl,omitempty"`
}

type statusBody struct {
	OperationNumber string `json:"operationNumber"`
	TransactionID   string `json:"transactionID_NEP,omitempty"`
}

func (g *Gateway) Initiate(_ context.Context, r ports.PaymentRequest) (ports.PaymentInit, error) {
	channel, ok := channels[r.Channel]
	if !ok {
		return ports.PaymentInit{}, fmt.Errorf("mtoca: cannot take %s payments", r.Channel)
	}
	if r.Amount <= 0 || len(r.Phone) != 9 {
		return ports.PaymentInit{}, errors.New("mtoca: invalid amount or phone")
	}
	op, err := newOperation()
	if err != nil {
		return ports.PaymentInit{}, err
	}
	description := strings.TrimSpace(r.Merchant)
	if description == "" {
		description = "Apoio Refresco"
	}
	if runes := []rune(description); len(runes) > 100 {
		description = string(runes[:100])
	}
	body := createBody{OperationNumber: op, Channel: channel, Amount: json.RawMessage(fmt.Sprintf("%d.%02d", int64(r.Amount)/100, int64(r.Amount)%100)),
		PhoneNumber: "258" + r.Phone, Description: description, CallbackURL: g.cfg.CallbackURL}
	now := g.cfg.Now()
	g.mu.Lock()
	for k, p := range g.payments {
		if now.Sub(p.started) > forgetAfter {
			delete(g.payments, k)
		}
	}
	g.payments[op] = &payment{started: now, creating: true}
	g.mu.Unlock()
	go g.create(op, body)
	return ports.PaymentInit{ProviderRef: op, NextAction: "wallet_pin", ExpiresAt: now.Add(Wait)}, nil
}

func (g *Gateway) create(op string, body createBody) {
	ctx, cancel := context.WithTimeout(context.Background(), createTimeout)
	defer cancel()
	status, out, err := g.post(ctx, "/api/payments", body)
	g.mu.Lock()
	defer g.mu.Unlock()
	p := g.payments[op]
	if p == nil {
		p = &payment{started: g.cfg.Now()}
		g.payments[op] = p
	}
	p.creating = false
	switch {
	case err != nil || status >= 500:
		// Mtoca may have taken it, so only the status can tell.
		g.cfg.Log.Warn("mtoca create lost its answer, asking the status", "op", op, "status", status, "err", err)
	case status >= 200 && status < 300:
		p.id = text(out, "vendorTransactionId_NEP")
	case status == http.StatusConflict:
		// Bim said no, and the status says the same.
		g.cfg.Log.Info("mtoca create refused", "op", op, "message", text(out, "message"))
	default:
		g.cfg.Log.Warn("mtoca create refused before bim", "op", op, "status", status, "message", text(out, "message"))
		p.final, p.last = true, ports.PaymentState{Status: d.SupportFailed, FailReason: "O operador não respondeu. Tenta de novo dentro de minutos"}
	}
}

func (g *Gateway) post(ctx context.Context, path string, body any) (int, map[string]any, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return 0, nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.cfg.BaseURL+path, bytes.NewReader(raw))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	res, err := g.http.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer res.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(res.Body, 64<<10))
	var out map[string]any
	_ = json.Unmarshal(data, &out)
	return res.StatusCode, out, nil
}

func text(m map[string]any, key string) string {
	s, _ := m[key].(string)
	return strings.TrimSpace(s)
}

// stateOf reads a status answer. Bim's rejection reasons don't say why, except 01.
func stateOf(m map[string]any) (ports.PaymentState, bool) {
	switch strings.ToUpper(text(m, "paymentStatus")) {
	case "P":
		return ports.PaymentState{Status: d.SupportConfirmed}, true
	case "C":
		return ports.PaymentState{Status: d.SupportFailed, FailReason: "O pagamento foi cancelado"}, true
	case "R":
		switch text(m, "rejectionReason") {
		case "01":
			return ports.PaymentState{Status: d.SupportExpired, FailReason: "O pedido expirou"}, true
		case "99":
			return ports.PaymentState{Status: d.SupportFailed, FailReason: "O operador não respondeu. Tenta de novo dentro de minutos"}, true
		case "02":
			return ports.PaymentState{Status: d.SupportFailed, FailReason: "O pagamento foi recusado. Confirma o saldo e tenta de novo"}, true
		}
		return ports.PaymentState{Status: d.SupportFailed, FailReason: "O pagamento não foi concluído. Confirma o saldo e tenta de novo"}, true
	}
	return ports.PaymentState{Status: d.SupportPending}, false
}

func (g *Gateway) Status(ctx context.Context, op string) (ports.PaymentState, error) {
	return g.status(ctx, op, false)
}

func (g *Gateway) status(ctx context.Context, op string, now bool) (ports.PaymentState, error) {
	g.mu.Lock()
	p, known := g.payments[op]
	if !known {
		// Not sent by this process, so it was sent before a restart.
		p = &payment{}
		g.payments[op] = p
	}
	if p.final {
		defer g.mu.Unlock()
		return p.last, nil
	}
	if !now && g.cfg.Now().Sub(p.asked) < askEvery {
		defer g.mu.Unlock()
		return ports.PaymentState{Status: d.SupportPending}, nil
	}
	p.asked = g.cfg.Now()
	id, creating, started := p.id, p.creating, p.started
	g.mu.Unlock()

	ctx, cancel := context.WithTimeout(ctx, statusTimeout)
	defer cancel()
	status, out, err := g.post(ctx, "/api/payments/status", statusBody{OperationNumber: op, TransactionID: id})
	switch {
	case err != nil:
		g.cfg.Log.Warn("mtoca status failed", "op", op, "err", err)
		return ports.PaymentState{Status: d.SupportPending}, nil
	case status == http.StatusNotFound:
		// Nobody knows it. It can't be paid unless the create call is still on its way.
		if creating || (known && g.cfg.Now().Sub(started) < Wait) {
			return ports.PaymentState{Status: d.SupportPending}, nil
		}
		return ports.PaymentState{Status: d.SupportExpired, FailReason: "O pedido expirou"}, nil
	case status < 200 || status >= 300:
		g.cfg.Log.Warn("mtoca status gave no answer", "op", op, "status", status, "message", text(out, "message"))
		return ports.PaymentState{Status: d.SupportPending}, nil
	}
	st, final := stateOf(out)
	if final {
		g.mu.Lock()
		p.final, p.last = true, st
		g.mu.Unlock()
	}
	return st, nil
}

// InFlight says the customer may still pay this operation, so it must not be replaced
// by a new request or called cancelled. An error reads as yes, because a second prompt
// could charge them twice.
func (g *Gateway) InFlight(op string) bool {
	g.mu.Lock()
	p, known := g.payments[op]
	if known && p.creating {
		g.mu.Unlock()
		return true
	}
	if known && p.final {
		g.mu.Unlock()
		return false
	}
	g.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	status, out, err := g.post(ctx, "/api/payments/status", statusBody{OperationNumber: op})
	if err != nil || status >= 500 {
		return true
	}
	if status == http.StatusNotFound {
		return false
	}
	_, final := stateOf(out)
	return !final
}

func (g *Gateway) SubmitOTP(context.Context, string, string) (ports.PaymentState, error) {
	return ports.PaymentState{}, errors.New("mtoca: there is no one-time code")
}

// ParseWebhook takes bim's callback. Anyone can reach that address, so its own words
// are never believed: this only asks the status straight away, and that answer counts.
// Polling and the reconcile job still run, so a callback that never comes costs only time.
func (g *Gateway) ParseWebhook(_ http.Header, body []byte) (string, ports.PaymentState, error) {
	var in struct {
		OperationNumber string `json:"operationNumber"`
	}
	if err := json.Unmarshal(body, &in); err != nil || len(in.OperationNumber) != 12 {
		return "", ports.PaymentState{}, errors.New("mtoca: callback without an operation number")
	}
	ctx, cancel := context.WithTimeout(context.Background(), statusTimeout)
	defer cancel()
	st, err := g.status(ctx, in.OperationNumber, true)
	if err != nil {
		return "", ports.PaymentState{}, err
	}
	g.cfg.Log.Info("mtoca callback", "op", in.OperationNumber, "status", st.Status)
	return in.OperationNumber, st, nil
}
