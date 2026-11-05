package notify

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"log/slog"
	"mime"
	"mime/multipart"
	"net"
	netmail "net/mail"
	"strings"
	"testing"
	"time"

	"github.com/refresco/refresco-v2/backend/internal/ports"
)

// smtpSink is the smallest SMTP server that takes one message.
func smtpSink(t *testing.T) (string, <-chan string) {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { l.Close() })
	got := make(chan string, 4)
	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}
			go func() {
				defer c.Close()
				r := bufio.NewReader(c)
				c.Write([]byte("220 sink\r\n"))
				var data strings.Builder
				inData := false
				for {
					line, err := r.ReadString('\n')
					if err != nil {
						return
					}
					if inData {
						if line == ".\r\n" {
							inData = false
							got <- data.String()
							c.Write([]byte("250 ok\r\n"))
							continue
						}
						data.WriteString(line)
						continue
					}
					switch strings.ToUpper(strings.Fields(line + " x")[0]) {
					case "DATA":
						inData = true
						c.Write([]byte("354 go\r\n"))
					case "QUIT":
						c.Write([]byte("221 bye\r\n"))
						return
					default:
						c.Write([]byte("250 ok\r\n"))
					}
				}
			}()
		}
	}()
	return l.Addr().String(), got
}

type recorder struct{ sent []ports.Message }

func (r *recorder) Send(_ context.Context, m ports.Message) error {
	r.sent = append(r.sent, m)
	return nil
}

func decode(t *testing.T, raw string) (map[string]string, *netmail.Message) {
	t.Helper()
	m, err := netmail.ReadMessage(strings.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	_, params, err := mime.ParseMediaType(m.Header.Get("Content-Type"))
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	r := multipart.NewReader(m.Body, params["boundary"])
	for {
		p, err := r.NextRawPart()
		if err == io.EOF {
			return out, m
		}
		if err != nil {
			t.Fatal(err)
		}
		media, _, _ := mime.ParseMediaType(p.Header.Get("Content-Type"))
		b, _ := io.ReadAll(p)
		dec, err := base64.StdEncoding.DecodeString(string(bytes.NewBufferString(strings.NewReplacer("\r", "", "\n", "").Replace(string(b))).Bytes()))
		if err != nil {
			t.Fatal(err)
		}
		out[media] = string(dec)
	}
}

func TestMailSendsAndRoutes(t *testing.T) {
	addr, got := smtpSink(t)
	other := &recorder{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	n := NewMail(ctx, Mail{Addr: addr, From: "Refresco <no-reply@rihagateway.com>", Other: other, Log: slog.New(slog.NewTextHandler(io.Discard, nil))})

	if err := n.Send(ctx, ports.Message{To: "+258 84 123 4567", Kind: "group_invite", Body: "x"}); err != nil {
		t.Fatal(err)
	}
	if len(other.sent) != 1 {
		t.Fatalf("a phone number must not reach SMTP: %+v", other.sent)
	}
	if err := n.Send(ctx, ports.Message{To: "a@b.co.mz", Kind: "password_reset_code", Body: "código 123456", Code: "123456"}); err != nil {
		t.Fatal(err)
	}
	var raw string
	select {
	case raw = <-got:
	case <-time.After(5 * time.Second):
		t.Fatal("no email")
	}
	parts, m := decode(t, raw)
	if m.Header.Get("From") != "Refresco <no-reply@rihagateway.com>" || !strings.Contains(m.Header.Get("To"), "a@b.co.mz") {
		t.Fatalf("headers: %v", m.Header)
	}
	if !strings.Contains(parts["text/plain"], "support@rihagateway.com") || !strings.Contains(parts["text/html"], "support@rihagateway.com") {
		t.Fatalf("no help line: %v", parts)
	}
	if !strings.Contains(parts["text/plain"], "123456") || !strings.Contains(parts["text/html"], ">123456</td>") {
		t.Fatalf("parts: %v", parts)
	}
}

func TestMailEscapesAndLinks(t *testing.T) {
	addr, _ := netmail.ParseAddress("a@b.co.mz")
	raw, err := buildMail("Refresco <no-reply@rihagateway.com>", addr, ports.Message{Kind: "split_approval", Body: "<b>Tu Pra Tu</b> propôs", Link: "https://pagarefresco.com/aprovar#tok"})
	if err != nil {
		t.Fatal(err)
	}
	parts, _ := decode(t, string(raw))
	if strings.Contains(parts["text/html"], "<b>Tu Pra Tu</b>") || !strings.Contains(parts["text/html"], "Ver a divisão</a>") {
		t.Fatalf("html: %s", parts["text/html"])
	}
	if !strings.Contains(parts["text/html"], "aprovar#tok") || !strings.Contains(parts["text/plain"], "https://pagarefresco.com/aprovar#tok") {
		t.Fatalf("link lost: %v", parts)
	}
}

func TestMailDoesNotWaitOnTheServer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Nothing listens here, and the send must return regardless.
	n := NewMail(ctx, Mail{Addr: "127.0.0.1:1", From: "x <a@b.co>", Other: &recorder{}, Log: slog.New(slog.NewTextHandler(io.Discard, nil))})
	start := time.Now()
	if err := n.Send(ctx, ports.Message{To: "a@b.co.mz", Kind: "x"}); err != nil {
		t.Fatal(err)
	}
	if time.Since(start) > time.Second {
		t.Fatal("send blocked")
	}
}
