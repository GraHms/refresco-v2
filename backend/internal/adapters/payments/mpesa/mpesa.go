// Package mpesa is the Vodacom M-Pesa gateway.
//
// A customer to business request blocks until the customer answers the prompt on
// their phone, while the app expects a payment to start at once and be polled. So
// Initiate sends the request on a goroutine and Status reads its outcome.
//
// A payment only counts as failed when M-Pesa says so. A timeout, a dropped
// connection or a restart leaves the outcome unknown, because the customer may
// have paid: those references are asked about with the query API until M-Pesa
// answers or they are too old to matter.
package mpesa

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	d "github.com/refresco/refresco-v2/backend/internal/domain"
	"github.com/refresco/refresco-v2/backend/internal/ports"
)

const (
	// Wait is how long the customer is shown to answer the prompt.
	Wait = 2 * time.Minute
	// requestTimeout outlasts Wait, and the 30 seconds the app allows after it.
	requestTimeout = Wait + 30*time.Second
	queryTimeout   = 20 * time.Second
	// A reference with no answer is shown as expired after this long.
	unknownExpiry = Wait + time.Minute
	queryEvery    = 20 * time.Second
	forgetAfter   = 2 * time.Hour
)

const (
	refPrefix   = "REFRES"
	refLen      = 20 // the most M-Pesa accepts
	refStampLen = 6
)

const refAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

type Config struct {
	BaseURL             string
	C2BPort             string
	QueryPort           string
	APIKey              string
	PublicKey           string
	ServiceProviderCode string
	Log                 *slog.Logger
}

type state int

const (
	running state = iota
	finished
	unknown
)

type payment struct {
	state     state
	result    ports.PaymentState
	lastQuery time.Time
}

type Gateway struct {
	cfg      Config
	key      *rsa.PublicKey
	c2bURL   string
	queryURL string
	http     *http.Client
	now      func() time.Time

	mu       sync.Mutex
	payments map[string]*payment
}

func New(cfg Config) (*Gateway, error) {
	key, err := parsePublicKey(cfg.PublicKey)
	if err != nil {
		return nil, err
	}
	if cfg.APIKey == "" || cfg.ServiceProviderCode == "" {
		return nil, errors.New("mpesa: the API key and the service provider code are required")
	}
	base := strings.TrimRight(cfg.BaseURL, "/")
	return &Gateway{
		cfg: cfg, key: key,
		c2bURL:   base + ":" + cfg.C2BPort + "/ipg/v1x/c2bPayment/singleStage/",
		queryURL: base + ":" + cfg.QueryPort + "/ipg/v1x/queryTransactionStatus/",
		http:     &http.Client{},
		now:      time.Now,
		payments: map[string]*payment{},
	}, nil
}

func parsePublicKey(b64 string) (*rsa.PublicKey, error) {
	der, err := base64.StdEncoding.DecodeString(strings.TrimSpace(b64))
	if err != nil {
		return nil, fmt.Errorf("mpesa: decode public key: %w", err)
	}
	parsed, err := x509.ParsePKIXPublicKey(der)
	if err != nil {
		return nil, fmt.Errorf("mpesa: parse public key: %w", err)
	}
	key, ok := parsed.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("mpesa: the public key is not RSA")
	}
	return key, nil
}

func (g *Gateway) Name() string { return "mpesa" }

// bearer is the API key encrypted with M-Pesa's public key, new for each request.
func (g *Gateway) bearer() (string, error) {
	out, err := rsa.EncryptPKCS1v15(rand.Reader, g.key, []byte(g.cfg.APIKey))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(out), nil
}

func (g *Gateway) headers(r *http.Request) error {
	token, err := g.bearer()
	if err != nil {
		return err
	}
	r.Header.Set("Authorization", "Bearer "+token)
	r.Header.Set("Origin", "*")
	r.Header.Set("Content-Type", "application/json")
	return nil
}

// newRef makes a reference of letters and digits, the form M-Pesa accepts. After
// the prefix comes the time, so its age is still known after a restart.
func (g *Gateway) newRef() string {
	stamp := strings.ToUpper(strconv.FormatInt(g.now().Unix(), 36))
	var b strings.Builder
	b.WriteString(refPrefix + stamp)
	for b.Len() < refLen {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(refAlphabet))))
		b.WriteByte(refAlphabet[n.Int64()])
	}
	return b.String()
}

func refAge(ref string, now time.Time) (time.Duration, bool) {
	if len(ref) != refLen || !strings.HasPrefix(ref, refPrefix) {
		return 0, false
	}
	sec, err := strconv.ParseInt(strings.ToLower(ref[len(refPrefix):len(refPrefix)+refStampLen]), 36, 64)
	if err != nil {
		return 0, false
	}
	return now.Sub(time.Unix(sec, 0)), true
}

type c2bBody struct {
	TransactionReference string `json:"input_TransactionReference"`
	CustomerMSISDN       string `json:"input_CustomerMSISDN"`
	Amount               string `json:"input_Amount"`
	ThirdPartyReference  string `json:"input_ThirdPartyReference"`
	ServiceProviderCode  string `json:"input_ServiceProviderCode"`
}

type response struct {
	ConversationID    string `json:"output_ConversationID"`
	ResponseCode      string `json:"output_ResponseCode"`
	ResponseDesc      string `json:"output_ResponseDesc"`
	ThirdPartyRef     string `json:"output_ThirdPartyReference"`
	TransactionID     string `json:"output_TransactionID"`
	TransactionStatus string `json:"output_ResponseTransactionStatus"`
}

func (g *Gateway) Initiate(_ context.Context, r ports.PaymentRequest) (ports.PaymentInit, error) {
	if r.Channel != d.ChannelMpesa {
		return ports.PaymentInit{}, fmt.Errorf("mpesa: cannot take %s payments", r.Channel)
	}
	if r.Amount <= 0 || len(r.Phone) != 9 {
		return ports.PaymentInit{}, errors.New("mpesa: invalid amount or phone")
	}
	ref := g.newRef()
	body, err := json.Marshal(c2bBody{
		TransactionReference: ref, ThirdPartyReference: ref, CustomerMSISDN: "258" + r.Phone,
		Amount: fmt.Sprintf("%d.%02d", int64(r.Amount)/100, int64(r.Amount)%100), ServiceProviderCode: g.cfg.ServiceProviderCode,
	})
	if err != nil {
		return ports.PaymentInit{}, err
	}
	g.mu.Lock()
	g.forgetOld()
	g.payments[ref] = &payment{state: running}
	g.mu.Unlock()
	go g.send(ref, body)
	return ports.PaymentInit{ProviderRef: ref, NextAction: "wallet_pin", ExpiresAt: g.now().Add(Wait)}, nil
}

func (g *Gateway) forgetOld() {
	for ref := range g.payments {
		if age, ok := refAge(ref, g.now()); ok && age > forgetAfter {
			delete(g.payments, ref)
		}
	}
}

func (g *Gateway) send(ref string, body []byte) {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	next := g.c2b(ctx, ref, body)
	g.mu.Lock()
	g.payments[ref] = next
	g.mu.Unlock()
}

func (g *Gateway) c2b(ctx context.Context, ref string, body []byte) *payment {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.c2bURL, bytes.NewReader(body))
	if err != nil {
		return g.notSent(ref, err)
	}
	if err := g.headers(req); err != nil {
		return g.notSent(ref, err)
	}
	res, err := g.http.Do(req)
	if err != nil {
		g.cfg.Log.Warn("mpesa request has no answer", "ref", ref, "err", err)
		return &payment{state: unknown}
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 1<<16))
	if err != nil {
		g.cfg.Log.Warn("mpesa answer cut short", "ref", ref, "err", err)
		return &payment{state: unknown}
	}
	var out response
	if err := json.Unmarshal(raw, &out); err != nil || out.ResponseCode == "" {
		g.cfg.Log.Warn("mpesa answer not understood", "ref", ref, "http", res.StatusCode)
		return &payment{state: unknown}
	}
	return g.outcome(ref, res.StatusCode, out)
}

// notSent is for a request that never left, so it can't have charged anyone.
func (g *Gateway) notSent(ref string, err error) *payment {
	g.cfg.Log.Error("mpesa request not sent", "ref", ref, "err", err)
	return &payment{state: finished, result: ports.PaymentState{Status: d.SupportFailed, FailReason: msgUnavailable}}
}

func (g *Gateway) outcome(ref string, status int, out response) *payment {
	if out.ResponseCode == "INS-0" && status == http.StatusCreated {
		g.cfg.Log.Info("mpesa payment confirmed", "ref", ref, "transaction", out.TransactionID)
		return &payment{state: finished, result: ports.PaymentState{Status: d.SupportConfirmed}}
	}
	// These don't say whether the customer paid.
	if out.ResponseCode == "INS-0" || ambiguous[out.ResponseCode] {
		g.cfg.Log.Warn("mpesa outcome unclear", "ref", ref, "code", out.ResponseCode, "http", status)
		return &payment{state: unknown}
	}
	msg, known := customerMessages[out.ResponseCode]
	if !known {
		// The customer can't act on these: a bad key or short code is ours to fix.
		g.cfg.Log.Error("mpesa refused the request", "ref", ref, "code", out.ResponseCode, "desc", out.ResponseDesc, "http", status)
		msg = msgUnavailable
	} else {
		g.cfg.Log.Info("mpesa payment failed", "ref", ref, "code", out.ResponseCode)
	}
	return &payment{state: finished, result: ports.PaymentState{Status: d.SupportFailed, FailReason: msg}}
}

// The page adds its own full stop after these.
const msgUnavailable = "O M-Pesa não está disponível de momento. Tenta mais tarde"

var ambiguous = map[string]bool{"INS-1": true, "INS-9": true, "INS-23": true}

// customerMessages holds the failures a customer can understand or act on.
var customerMessages = map[string]string{
	"INS-4":    "A tua conta M-Pesa está inativa. Contacta a assistência M-Pesa",
	"INS-5":    "Cancelaste o pedido no telemóvel",
	"INS-6":    "O pagamento falhou. Verifica a tua conta e tenta de novo",
	"INS-10":   "Este pagamento já foi feito. Confirma o teu saldo antes de tentar de novo",
	"INS-15":   "Valor inválido. Verifica o montante do refresco",
	"INS-16":   "O M-Pesa está temporariamente indisponível. Tenta mais tarde",
	"INS-995":  "O teu perfil M-Pesa tem um problema. Contacta a assistência M-Pesa",
	"INS-996":  "A tua conta M-Pesa está inativa. Contacta a assistência M-Pesa",
	"INS-2001": "PIN errado. Tenta de novo",
	"INS-2002": "Número M-Pesa inválido",
	"INS-2006": "O teu saldo M-Pesa é insuficiente. Escolhe um valor mais baixo ou recarrega antes de tentar de novo",
	"INS-2051": "Número M-Pesa inválido",
}

func (g *Gateway) Status(ctx context.Context, ref string) (ports.PaymentState, error) {
	g.mu.Lock()
	p, ok := g.payments[ref]
	if !ok {
		// Not sent by this process, so it was running when the app restarted.
		p = &payment{state: unknown}
		g.payments[ref] = p
	}
	switch p.state {
	case running:
		g.mu.Unlock()
		return ports.PaymentState{Status: d.SupportPending}, nil
	case finished:
		defer g.mu.Unlock()
		return p.result, nil
	}
	due := g.now().Sub(p.lastQuery) >= queryEvery
	if due {
		p.lastQuery = g.now()
	}
	g.mu.Unlock()

	if due {
		st, answered, err := g.query(ctx, ref)
		if err != nil {
			g.cfg.Log.Warn("mpesa query failed", "ref", ref, "err", err)
		} else if answered {
			g.mu.Lock()
			g.payments[ref] = &payment{state: finished, result: st}
			g.mu.Unlock()
			return st, nil
		}
	}
	if g.pastAnswering(ref) {
		return ports.PaymentState{Status: d.SupportExpired, FailReason: "O pedido expirou"}, nil
	}
	return ports.PaymentState{Status: d.SupportPending}, nil
}

func (g *Gateway) pastAnswering(ref string) bool {
	age, ok := refAge(ref, g.now())
	return !ok || age > unknownExpiry
}

// InFlight says the customer may still pay this reference, so it must not be
// replaced by a new request or called cancelled.
func (g *Gateway) InFlight(ref string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	p, ok := g.payments[ref]
	if ok && p.state == running {
		return true
	}
	if ok && p.state == finished {
		return false
	}
	return !g.pastAnswering(ref)
}

func (g *Gateway) query(ctx context.Context, ref string) (ports.PaymentState, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()
	q := url.Values{"input_ThirdPartyReference": {ref}, "input_QueryReference": {ref}, "input_ServiceProviderCode": {g.cfg.ServiceProviderCode}}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, g.queryURL+"?"+q.Encode(), nil)
	if err != nil {
		return ports.PaymentState{}, false, err
	}
	if err := g.headers(req); err != nil {
		return ports.PaymentState{}, false, err
	}
	res, err := g.http.Do(req)
	if err != nil {
		return ports.PaymentState{}, false, err
	}
	defer res.Body.Close()
	var out response
	if err := json.NewDecoder(io.LimitReader(res.Body, 1<<16)).Decode(&out); err != nil {
		return ports.PaymentState{}, false, fmt.Errorf("query answer not understood (http %d)", res.StatusCode)
	}
	switch strings.ToLower(out.TransactionStatus) {
	case "completed":
		g.cfg.Log.Info("mpesa payment confirmed by query", "ref", ref)
		return ports.PaymentState{Status: d.SupportConfirmed}, true, nil
	case "cancelled", "canceled":
		return ports.PaymentState{Status: d.SupportFailed, FailReason: customerMessages["INS-5"]}, true, nil
	case "expired":
		return ports.PaymentState{Status: d.SupportExpired, FailReason: "O pedido expirou"}, true, nil
	}
	return ports.PaymentState{}, false, nil
}

func (g *Gateway) SubmitOTP(context.Context, string, string) (ports.PaymentState, error) {
	return ports.PaymentState{}, errors.New("mpesa: there is no one-time code")
}

// ParseWebhook refuses everything: the outcome comes from the request itself and
// the query API, so nothing should be calling this.
func (g *Gateway) ParseWebhook(http.Header, []byte) (string, ports.PaymentState, error) {
	return "", ports.PaymentState{}, errors.New("mpesa: webhooks are not used")
}
