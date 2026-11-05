package mtoca

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	d "github.com/refresco/refresco-v2/backend/internal/domain"
	"github.com/refresco/refresco-v2/backend/internal/ports"
)

const nep = "8d2e0220af6b4c1d918e2868746e7560"

// fake answers as mtoca does, refusing "+258" and any status field beyond the two it takes.
type fake struct {
	mu       sync.Mutex
	creates  []map[string]any
	statuses []map[string]any
	reply    map[string]any
	status   int
	create   int
	hang     time.Duration
}

func (f *fake) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var b map[string]any
	_ = json.NewDecoder(r.Body).Decode(&b)
	if r.URL.Path == "/api/payments" && f.hang > 0 {
		time.Sleep(f.hang)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if r.URL.Path == "/api/payments" {
		f.creates = append(f.creates, b)
		if strings.HasPrefix(b["phoneNumber"].(string), "+") {
			w.WriteHeader(400)
			return
		}
		if f.create != 0 {
			w.WriteHeader(f.create)
			_ = json.NewEncoder(w).Encode(map[string]any{"message": "nope"})
			return
		}
		w.WriteHeader(201)
		_ = json.NewEncoder(w).Encode(map[string]any{"vendorTransactionId_NEP": nep, "vendorTransactionStatus": "OK"})
		return
	}
	f.statuses = append(f.statuses, b)
	if b["operationNumber"] == nil || len(b) > 2 || (len(b) == 2 && b["transactionID_NEP"] == nil) {
		w.WriteHeader(400)
		return
	}
	if f.status != 0 {
		w.WriteHeader(f.status)
		return
	}
	w.WriteHeader(201)
	_ = json.NewEncoder(w).Encode(f.reply)
}

func (f *fake) set(reply map[string]any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reply = reply
}

func gateway(t *testing.T, f *fake) *Gateway {
	t.Helper()
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	g, err := New(Config{BaseURL: srv.URL + "/", CallbackURL: "https://pagarefresco.com/api/v1/webhooks/payments"})
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func start(t *testing.T, g *Gateway, ch d.Channel, phone string) string {
	t.Helper()
	init, err := g.Initiate(context.Background(), ports.PaymentRequest{Ref: "R1", Amount: 12345, Channel: ch, Phone: phone, Merchant: "Cardo"})
	if err != nil {
		t.Fatal(err)
	}
	if len(init.ProviderRef) != 12 || init.NextAction != "wallet_pin" || init.ExpiresAt.IsZero() {
		t.Fatalf("init %+v", init)
	}
	return init.ProviderRef
}

func eventually(t *testing.T, ok func() bool) {
	t.Helper()
	for i := 0; i < 300; i++ {
		if ok() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("timed out")
}

func fresh(t *testing.T, g *Gateway, op string) ports.PaymentState {
	t.Helper()
	st, err := g.status(context.Background(), op, true)
	if err != nil {
		t.Fatal(err)
	}
	return st
}

func TestCreateSendsWhatMtocaTakes(t *testing.T) {
	f := &fake{reply: map[string]any{"paymentStatus": "N"}}
	g := gateway(t, f)
	start(t, g, d.ChannelEmola, "861234567")
	eventually(t, func() bool { f.mu.Lock(); defer f.mu.Unlock(); return len(f.creates) == 1 })
	b := f.creates[0]
	if b["channel"] != "EMOLA" || b["phoneNumber"] != "258861234567" || b["amount"] != 123.45 || b["description"] != "Cardo" || b["callbackUrl"] != "https://pagarefresco.com/api/v1/webhooks/payments" {
		t.Fatalf("create %v", b)
	}
	if len(b["operationNumber"].(string)) != 12 {
		t.Fatalf("operation %v", b["operationNumber"])
	}
}

func TestRefusesWhatItCannotTake(t *testing.T) {
	g := gateway(t, &fake{})
	for _, r := range []ports.PaymentRequest{{Amount: 100, Channel: d.ChannelCard, Phone: "841234567"}, {Amount: 0, Channel: d.ChannelMpesa, Phone: "841234567"}, {Amount: 100, Channel: d.ChannelMpesa, Phone: "84"}} {
		if _, err := g.Initiate(context.Background(), r); err == nil {
			t.Fatalf("must refuse %+v", r)
		}
	}
}

func TestInitiateDoesNotWaitForBim(t *testing.T) {
	g := gateway(t, &fake{hang: 300 * time.Millisecond, reply: map[string]any{"paymentStatus": "N"}})
	t0 := time.Now()
	start(t, g, d.ChannelMpesa, "841234567")
	if took := time.Since(t0); took > 100*time.Millisecond {
		t.Fatalf("took %s", took)
	}
}

func TestStatusAnswers(t *testing.T) {
	cases := []struct {
		reply  map[string]any
		status d.SupportStatus
		reason string
	}{
		{map[string]any{"paymentStatus": "N"}, d.SupportPending, ""},
		{map[string]any{"paymentStatus": ""}, d.SupportPending, ""},
		{map[string]any{"paymentStatus": "P"}, d.SupportConfirmed, ""},
		{map[string]any{"paymentStatus": "C"}, d.SupportFailed, "cancelado"},
		{map[string]any{"paymentStatus": "R", "rejectionReason": "01"}, d.SupportExpired, "expirou"},
		{map[string]any{"paymentStatus": "R", "rejectionReason": "02"}, d.SupportFailed, "recusado"},
		{map[string]any{"paymentStatus": "R", "rejectionReason": "03"}, d.SupportFailed, "não foi concluído"},
		{map[string]any{"paymentStatus": "R", "rejectionReason": "99"}, d.SupportFailed, "operador"},
		// The create answer means bim took it. It must never read as paid.
		{map[string]any{"vendorTransactionStatus": "OK", "vendorTransactionId_NEP": nep}, d.SupportPending, ""},
	}
	for _, tc := range cases {
		f := &fake{reply: tc.reply}
		g := gateway(t, f)
		st := fresh(t, g, "123456789012")
		if st.Status != tc.status || !strings.Contains(st.FailReason, tc.reason) {
			t.Fatalf("%v: %+v", tc.reply, st)
		}
	}
}

func TestLostAnswersAreNeverAFailure(t *testing.T) {
	for _, code := range []int{500, 502, 503} {
		f := &fake{status: code}
		g := gateway(t, f)
		if st := fresh(t, g, "123456789012"); st.Status != d.SupportPending {
			t.Fatalf("%d: %+v", code, st)
		}
	}
	// A 5xx on create can still have reached bim: the status decides.
	f := &fake{create: 502, reply: map[string]any{"paymentStatus": "P"}}
	g := gateway(t, f)
	op := start(t, g, d.ChannelMpesa, "841234567")
	eventually(t, func() bool { g.mu.Lock(); defer g.mu.Unlock(); return !g.payments[op].creating })
	if st := fresh(t, g, op); st.Status != d.SupportConfirmed {
		t.Fatalf("paid after a 5xx must count: %+v", st)
	}
	if f.statuses[0]["transactionID_NEP"] != nil {
		t.Fatalf("must ask by operation number: %v", f.statuses[0])
	}
}

func TestStatusUsesTheIdAndStopsAtAnEnd(t *testing.T) {
	f := &fake{reply: map[string]any{"paymentStatus": "N"}}
	g := gateway(t, f)
	op := start(t, g, d.ChannelMpesa, "841234567")
	eventually(t, func() bool { g.mu.Lock(); defer g.mu.Unlock(); return g.payments[op].id == nep })
	fresh(t, g, op)
	if f.statuses[0]["transactionID_NEP"] != nep {
		t.Fatalf("status %v", f.statuses[0])
	}
	f.set(map[string]any{"paymentStatus": "P"})
	fresh(t, g, op)
	n := len(f.statuses)
	if st := fresh(t, g, op); st.Status != d.SupportConfirmed || len(f.statuses) != n {
		t.Fatalf("an end needs no more asking: %+v, %d calls", st, len(f.statuses))
	}
}

func TestBusyPagesDontReachBimEachPoll(t *testing.T) {
	f := &fake{reply: map[string]any{"paymentStatus": "N"}}
	g := gateway(t, f)
	for i := 0; i < 5; i++ {
		if _, err := g.Status(context.Background(), "123456789012"); err != nil {
			t.Fatal(err)
		}
	}
	if len(f.statuses) != 1 {
		t.Fatalf("calls %d", len(f.statuses))
	}
}

func TestNoRecord(t *testing.T) {
	f := &fake{status: 404}
	g := gateway(t, f)
	if st := fresh(t, g, "123456789012"); st.Status != d.SupportExpired {
		t.Fatalf("from before a restart, unknown to mtoca: %+v", st)
	}
	op := start(t, g, d.ChannelMpesa, "841234567")
	if st := fresh(t, g, op); st.Status != d.SupportPending {
		t.Fatalf("just sent: %+v", st)
	}
}

func TestRefusedBeforeBim(t *testing.T) {
	f := &fake{create: 400}
	g := gateway(t, f)
	op := start(t, g, d.ChannelMpesa, "841234567")
	eventually(t, func() bool { g.mu.Lock(); defer g.mu.Unlock(); return !g.payments[op].creating })
	if st := fresh(t, g, op); st.Status != d.SupportFailed {
		t.Fatalf("%+v", st)
	}
}

func TestInFlight(t *testing.T) {
	f := &fake{hang: 200 * time.Millisecond, reply: map[string]any{"paymentStatus": "N"}}
	g := gateway(t, f)
	op := start(t, g, d.ChannelMpesa, "841234567")
	if !g.InFlight(op) {
		t.Fatal("creating")
	}
	eventually(t, func() bool { g.mu.Lock(); defer g.mu.Unlock(); return !g.payments[op].creating })
	if !g.InFlight(op) {
		t.Fatal("pending at bim")
	}
	f.set(map[string]any{"paymentStatus": "R", "rejectionReason": "03"})
	if g.InFlight(op) {
		t.Fatal("ended")
	}
	// Not knowing must not allow a second prompt.
	f.mu.Lock()
	f.status = 502
	f.mu.Unlock()
	if !g.InFlight("123456789012") {
		t.Fatal("no answer reads as in flight")
	}
	f.mu.Lock()
	f.status = 404
	f.mu.Unlock()
	if g.InFlight("123456789013") {
		t.Fatal("unknown to mtoca")
	}
}

func TestCallbackOnlyTriggersAStatusCheck(t *testing.T) {
	f := &fake{reply: map[string]any{"paymentStatus": "N"}}
	g := gateway(t, f)
	// A forged callback that claims it was paid.
	ref, st, err := g.ParseWebhook(nil, []byte(`{"operationNumber":"123456789012","paymentStatus":"P","paymentTimestamp":"2026-11-05 10:25:30"}`))
	if err != nil || ref != "123456789012" || st.Status != d.SupportPending {
		t.Fatalf("%s %+v %v", ref, st, err)
	}
	f.set(map[string]any{"paymentStatus": "P"})
	if _, st, err = g.ParseWebhook(nil, []byte(`{"transactionID_NEP":"x","operationNumber":"123456789012","amount":1.0,"comissionAmount":0.01,"paymentType":"M"}`)); err != nil || st.Status != d.SupportConfirmed {
		t.Fatalf("%+v %v", st, err)
	}
	for _, junk := range []string{"", "not json", `{}`, `{"operationNumber":"12"}`} {
		if _, _, err := g.ParseWebhook(nil, []byte(junk)); err == nil {
			t.Fatalf("%q must be refused", junk)
		}
	}
}
