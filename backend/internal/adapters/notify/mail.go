package notify

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"html/template"
	"log/slog"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"

	"github.com/refresco/refresco-v2/backend/internal/ports"
)

const (
	mailTimeout = 30 * time.Second
	mailBacklog = 256
)

var mailRetries = []time.Duration{10 * time.Second, time.Minute, 5 * time.Minute}

// Mail delivers messages addressed to an email by SMTP, on a background worker:
// a request never waits on the mail server, so how long the forgot password form
// takes doesn't show whether an account exists. Anything else, a phone number
// for now, goes to Other. The backlog lives in memory; the mail server has its
// own queue for what it accepted.
type Mail struct {
	Addr, Username, Password, From string
	Other                          ports.Notifier
	Log                            *slog.Logger
	queue                          chan ports.Message
}

func NewMail(ctx context.Context, m Mail) *Mail {
	m.queue = make(chan ports.Message, mailBacklog)
	go m.work(ctx)
	return &m
}

func (n *Mail) Send(ctx context.Context, m ports.Message) error {
	if !strings.Contains(m.To, "@") {
		return n.Other.Send(ctx, m)
	}
	select {
	case n.queue <- m:
		return nil
	default:
		return fmt.Errorf("email backlog full")
	}
}

func (n *Mail) work(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case m := <-n.queue:
			n.deliver(ctx, m)
		}
	}
}

func (n *Mail) deliver(ctx context.Context, m ports.Message) {
	for attempt := 0; ; attempt++ {
		err := n.smtp(ctx, m)
		if err == nil {
			n.Log.Info("email sent", "kind", m.Kind)
			return
		}
		n.Log.Warn("email failed", "kind", m.Kind, "attempt", attempt+1, "err", err)
		if attempt >= len(mailRetries) {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(mailRetries[attempt]):
		}
	}
}

func (n *Mail) smtp(ctx context.Context, m ports.Message) error {
	from, err := mail.ParseAddress(n.From)
	if err != nil {
		return err
	}
	to, err := mail.ParseAddress(m.To)
	if err != nil {
		return err
	}
	msg, err := buildMail(n.From, to, m)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, mailTimeout)
	defer cancel()
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", n.Addr)
	if err != nil {
		return err
	}
	defer conn.Close()
	deadline, _ := ctx.Deadline()
	if err := conn.SetDeadline(deadline); err != nil {
		return err
	}
	host, _, err := net.SplitHostPort(n.Addr)
	if err != nil {
		return err
	}
	c, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer c.Close()
	if err := c.Hello(from.Address[strings.LastIndex(from.Address, "@")+1:]); err != nil {
		return err
	}
	if ok, _ := c.Extension("STARTTLS"); ok {
		if err := c.StartTLS(&tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}); err != nil {
			return err
		}
	}
	if n.Username != "" {
		if err := c.Auth(smtp.PlainAuth("", n.Username, n.Password, host)); err != nil {
			return err
		}
	}
	if err := c.Mail(from.Address); err != nil {
		return err
	}
	if err := c.Rcpt(to.Address); err != nil {
		return err
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return c.Quit()
}

type mailCopy struct{ Subject, Title, Action string }

var mailCopies = map[string]mailCopy{
	"password_reset_code":  {"O teu código Refresco", "Recupera a tua palavra-passe", ""},
	"payout_account_added": {"Nova conta de levantamento no Refresco", "Foi adicionada uma conta de levantamento", ""},
	"withdrawal_paid":      {"O teu levantamento foi pago", "O teu levantamento foi pago", ""},
	"withdrawal_rejected":  {"O teu levantamento foi rejeitado", "O teu levantamento foi rejeitado", ""},
	"split_approval":       {"Nova divisão para aprovar no Refresco", "Tens uma divisão para aprovar", "Ver a divisão"},
}

func copyFor(kind string) mailCopy {
	if c, ok := mailCopies[kind]; ok {
		return c
	}
	return mailCopy{"Notificação do Refresco", "Notificação do Refresco", ""}
}

const (
	mailFooter = "Refresco, um produto RIHA"
	mailAuto   = "Este é um email automático. Para ajuda, escreve para support@rihagateway.com."
)

func buildMail(from string, to *mail.Address, m ports.Message) ([]byte, error) {
	c := copyFor(m.Kind)
	var html bytes.Buffer
	if err := mailHTML.Execute(&html, struct {
		mailCopy
		ports.Message
		Footer, Auto string
	}{c, m, mailFooter, mailAuto}); err != nil {
		return nil, err
	}
	var text strings.Builder
	text.WriteString(c.Title + "\n\n" + m.Body + "\n\n")
	if m.Link != "" {
		text.WriteString(c.Action + ":\n" + m.Link + "\n\n")
	}
	text.WriteString("--\n" + mailFooter + "\n" + mailAuto + "\n")

	var b bytes.Buffer
	h := func(k, v string) { fmt.Fprintf(&b, "%s: %s\r\n", k, v) }
	h("From", from)
	h("To", to.String())
	h("Subject", mime.QEncoding.Encode("utf-8", c.Subject))
	h("Date", time.Now().Format(time.RFC1123Z))
	h("MIME-Version", "1.0")
	h("Auto-Submitted", "auto-generated")
	boundary := fmt.Sprintf("refresco-%d", time.Now().UnixNano())
	h("Content-Type", `multipart/alternative; boundary="`+boundary+`"`)
	part := func(contentType, body string) {
		fmt.Fprintf(&b, "\r\n--%s\r\nContent-Type: %s; charset=utf-8\r\nContent-Transfer-Encoding: base64\r\n\r\n", boundary, contentType)
		enc := base64.StdEncoding.EncodeToString([]byte(body))
		for len(enc) > 76 {
			b.WriteString(enc[:76] + "\r\n")
			enc = enc[76:]
		}
		b.WriteString(enc + "\r\n")
	}
	part("text/plain", text.String())
	part("text/html", html.String())
	b.WriteString("\r\n--" + boundary + "--\r\n")
	return b.Bytes(), nil
}

// Email clients strip style blocks and ignore most layout CSS, hence the tables
// and inline styles.
var mailHTML = template.Must(template.New("mail").Parse(`<!doctype html>
<html lang="pt"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><meta name="color-scheme" content="light"><title>{{.Subject}}</title></head>
<body style="margin:0;padding:0;background:#F6F1EA;">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background:#F6F1EA;"><tr><td align="center" style="padding:32px 16px;">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="max-width:560px;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;color:#1A1410;">
<tr><td style="padding:0 4px 18px;font-size:21px;font-weight:800;letter-spacing:-0.4px;color:#1A1410;">Refres<span style="color:#D14400;">co</span></td></tr>
<tr><td style="background:#FFFFFF;border:1px solid #E4DDD1;border-radius:16px;padding:32px 28px;">
<h1 style="margin:0 0 16px;font-size:23px;line-height:1.25;font-weight:700;letter-spacing:-0.3px;color:#1A1410;">{{.Title}}</h1>
{{if .Code}}<p style="margin:0 0 14px;font-size:15px;line-height:1.55;color:#57524B;">Usa este código para criar uma nova palavra-passe. Expira em 10 minutos.</p>
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="margin:6px 0 16px;"><tr><td align="center" style="background:#FBF7F1;border:1px solid #EFE7DC;border-radius:12px;padding:18px;font-family:ui-monospace,Menlo,Consolas,monospace;font-size:30px;font-weight:700;letter-spacing:8px;color:#1A1410;">{{.Code}}</td></tr></table>
<p style="margin:0;font-size:13px;line-height:1.5;color:#6E6861;">Se não foste tu, ignora este email: a tua palavra-passe não muda.</p>
{{else}}<p style="margin:0 0 14px;font-size:15px;line-height:1.55;color:#57524B;">{{.Body}}</p>{{end}}
{{if .Link}}<table role="presentation" cellpadding="0" cellspacing="0" style="margin:10px 0 6px;"><tr><td style="background:#D14400;border-radius:11px;"><a href="{{.Link}}" style="display:inline-block;padding:13px 24px;font-size:15px;font-weight:600;color:#FFFFFF;text-decoration:none;">{{.Action}}</a></td></tr></table>{{end}}
</td></tr>
<tr><td style="padding:18px 4px 0;font-size:12.5px;line-height:1.6;color:#6E6861;">{{.Footer}}<br>{{.Auto}}</td></tr>
</table></td></tr></table></body></html>
`))
