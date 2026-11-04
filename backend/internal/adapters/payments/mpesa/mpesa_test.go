package mpesa

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	d "github.com/refresco/refresco-v2/backend/internal/domain"
	"github.com/refresco/refresco-v2/backend/internal/ports"
)

type fake struct {
	mu      sync.Mutex
	c2b     func(w http.ResponseWriter, body c2bBody)
	query   func(w http.ResponseWriter, r *http.Request)
	auth    []string
	origin  string
	queries int
}

func answer(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"output_ResponseCode": code, "output_ResponseDesc": "x", "output_TransactionID": "TX1"})
}

func setup(t *testing.T, f *fake) (*Gateway, *rsa.PrivateKey) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	der, _ := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	mux := http.NewServeMux()
	mux.HandleFunc("/ipg/v1x/c2bPayment/singleStage/", func(w http.ResponseWriter, r *http.Request) {
		var b c2bBody
		json.NewDecoder(r.Body).Decode(&b)
		f.mu.Lock()
		f.auth = append(f.auth, r.Header.Get("Authorization"))
		f.origin = r.Header.Get("Origin")
		c := f.c2b
		f.mu.Unlock()
		c(w, b)
	})
	mux.HandleFunc("/ipg/v1x/queryTransactionStatus/", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.queries++
		c := f.query
		f.mu.Unlock()
		if c == nil {
			w.WriteHeader(404)
			return
		}
		c(w, r)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	addr := srv.URL[strings.LastIndex(srv.URL, ":")+1:]
	g, err := New(Config{BaseURL: srv.URL[:strings.LastIndex(srv.URL, ":")], C2BPort: addr, QueryPort: addr, APIKey: "chave-api", PublicKey: base64.StdEncoding.EncodeToString(der),
		ServiceProviderCode: "171717", Log: slog.New(slog.NewTextHandler(io.Discard, nil))})
	if err != nil {
		t.Fatal(err)
	}
	return g, priv
}

func start(t *testing.T, g *Gateway) string {
	t.Helper()
	init, err := g.Initiate(context.Background(), ports.PaymentRequest{Amount: 5000, Channel: d.ChannelMpesa, Phone: "841234567"})
	if err != nil {
		t.Fatal(err)
	}
	return init.ProviderRef
}

func settled(t *testing.T, g *Gateway, ref string) ports.PaymentState {
	t.Helper()
	for i := 0; i < 200; i++ {
		g.mu.Lock()
		s := g.payments[ref].state
		g.mu.Unlock()
		if s != running {
			st, err := g.Status(context.Background(), ref)
			if err != nil {
				t.Fatal(err)
			}
			return st
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("the request never finished")
	return ports.PaymentState{}
}

func TestRequestMatchesTheV1Integration(t *testing.T) {
	f := &fake{}
	var got c2bBody
	done := make(chan struct{})
	f.c2b = func(w http.ResponseWriter, b c2bBody) { got = b; answer(w, 201, "INS-0"); close(done) }
	g, priv := setup(t, f)
	ref := start(t, g)
	<-done
	if got.CustomerMSISDN != "258841234567" || got.Amount != "50.00" || got.ServiceProviderCode != "171717" || got.ThirdPartyReference != ref || got.TransactionReference != ref {
		t.Fatalf("body: %+v", got)
	}
	if !regexp.MustCompile(`^REFRES[A-Z0-9]{14}$`).MatchString(ref) {
		t.Fatalf("reference %q", ref)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.origin != "*" {
		t.Fatalf("origin %q", f.origin)
	}
	cipher, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(f.auth[0], "Bearer "))
	if err != nil {
		t.Fatal(err)
	}
	plain, err := rsa.DecryptPKCS1v15(rand.Reader, priv, cipher)
	if err != nil || string(plain) != "chave-api" {
		t.Fatalf("token: %v %q", err, plain)
	}
}

func TestOutcomes(t *testing.T) {
	for _, c := range []struct {
		name, code string
		http       int
		want       d.SupportStatus
		reason     string
	}{
		{"paid", "INS-0", 201, d.SupportConfirmed, ""},
		{"low balance", "INS-2006", 422, d.SupportFailed, "saldo"},
		{"wrong pin", "INS-2001", 400, d.SupportFailed, "PIN"},
		{"cancelled", "INS-5", 401, d.SupportFailed, "Cancelaste"},
		{"bad key is not the customer's business", "INS-2", 401, d.SupportFailed, "não está disponível"},
		{"unknown code", "INS-777", 400, d.SupportFailed, "não está disponível"},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := &fake{c2b: func(w http.ResponseWriter, _ c2bBody) { answer(w, c.http, c.code) }}
			g, _ := setup(t, f)
			st := settled(t, g, start(t, g))
			if st.Status != c.want || !strings.Contains(st.FailReason, c.reason) {
				t.Fatalf("%+v", st)
			}
			if strings.Contains(st.FailReason, "INS-") {
				t.Fatalf("code shown to the customer: %q", st.FailReason)
			}
		})
	}
}

func TestPendingWhileTheCustomerDecides(t *testing.T) {
	release := make(chan struct{})
	f := &fake{c2b: func(w http.ResponseWriter, _ c2bBody) { <-release; answer(w, 201, "INS-0") }}
	g, _ := setup(t, f)
	start2 := time.Now()
	ref := start(t, g)
	if time.Since(start2) > time.Second {
		t.Fatal("Initiate waited for the customer")
	}
	if st, _ := g.Status(context.Background(), ref); st.Status != d.SupportPending {
		t.Fatalf("%+v", st)
	}
	if !g.InFlight(ref) {
		t.Fatal("a running request is in flight")
	}
	close(release)
	if st := settled(t, g, ref); st.Status != d.SupportConfirmed {
		t.Fatalf("%+v", st)
	}
	if g.InFlight(ref) {
		t.Fatal("a finished request is not in flight")
	}
}

// The customer may have paid, so none of these can count as a failure.
func TestUnclearAnswersAreNeverFailures(t *testing.T) {
	for name, c2b := range map[string]func(w http.ResponseWriter, _ c2bBody){
		"internal error code": func(w http.ResponseWriter, _ c2bBody) { answer(w, 500, "INS-1") },
		"not json":            func(w http.ResponseWriter, _ c2bBody) { w.WriteHeader(502); w.Write([]byte("<html>bad gateway")) },
		"success code, wrong http": func(w http.ResponseWriter, _ c2bBody) {
			answer(w, 200, "INS-0")
		},
	} {
		t.Run(name, func(t *testing.T) {
			f := &fake{c2b: c2b, query: func(w http.ResponseWriter, _ *http.Request) {
				json.NewEncoder(w).Encode(map[string]string{"output_ResponseCode": "INS-0", "output_ResponseTransactionStatus": "Completed"})
			}}
			g, _ := setup(t, f)
			if st := settled(t, g, start(t, g)); st.Status != d.SupportConfirmed {
				t.Fatalf("the query said completed: %+v", st)
			}
		})
	}
}

func TestDroppedConnectionAsksTheQueryAPI(t *testing.T) {
	f := &fake{c2b: func(w http.ResponseWriter, _ c2bBody) {
		c, _, _ := w.(http.Hijacker).Hijack()
		c.Close()
	}, query: func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"output_ResponseTransactionStatus": "Completed"})
	}}
	g, _ := setup(t, f)
	if st := settled(t, g, start(t, g)); st.Status != d.SupportConfirmed {
		t.Fatalf("%+v", st)
	}
}

func TestUnknownReferenceStaysPendingThenExpires(t *testing.T) {
	f := &fake{c2b: func(w http.ResponseWriter, _ c2bBody) { answer(w, 500, "INS-1") }}
	g, _ := setup(t, f)
	ref := start(t, g)
	if st := settled(t, g, ref); st.Status != d.SupportPending {
		t.Fatalf("no answer yet, so pending: %+v", st)
	}
	if !g.InFlight(ref) {
		t.Fatal("still may be paid")
	}
	real := g.now
	g.now = func() time.Time { return real().Add(unknownExpiry + time.Minute) }
	if st, _ := g.Status(context.Background(), ref); st.Status != d.SupportExpired {
		t.Fatalf("%+v", st)
	}
	if g.InFlight(ref) {
		t.Fatal("too old to pay")
	}

	// M-Pesa answering late still counts: the app settles an expired support.
	f.mu.Lock()
	f.query = func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"output_ResponseTransactionStatus": "Completed"})
	}
	f.mu.Unlock()
	g.now = func() time.Time { return real().Add(unknownExpiry + 2*time.Minute) }
	if st, _ := g.Status(context.Background(), ref); st.Status != d.SupportConfirmed {
		t.Fatalf("%+v", st)
	}
}

func TestAfterARestartTheOutcomeComesFromTheQueryAPI(t *testing.T) {
	f := &fake{query: func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("input_QueryReference") == "" || r.URL.Query().Get("input_ServiceProviderCode") != "171717" {
			w.WriteHeader(400)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"output_ResponseTransactionStatus": "Cancelled"})
	}}
	g, _ := setup(t, f)
	ref := g.newRef() // sent before the restart: this process knows nothing about it
	st, err := g.Status(context.Background(), ref)
	if err != nil || st.Status != d.SupportFailed {
		t.Fatalf("%+v %v", st, err)
	}
}

func TestRefusesOtherChannelsAndBadInput(t *testing.T) {
	g, _ := setup(t, &fake{})
	if _, err := g.Initiate(context.Background(), ports.PaymentRequest{Amount: 100, Channel: d.ChannelEmola, Phone: "861234567"}); err == nil {
		t.Fatal("e-Mola through M-Pesa")
	}
	if _, err := g.Initiate(context.Background(), ports.PaymentRequest{Amount: 100, Channel: d.ChannelMpesa, Phone: "84"}); err == nil {
		t.Fatal("short phone")
	}
	if _, _, err := g.ParseWebhook(nil, nil); err == nil {
		t.Fatal("webhook accepted")
	}
	if _, err := New(Config{PublicKey: "not-a-key"}); err == nil {
		t.Fatal("bad key accepted")
	}
}

func TestTimeoutFreesTheCustomerToTryAgain(t *testing.T) {
	f := &fake{c2b: func(w http.ResponseWriter, _ c2bBody) { answer(w, 408, "INS-9") }, query: func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(map[string]string{"output_ResponseCode": "INS-997", "output_ResponseTransactionStatus": "N/A"})
	}}
	g, _ := setup(t, f)
	ref := start(t, g)
	if st := settled(t, g, ref); st.Status != d.SupportExpired {
		t.Fatalf("%+v", st)
	}
	if g.InFlight(ref) {
		t.Fatal("the customer should be able to try again")
	}

	// A late confirmation still counts: the app settles an expired support.
	f.mu.Lock()
	f.query = func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"output_ResponseTransactionStatus": "Completed"})
	}
	f.mu.Unlock()
	real := g.now
	g.now = func() time.Time { return real().Add(queryEvery + time.Second) }
	if st, _ := g.Status(context.Background(), ref); st.Status != d.SupportConfirmed {
		t.Fatalf("%+v", st)
	}
}
