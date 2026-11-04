// Package pdf renders payment orders that operators use to execute withdrawals.
package pdf

import (
	"bytes"
	_ "embed"
	"fmt"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"

	d "github.com/refresco/refresco-v2/backend/internal/domain"
	"github.com/refresco/refresco-v2/backend/internal/ports"
)

//go:embed logo.png
var logo []byte

type Renderer struct{ Loc *time.Location }

type rgb struct{ r, g, b int }

var (
	ink   = rgb{26, 20, 16}
	brand = rgb{209, 68, 0}
	muted = rgb{110, 104, 97}
	line  = rgb{228, 221, 209}
	paper = rgb{251, 247, 241}
	okC   = rgb{23, 128, 61}
	warnC = rgb{154, 91, 0}
	errC  = rgb{180, 35, 24}
)

var statusName = map[d.WithdrawalStatus]string{d.WithdrawalRequested: "Pedido", d.WithdrawalApproved: "Aprovado", d.WithdrawalPaid: "Pago", d.WithdrawalRejected: "Rejeitado"}
var accStatus = map[d.AccountStatus]string{d.AccountActive: "Ativa", d.AccountPending: "Por verificar", d.AccountSuspended: "Suspensa"}
var months = []string{"jan", "fev", "mar", "abr", "mai", "jun", "jul", "ago", "set", "out", "nov", "dez"}

type doc struct {
	*fpdf.Fpdf
	tr  func(string) string
	loc *time.Location
}

func (r Renderer) PaymentOrders(orders []ports.PaymentOrder, by string, at time.Time) ([]byte, error) {
	if len(orders) == 0 {
		return nil, fmt.Errorf("no orders")
	}
	loc := r.Loc
	if loc == nil {
		loc = time.UTC
	}
	f := fpdf.New("P", "mm", "A4", "")
	f.SetMargins(18, 18, 18)
	f.SetAutoPageBreak(true, 22)
	f.SetTitle("Ordens de pagamento refres.co", true)
	f.SetAuthor("refres.co", true)
	f.SetCreationDate(at)
	p := &doc{Fpdf: f, tr: f.UnicodeTranslatorFromDescriptor(""), loc: loc}
	f.RegisterImageOptionsReader("logo", fpdf.ImageOptions{ImageType: "PNG"}, bytes.NewReader(logo))
	hasLogo := f.Ok()
	if !hasLogo {
		f.ClearError()
	}
	f.SetFooterFunc(func() {
		f.SetY(-14)
		p.font("", 7.5, muted)
		f.CellFormat(0, 4, p.tr(fmt.Sprintf("Documento confidencial com dados bancários. Gerado por %s a %s.", by, p.dt(at))), "", 0, "L", false, 0, "")
		f.CellFormat(0, 4, fmt.Sprintf("%d/{nb}", f.PageNo()), "", 0, "R", false, 0, "")
	})
	f.AliasNbPages("{nb}")

	if len(orders) > 1 {
		p.summary(orders, at, hasLogo)
	}
	for _, o := range orders {
		p.order(o, hasLogo)
	}
	if err := f.Error(); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := f.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (p *doc) font(style string, size float64, c rgb) {
	p.SetFont("Helvetica", style, size)
	p.SetTextColor(c.r, c.g, c.b)
}

func (p *doc) dt(t time.Time) string {
	t = t.In(p.loc)
	return fmt.Sprintf("%02d %s %d, %02d:%02d", t.Day(), months[t.Month()-1], t.Year(), t.Hour(), t.Minute())
}

func money(m d.Money) string {
	v := int64(m)
	neg := v < 0
	if neg {
		v = -v
	}
	whole, cents := fmt.Sprint(v/100), v%100
	var b strings.Builder
	for i, c := range whole {
		if i > 0 && (len(whole)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(c)
	}
	s := fmt.Sprintf("%s,%02d MT", b.String(), cents)
	if neg {
		s = "-" + s
	}
	return s
}

func group4(n string) string {
	var b strings.Builder
	for i, r := range n {
		if i > 0 && i%4 == 0 {
			b.WriteByte(' ')
		}
		b.WriteRune(r)
	}
	return b.String()
}

func dest(o ports.PaymentOrder) (kind, number string) {
	switch o.DestKind {
	case d.PayoutNIB:
		return "Transferência bancária · " + o.DestBank, group4(o.DestNumber)
	case d.PayoutEmola:
		return "e-Mola (Movitel)", d.FormatMZPhone(o.DestNumber)
	}
	return "M-Pesa (Vodacom)", d.FormatMZPhone(o.DestNumber)
}

func (p *doc) header(title, sub string, hasLogo bool) {
	p.AddPage()
	if hasLogo {
		p.ImageOptions("logo", 18, 15, 0, 11, false, fpdf.ImageOptions{ImageType: "PNG"}, 0, "")
	} else {
		p.font("B", 16, brand)
		p.Text(18, 23, "refres.co")
	}
	p.font("", 8, muted)
	p.SetXY(110, 15)
	p.MultiCell(82, 4, p.tr("Refres, Lda · Maputo\nOperações de levantamento"), "", "R", false)
	p.SetY(34)
	p.font("B", 20, ink)
	p.CellFormat(0, 9, p.tr(title), "", 1, "L", false, 0, "")
	p.font("", 9.5, muted)
	p.CellFormat(0, 5, p.tr(sub), "", 1, "L", false, 0, "")
	p.Ln(4)
}

func (p *doc) section(t string) {
	p.Ln(3)
	p.font("B", 8, muted)
	p.CellFormat(0, 5, p.tr(strings.ToUpper(t)), "", 1, "L", false, 0, "")
	p.SetDrawColor(line.r, line.g, line.b)
	p.Line(18, p.GetY(), 192, p.GetY())
	p.Ln(1.5)
}

func (p *doc) kv(k, v string, mono bool) {
	p.font("", 9.5, muted)
	p.CellFormat(52, 6.5, p.tr(k), "", 0, "L", false, 0, "")
	if mono {
		p.font("B", 10.5, ink)
		p.SetFont("Courier", "B", 10.5)
	} else {
		p.font("", 10, ink)
	}
	p.MultiCell(0, 6.5, p.tr(v), "", "L", false)
}

func statusColor(s d.WithdrawalStatus) rgb {
	switch s {
	case d.WithdrawalPaid:
		return okC
	case d.WithdrawalRejected:
		return errC
	}
	return warnC
}

func (p *doc) order(o ports.PaymentOrder, hasLogo bool) {
	p.header("Ordem de pagamento "+o.Code, "Pedido a "+p.dt(o.RequestedAt)+" · prazo de pagamento até "+p.dt(o.DueAt), hasLogo)

	y := p.GetY()
	p.SetFillColor(paper.r, paper.g, paper.b)
	p.SetDrawColor(line.r, line.g, line.b)
	p.RoundedRect(18, y, 174, 26, 3, "1234", "FD")
	p.SetXY(24, y+4)
	p.font("", 9, muted)
	p.CellFormat(100, 5, p.tr("Valor a transferir"), "", 0, "L", false, 0, "")
	sc := statusColor(o.Status)
	p.font("B", 9, sc)
	p.CellFormat(62, 5, p.tr("Estado: "+statusName[o.Status]), "", 1, "R", false, 0, "")
	p.SetX(24)
	p.font("B", 24, ink)
	p.CellFormat(0, 13, p.tr(money(o.Amount)), "", 1, "L", false, 0, "")
	p.SetY(y + 30)

	kind, num := dest(o)
	p.section("Destino")
	p.kv("Meio", kind, false)
	if o.DestKind == d.PayoutNIB {
		p.kv("NIB", num, true)
		holder := o.Beneficiary
		if holder == "" {
			holder = "Não indicado. Confirmar com o criador antes de pagar."
		}
		p.kv("Titular da conta", holder, false)
	} else {
		p.kv("Número", num, true)
		p.kv("Nome", o.MemberName, false)
	}
	p.kv("Descritivo", "REFRES "+o.Code, true)

	p.section("Beneficiário")
	p.kv("Membro", strings.TrimSpace(o.MemberName+" · "+o.MemberRole), false)
	p.kv("Contacto", d.FormatMZPhone(o.MemberPhone), false)
	p.kv("Conta refres.co", o.AccountName+" (refres.co/"+o.AccountHandle+")", false)
	p.kv("Estado da conta", accStatus[o.AccountStatus], false)

	p.section("Histórico")
	p.kv("Pedido", p.dt(o.RequestedAt), false)
	if o.ApprovedAt != nil {
		label := "Aprovado"
		if o.Status == d.WithdrawalRejected {
			label = "Rejeitado"
		}
		p.kv(label, p.dt(*o.ApprovedAt), false)
	}
	if o.Reason != "" {
		p.kv("Motivo", o.Reason, false)
	}
	if o.PaidAt != nil {
		p.kv("Pago", p.dt(*o.PaidAt)+" · ref. "+o.TransferRef, false)
	}

	if o.Status == d.WithdrawalApproved || o.Status == d.WithdrawalRequested {
		p.section("Execução (a preencher pelo operador)")
		p.Ln(2)
		for _, l := range []string{"Referência da transferência", "Executado por", "Data e hora", "Assinatura"} {
			p.font("", 9.5, muted)
			p.CellFormat(52, 9, p.tr(l), "", 0, "L", false, 0, "")
			p.SetDrawColor(line.r, line.g, line.b)
			p.Line(70, p.GetY()+7.5, 192, p.GetY()+7.5)
			p.Ln(10)
		}
		p.Ln(2)
		p.font("", 8.5, muted)
		p.MultiCell(0, 4.5, p.tr("Depois de transferir, marca o pedido como pago no painel de admin com a referência acima. O criador é notificado automaticamente."), "", "L", false)
	}
}

func (p *doc) summary(orders []ports.PaymentOrder, at time.Time, hasLogo bool) {
	var total d.Money
	for _, o := range orders {
		total += o.Amount
	}
	p.header("Lote de levantamentos", fmt.Sprintf("%d ordens · total %s · gerado a %s", len(orders), money(total), p.dt(at)), hasLogo)
	cols := []float64{22, 48, 70, 34}
	heads := []string{"Pedido", "Beneficiário", "Destino", "Valor"}
	p.SetFillColor(paper.r, paper.g, paper.b)
	p.font("B", 8, muted)
	for i, h := range heads {
		align := "L"
		if i == 3 {
			align = "R"
		}
		p.CellFormat(cols[i], 7, p.tr(strings.ToUpper(h)), "B", 0, align, true, 0, "")
	}
	p.Ln(-1)
	for _, o := range orders {
		kind, num := dest(o)
		name := o.Beneficiary
		if name == "" {
			name = o.MemberName
		}
		p.font("B", 9, ink)
		p.SetFont("Courier", "B", 9)
		p.CellFormat(cols[0], 7, o.Code, "B", 0, "L", false, 0, "")
		p.font("", 9, ink)
		p.CellFormat(cols[1], 7, p.tr(trunc(name+" · "+o.AccountName, 30)), "B", 0, "L", false, 0, "")
		p.CellFormat(cols[2], 7, p.tr(trunc(strings.TrimPrefix(kind, "Transferência bancária · ")+" "+num, 44)), "B", 0, "L", false, 0, "")
		p.font("B", 9, ink)
		p.CellFormat(cols[3], 7, p.tr(money(o.Amount)), "B", 1, "R", false, 0, "")
	}
	p.font("B", 10, ink)
	p.CellFormat(cols[0]+cols[1]+cols[2], 9, "Total", "", 0, "R", false, 0, "")
	p.CellFormat(cols[3], 9, p.tr(money(total)), "", 1, "R", false, 0, "")
}

func trunc(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}
