package mock

import (
	"net/http"
	"strconv"
	"testing"
	"time"
)

func TestWebhookRequiresFreshSignedTimestamp(t *testing.T) {
	g := New([]byte("secret"), 0)
	body := []byte(`{"provider_ref":"mock_x","status":"confirmed"}`)
	hdr := func(ts time.Time, sig string) http.Header {
		h := http.Header{}
		s := strconv.FormatInt(ts.Unix(), 10)
		h.Set("X-Mock-Timestamp", s)
		if sig == "" {
			sig = g.Sign(s, body)
		}
		h.Set("X-Mock-Signature", sig)
		return h
	}
	if _, _, err := g.ParseWebhook(hdr(time.Now(), ""), body); err != nil {
		t.Fatalf("fresh webhook rejected: %v", err)
	}
	if _, _, err := g.ParseWebhook(hdr(time.Now().Add(-10*time.Minute), ""), body); err == nil {
		t.Fatal("replayed webhook accepted")
	}
	h := hdr(time.Now(), "")
	h.Set("X-Mock-Timestamp", strconv.FormatInt(time.Now().Unix()+1, 10))
	if _, _, err := g.ParseWebhook(h, body); err == nil {
		t.Fatal("signature not bound to timestamp")
	}
}
