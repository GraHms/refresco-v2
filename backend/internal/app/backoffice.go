package app

import (
	"context"
	"errors"
	"strings"
	"time"

	d "github.com/refresco/refresco-v2/backend/internal/domain"
	"github.com/refresco/refresco-v2/backend/internal/ports"
)

type Creator struct {
	User    *d.User
	Member  *d.Member
	Account *d.Account
}

func (a *App) CreatorContext(ctx context.Context, u *d.User) (*Creator, error) {
	if u.Role != d.RoleCreator {
		return nil, d.ErrForbidden
	}
	m, err := a.Store.Members().ByUser(ctx, u.ID)
	if err != nil {
		return nil, d.ErrForbidden
	}
	acc, err := a.Store.Accounts().ByID(ctx, m.AccountID)
	if err != nil {
		return nil, err
	}
	return &Creator{User: u, Member: m, Account: acc}, nil
}

type RangeKey string

type rangeSpec struct {
	from, to, prevFrom time.Time
	bucket             ports.Bucket
}

func (a *App) rangeOf(key string) (rangeSpec, error) {
	now := a.now().In(a.Location)
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, a.Location)
	switch key {
	case "7d", "":
		from := day.AddDate(0, 0, -6)
		return rangeSpec{from: from, to: day.AddDate(0, 0, 1), prevFrom: from.AddDate(0, 0, -7), bucket: ports.BucketDay}, nil
	case "30d":
		from := day.AddDate(0, 0, -29)
		return rangeSpec{from: from, to: day.AddDate(0, 0, 1), prevFrom: from.AddDate(0, 0, -30), bucket: ports.BucketDay}, nil
	case "12m":
		m := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, a.Location)
		from := m.AddDate(0, -11, 0)
		return rangeSpec{from: from, to: m.AddDate(0, 1, 0), prevFrom: from.AddDate(0, -12, 0), bucket: ports.BucketMonth}, nil
	}
	return rangeSpec{}, d.Invalid("invalid_range", "Período inválido.")
}

type Overview struct {
	Range      string              `json:"range"`
	Gross      d.Money             `json:"gross"`
	Net        d.Money             `json:"net"`
	Fee        d.Money             `json:"fee"`
	GrossPrev  d.Money             `json:"gross_prev"`
	Count      int64               `json:"count"`
	CountPrev  int64               `json:"count_prev"`
	Supporters int64               `json:"supporters"`
	Average    d.Money             `json:"average"`
	TopAmount  d.Money             `json:"top_amount"`
	ByChannel  ports.ChannelTotals `json:"by_channel"`
	Series     []ports.SeriesPoint `json:"series"`
	Bucket     ports.Bucket        `json:"bucket"`
	FeeBPS     d.BPS               `json:"fee_bps"`
	Split      []SplitRow          `json:"split"`
	Recent     []SupportView       `json:"recent"`
	Unread     int64               `json:"unread"`
}

type SplitRow struct {
	MemberID string  `json:"member_id"`
	Name     string  `json:"name"`
	Pct      int     `json:"pct"`
	Wallet   string  `json:"wallet"`
	Amount   d.Money `json:"amount"`
}

func (a *App) Overview(ctx context.Context, c *Creator, key string) (*Overview, error) {
	r, err := a.rangeOf(key)
	if err != nil {
		return nil, err
	}
	cur, err := a.Metrics.Totals(ctx, c.Account.ID, r.from, r.to)
	if err != nil {
		return nil, err
	}
	prev, err := a.Metrics.Totals(ctx, c.Account.ID, r.prevFrom, r.from)
	if err != nil {
		return nil, err
	}
	series, err := a.Metrics.Series(ctx, c.Account.ID, r.from, r.to, r.bucket, a.Location)
	if err != nil {
		return nil, err
	}
	members, err := a.Store.Members().ByAccount(ctx, c.Account.ID)
	if err != nil {
		return nil, err
	}
	recent, _, err := a.Store.Supports().List(ctx, c.Account.ID, ports.SupportFilter{Limit: 5})
	if err != nil {
		return nil, err
	}
	unread, err := a.Store.Supports().CountUnread(ctx, c.Account.ID)
	if err != nil {
		return nil, err
	}
	if key == "" {
		key = "7d"
	}
	o := &Overview{Range: key, Gross: cur.Gross, Fee: cur.Fee, Net: cur.Gross - cur.Fee, GrossPrev: prev.Gross, Count: cur.Count, CountPrev: prev.Count,
		Supporters: cur.Supporter, TopAmount: cur.TopAmount, ByChannel: cur.ByChan, Series: series, Bucket: r.bucket, FeeBPS: c.Account.EffectiveFee(a.now()), Unread: unread}
	if cur.Count > 0 {
		o.Average = d.Money(int64(cur.Gross) / cur.Count)
	}
	pcts := make([]int, len(members))
	admin := 0
	for i, m := range members {
		pcts[i] = m.Pct
		if m.IsAdmin {
			admin = i
		}
	}
	shares := d.Split(o.Net, pcts, admin)
	for i, m := range members {
		if m.Pct > 0 {
			o.Split = append(o.Split, SplitRow{MemberID: m.ID, Name: m.Name, Pct: m.Pct, Wallet: string(m.Wallet), Amount: shares[i]})
		}
	}
	for _, s := range recent {
		o.Recent = append(o.Recent, supportView(&s, nil))
	}
	return o, nil
}

type SupportView struct {
	ID         string          `json:"id"`
	Ref        string          `json:"ref"`
	Name       string          `json:"name"`
	Net        string          `json:"net,omitempty"`
	Message    string          `json:"message"`
	Amount     d.Money         `json:"amount"`
	Preset     bool            `json:"preset"`
	Fee        d.Money         `json:"fee"`
	FeeBPS     d.BPS           `json:"fee_bps"`
	NetAmount  d.Money         `json:"net_amount"`
	Channel    d.Channel       `json:"channel"`
	Contact    string          `json:"contact"`
	Card       string          `json:"card,omitempty"`
	Status     d.SupportStatus `json:"status"`
	FailReason string          `json:"fail_reason,omitempty"`
	Read       bool            `json:"read"`
	CreatedAt  time.Time       `json:"created_at"`
	Splits     []SplitRow      `json:"splits,omitempty"`
	Hidden     bool            `json:"hidden"`
}

func supportView(s *d.Support, splits []d.SupportSplit) SupportView {
	v := SupportView{ID: s.ID, Ref: s.Ref, Name: s.SupporterName, Net: s.SupporterNet, Message: s.Message, Amount: s.Amount, Preset: s.Preset,
		Fee: s.Fee, FeeBPS: s.FeeBPS, NetAmount: s.Net, Channel: s.Channel, Status: s.Status, FailReason: s.FailReason, Read: s.ReadAt != nil, CreatedAt: s.CreatedAt}
	if s.Status != d.SupportConfirmed {
		v.Name, v.Net, v.Message, v.Hidden = "", "", "", true
	}
	if s.Channel.IsWallet() {
		v.Contact = d.FormatMZPhone(s.PayerPhone)
	} else {
		v.Card = cardLabel(s.CardBrand, s.CardLast4)
		v.Contact = v.Card
	}
	for _, sp := range splits {
		v.Splits = append(v.Splits, SplitRow{MemberID: sp.MemberID, Name: sp.Name, Pct: sp.Pct, Amount: sp.Amount})
	}
	return v
}

func cardLabel(brand, last4 string) string {
	b := "Visa"
	if brand == "mastercard" {
		b = "Mastercard"
	}
	return b + " ···· " + last4
}

type SupportList struct {
	Items    []SupportView       `json:"items"`
	Total    int64               `json:"total"`
	All      int64               `json:"all"`
	Unread   int64               `json:"unread"`
	Channels map[d.Channel]int64 `json:"channels"`
}

func (a *App) ListSupports(ctx context.Context, c *Creator, f ports.SupportFilter) (*SupportList, error) {
	if f.Limit <= 0 || f.Limit > 200 {
		f.Limit = 50
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
	items, total, err := a.Store.Supports().List(ctx, c.Account.ID, f)
	if err != nil {
		return nil, err
	}
	counts, err := a.Store.Supports().ChannelCounts(ctx, c.Account.ID)
	if err != nil {
		return nil, err
	}
	unread, err := a.Store.Supports().CountUnread(ctx, c.Account.ID)
	if err != nil {
		return nil, err
	}
	out := &SupportList{Total: total, Unread: unread, Channels: counts, Items: []SupportView{}}
	for _, n := range counts {
		out.All += n
	}
	for i := range items {
		out.Items = append(out.Items, supportView(&items[i], nil))
	}
	return out, nil
}

func (a *App) GetSupport(ctx context.Context, c *Creator, id string) (*SupportView, error) {
	s, err := a.Store.Supports().ByID(ctx, id)
	if err != nil || s.AccountID != c.Account.ID {
		return nil, d.ErrNotFound
	}
	if s.ReadAt == nil {
		now := a.now()
		if err := a.Store.Supports().SetRead(ctx, s.ID, &now); err != nil {
			return nil, err
		}
		s.ReadAt = &now
	}
	splits, err := a.Store.Supports().Splits(ctx, s.ID)
	if err != nil {
		return nil, err
	}
	v := supportView(s, splits)
	return &v, nil
}

func (a *App) MarkUnread(ctx context.Context, c *Creator, id string) error {
	s, err := a.Store.Supports().ByID(ctx, id)
	if err != nil || s.AccountID != c.Account.ID {
		return d.ErrNotFound
	}
	return a.Store.Supports().SetRead(ctx, s.ID, nil)
}

func (a *App) ExportSupports(ctx context.Context, c *Creator, f ports.SupportFilter) ([]d.Support, error) {
	f.Limit, f.Offset = 10000, 0
	items, _, err := a.Store.Supports().List(ctx, c.Account.ID, f)
	return items, err
}

type PageInput struct {
	Name        string            `json:"name"`
	Handle      string            `json:"handle"`
	Category    string            `json:"category"`
	Bio         string            `json:"bio"`
	Thanks      string            `json:"thanks"`
	Amounts     []d.Money         `json:"amounts"`
	DefaultIdx  int               `json:"default_idx"`
	AllowCustom bool              `json:"allow_custom"`
	MinAmount   d.Money           `json:"min_amount"`
	Socials     map[string]string `json:"socials"`
	Channels    d.ChannelSet      `json:"channels"`
}

func (a *App) UpdatePage(ctx context.Context, c *Creator, in PageInput) (*d.Account, error) {
	if !c.Member.IsAdmin {
		return nil, d.ErrForbidden
	}
	var out *d.Account
	err := a.Store.Tx(ctx, func(s ports.Store) error {
		acc, err := s.Accounts().ByIDForUpdate(ctx, c.Account.ID)
		if err != nil {
			return err
		}
		handle := strings.ToLower(strings.TrimSpace(in.Handle))
		if handle != acc.Handle {
			ok, why, err := a.HandleAvailable(ctx, handle, acc.ID)
			if err != nil {
				return err
			}
			if !ok {
				if why == "taken" {
					return d.Conflict("handle_taken", "Este link já está ocupado.")
				}
				return d.Invalid("invalid_handle", "Usa 3 a 24 letras minúsculas, números ou pontos.")
			}
		}
		acc.Name, acc.Handle, acc.Category, acc.Bio, acc.Thanks = in.Name, handle, in.Category, strings.TrimSpace(in.Bio), strings.TrimSpace(in.Thanks)
		acc.Amounts, acc.DefaultIdx, acc.AllowCustom, acc.MinAmount = in.Amounts, in.DefaultIdx, in.AllowCustom, in.MinAmount
		acc.Socials, acc.Channels = in.Socials, in.Channels
		if err := acc.ValidatePage(); err != nil {
			return err
		}
		out = acc
		return s.Accounts().Update(ctx, acc)
	})
	return out, err
}

func (a *App) adminOnly(c *Creator) error {
	if !c.Member.IsAdmin {
		return d.Forbidden("admin_only", "Só o admin do grupo pode fazer isto.")
	}
	return nil
}

func ensureMember(err error) error {
	if errors.Is(err, d.ErrNotFound) {
		return d.ErrNotFound
	}
	return err
}
