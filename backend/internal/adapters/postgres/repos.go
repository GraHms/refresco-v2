package postgres

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	d "github.com/refresco/refresco-v2/backend/internal/domain"
	"github.com/refresco/refresco-v2/backend/internal/ports"
)

type userRepo struct{ db *gorm.DB }

func toUser(m *userModel) *d.User {
	return &d.User{ID: m.ID, Email: m.Email, Phone: deref(m.Phone), Name: m.Name, PasswordHash: m.PasswordHash, Role: d.Role(m.Role), TokenVersion: m.TokenVersion, FailedLogins: m.FailedLogins, LockedUntil: m.LockedUntil, CreatedAt: m.CreatedAt}
}

func fromUser(u *d.User) *userModel {
	return &userModel{ID: u.ID, Email: u.Email, Phone: strPtr(u.Phone), Name: u.Name, PasswordHash: u.PasswordHash, Role: string(u.Role), TokenVersion: u.TokenVersion, CreatedAt: u.CreatedAt}
}

func (r userRepo) Create(ctx context.Context, u *d.User) error {
	return mapErr(r.db.WithContext(ctx).Create(fromUser(u)).Error)
}

func (r userRepo) first(ctx context.Context, q string, v any) (*d.User, error) {
	var m userModel
	if err := r.db.WithContext(ctx).Where(q, v).First(&m).Error; err != nil {
		return nil, mapErr(err)
	}
	return toUser(&m), nil
}

func (r userRepo) ByID(ctx context.Context, id string) (*d.User, error) {
	return r.first(ctx, "id = ?", id)
}
func (r userRepo) ByEmail(ctx context.Context, e string) (*d.User, error) {
	return r.first(ctx, "email = ?", e)
}
func (r userRepo) ByPhone(ctx context.Context, p string) (*d.User, error) {
	return r.first(ctx, "phone = ?", p)
}

func (r userRepo) Update(ctx context.Context, u *d.User) error {
	return mapErr(r.db.WithContext(ctx).Model(&userModel{ID: u.ID}).Updates(map[string]any{
		"name": u.Name, "password_hash": u.PasswordHash, "token_version": u.TokenVersion, "updated_at": time.Now().UTC(),
	}).Error)
}

type accountRepo struct{ db *gorm.DB }

func toAccount(m *accountModel) *d.Account {
	a := &d.Account{ID: m.ID, OwnerID: m.OwnerID, Handle: m.Handle, Name: m.Name, Category: m.Category, Bio: m.Bio, Thanks: m.Thanks,
		DefaultIdx: m.DefaultIdx, AllowCustom: m.AllowCustom, MinAmount: d.Money(m.MinAmount), Socials: map[string]string{},
		Channels: d.ChannelSet{Mpesa: m.ChMpesa, Emola: m.ChEmola, Card: m.ChCard}, FeeBPS: d.BPS(m.FeeBPS), FeeNote: m.FeeNote, FeeUntil: m.FeeUntil,
		Status: d.AccountStatus(m.Status), StatusNote: m.StatusNote, InviteCode: m.InviteCode, AutoSplit: m.AutoSplit, CreatedAt: m.CreatedAt}
	for _, v := range m.Amounts {
		a.Amounts = append(a.Amounts, d.Money(v))
	}
	for k, v := range m.Socials {
		if s, ok := v.(string); ok {
			a.Socials[k] = s
		}
	}
	return a
}

func fromAccount(a *d.Account) *accountModel {
	m := &accountModel{ID: a.ID, OwnerID: a.OwnerID, Handle: a.Handle, Name: a.Name, Category: a.Category, Bio: a.Bio, Thanks: a.Thanks,
		DefaultIdx: a.DefaultIdx, AllowCustom: a.AllowCustom, MinAmount: int64(a.MinAmount), Socials: datatypes.JSONMap{},
		ChMpesa: a.Channels.Mpesa, ChEmola: a.Channels.Emola, ChCard: a.Channels.Card, FeeBPS: int(a.FeeBPS), FeeNote: a.FeeNote, FeeUntil: a.FeeUntil,
		Status: string(a.Status), StatusNote: a.StatusNote, InviteCode: a.InviteCode, AutoSplit: a.AutoSplit, CreatedAt: a.CreatedAt}
	for _, v := range a.Amounts {
		m.Amounts = append(m.Amounts, int64(v))
	}
	for k, v := range a.Socials {
		m.Socials[k] = v
	}
	return m
}

func (r accountRepo) Create(ctx context.Context, a *d.Account) error {
	return mapErr(r.db.WithContext(ctx).Create(fromAccount(a)).Error)
}

func (r accountRepo) get(db *gorm.DB, q string, v any) (*d.Account, error) {
	var m accountModel
	if err := db.Where(q, v).First(&m).Error; err != nil {
		return nil, mapErr(err)
	}
	return toAccount(&m), nil
}

func (r accountRepo) ByID(ctx context.Context, id string) (*d.Account, error) {
	return r.get(r.db.WithContext(ctx), "id = ?", id)
}
func (r accountRepo) ByIDForUpdate(ctx context.Context, id string) (*d.Account, error) {
	return r.get(forUpdate(r.db.WithContext(ctx)), "id = ?", id)
}
func (r accountRepo) ByHandle(ctx context.Context, h string) (*d.Account, error) {
	return r.get(r.db.WithContext(ctx), "handle = ?", h)
}

func (r accountRepo) HandleTaken(ctx context.Context, h, except string) (bool, error) {
	var n int64
	q := r.db.WithContext(ctx).Model(&accountModel{}).Where("handle = ?", h)
	if except != "" {
		q = q.Where("id <> ?", except)
	}
	err := q.Count(&n).Error
	return n > 0, err
}

func (r accountRepo) Update(ctx context.Context, a *d.Account) error {
	m := fromAccount(a)
	m.UpdatedAt = time.Now().UTC()
	return mapErr(r.db.WithContext(ctx).Model(&accountModel{ID: a.ID}).Select("*").Omit("id", "owner_id", "created_at").Updates(m).Error)
}

func likeEscape(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

func (r accountRepo) List(ctx context.Context, f ports.AccountFilter) ([]d.Account, error) {
	q := r.db.WithContext(ctx).Model(&accountModel{}).Order("created_at ASC")
	if f.Status != "" {
		q = q.Where("status = ?", string(f.Status))
	}
	if f.Negotiated {
		q = q.Where("fee_bps <> ? AND (fee_until IS NULL OR fee_until > now())", int(d.DefaultFeeBPS))
	}
	if s := strings.TrimSpace(f.Query); s != "" {
		like := "%" + likeEscape(strings.ToLower(s)) + "%"
		q = q.Where("lower(name) LIKE ? OR handle LIKE ? OR owner_id IN (SELECT id FROM users WHERE lower(name) LIKE ?)", like, like, like)
	}
	var ms []accountModel
	if err := q.Limit(1000).Find(&ms).Error; err != nil {
		return nil, err
	}
	out := make([]d.Account, len(ms))
	for i := range ms {
		out[i] = *toAccount(&ms[i])
	}
	return out, nil
}

func (r accountRepo) ByInviteCodes(ctx context.Context, codes []string) (map[string][]string, error) {
	out := map[string][]string{}
	if len(codes) == 0 {
		return out, nil
	}
	var rows []struct{ InviteCode, Name string }
	if err := r.db.WithContext(ctx).Model(&accountModel{}).Select("invite_code, name").Where("invite_code IN ?", codes).Order("created_at").Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, x := range rows {
		out[x.InviteCode] = append(out[x.InviteCode], x.Name)
	}
	return out, nil
}

type memberRepo struct{ db *gorm.DB }

func toMember(m *memberModel) d.Member {
	return d.Member{ID: m.ID, AccountID: m.AccountID, UserID: m.UserID, Name: m.Name, Role: m.Role, Pct: m.Pct, Wallet: d.Channel(m.Wallet),
		WalletNumber: m.WalletNumber, IsAdmin: m.IsAdmin, Status: d.MemberStatus(m.Status), Balance: d.Money(m.Balance), CreatedAt: m.CreatedAt}
}

func fromMember(m *d.Member) *memberModel {
	return &memberModel{ID: m.ID, AccountID: m.AccountID, UserID: m.UserID, Name: m.Name, Role: m.Role, Pct: m.Pct, Wallet: string(m.Wallet),
		WalletNumber: m.WalletNumber, IsAdmin: m.IsAdmin, Status: string(m.Status), Balance: int64(m.Balance), CreatedAt: m.CreatedAt}
}

func (r memberRepo) Create(ctx context.Context, m *d.Member) error {
	return mapErr(r.db.WithContext(ctx).Create(fromMember(m)).Error)
}

func (r memberRepo) ByID(ctx context.Context, id string) (*d.Member, error) {
	var m memberModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error; err != nil {
		return nil, mapErr(err)
	}
	x := toMember(&m)
	return &x, nil
}

func (r memberRepo) list(db *gorm.DB, accountID string) ([]d.Member, error) {
	var ms []memberModel
	if err := db.Where("account_id = ?", accountID).Order("is_admin DESC, created_at ASC, id ASC").Find(&ms).Error; err != nil {
		return nil, err
	}
	out := make([]d.Member, len(ms))
	for i := range ms {
		out[i] = toMember(&ms[i])
	}
	return out, nil
}

func (r memberRepo) ByAccount(ctx context.Context, id string) ([]d.Member, error) {
	return r.list(r.db.WithContext(ctx), id)
}
func (r memberRepo) LockByAccount(ctx context.Context, id string) ([]d.Member, error) {
	return r.list(forUpdate(r.db.WithContext(ctx)), id)
}

func (r memberRepo) ByUser(ctx context.Context, uid string) (*d.Member, error) {
	var m memberModel
	if err := r.db.WithContext(ctx).Where("user_id = ?", uid).First(&m).Error; err != nil {
		return nil, mapErr(err)
	}
	x := toMember(&m)
	return &x, nil
}

func (r memberRepo) Update(ctx context.Context, m *d.Member) error {
	return mapErr(r.db.WithContext(ctx).Model(&memberModel{ID: m.ID}).Updates(map[string]any{
		"name": m.Name, "role": m.Role, "pct": m.Pct, "wallet": string(m.Wallet), "wallet_number": m.WalletNumber,
		"status": string(m.Status), "balance": int64(m.Balance), "user_id": m.UserID, "updated_at": time.Now().UTC(),
	}).Error)
}

func (r memberRepo) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("member_id = ?", id).Delete(&payoutAccountModel{}).Error; err != nil {
			return err
		}
		return tx.Where("id = ?", id).Delete(&memberModel{}).Error
	})
}

type payoutRepo struct{ db *gorm.DB }

func toPayout(m *payoutAccountModel) d.PayoutAccount {
	return d.PayoutAccount{ID: m.ID, MemberID: m.MemberID, Kind: d.PayoutKind(m.Kind), Number: m.Number, BankName: m.BankName, Holder: m.Holder, IsMain: m.IsMain, CreatedAt: m.CreatedAt}
}

func (r payoutRepo) Create(ctx context.Context, p *d.PayoutAccount) error {
	return mapErr(r.db.WithContext(ctx).Create(&payoutAccountModel{ID: p.ID, MemberID: p.MemberID, Kind: string(p.Kind), Number: p.Number, BankName: p.BankName, Holder: p.Holder, IsMain: p.IsMain, CreatedAt: p.CreatedAt}).Error)
}

func (r payoutRepo) ByID(ctx context.Context, id string) (*d.PayoutAccount, error) {
	var m payoutAccountModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error; err != nil {
		return nil, mapErr(err)
	}
	x := toPayout(&m)
	return &x, nil
}

func (r payoutRepo) ByMember(ctx context.Context, mid string) ([]d.PayoutAccount, error) {
	var ms []payoutAccountModel
	if err := r.db.WithContext(ctx).Where("member_id = ?", mid).Order("is_main DESC, created_at").Find(&ms).Error; err != nil {
		return nil, err
	}
	out := make([]d.PayoutAccount, len(ms))
	for i := range ms {
		out[i] = toPayout(&ms[i])
	}
	return out, nil
}

func (r payoutRepo) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&payoutAccountModel{}).Error
}

func (r payoutRepo) ClearMain(ctx context.Context, mid string) error {
	return r.db.WithContext(ctx).Model(&payoutAccountModel{}).Where("member_id = ?", mid).Update("is_main", false).Error
}

type supportRepo struct{ db *gorm.DB }

func toSupport(m *supportModel) d.Support {
	return d.Support{ID: m.ID, Ref: m.Ref, AccountID: m.AccountID, Amount: d.Money(m.Amount), FeeBPS: d.BPS(m.FeeBPS), Fee: d.Money(m.Fee), Net: d.Money(m.Net),
		Preset: m.Preset, Channel: d.Channel(m.Channel), PayerPhone: m.PayerPhone, CardBrand: m.CardBrand, CardLast4: m.CardLast4,
		SupporterName: m.SupporterName, SupporterNet: m.SupporterNet, Message: m.Message, Status: d.SupportStatus(m.Status), FailReason: m.FailReason,
		ProviderRef: deref(m.ProviderRef), NextAction: m.NextAction, Pushes: m.Pushes, ExpiresAt: m.ExpiresAt, IdemKey: deref(m.IdemKey), ReadAt: m.ReadAt,
		ConfirmedAt: m.ConfirmedAt, CreatedAt: m.CreatedAt}
}

func fromSupport(s *d.Support) *supportModel {
	m := &supportModel{ID: s.ID, Ref: s.Ref, AccountID: s.AccountID, Amount: int64(s.Amount), FeeBPS: int(s.FeeBPS), Fee: int64(s.Fee), Net: int64(s.Net),
		Preset: s.Preset, Channel: string(s.Channel), PayerPhone: s.PayerPhone, CardBrand: s.CardBrand, CardLast4: s.CardLast4,
		SupporterName: s.SupporterName, SupporterNet: s.SupporterNet, Message: s.Message, Status: string(s.Status), FailReason: s.FailReason,
		ProviderRef: strPtr(s.ProviderRef), NextAction: s.NextAction, Pushes: s.Pushes, ExpiresAt: s.ExpiresAt, IdemKey: strPtr(s.IdemKey), ReadAt: s.ReadAt,
		ConfirmedAt: s.ConfirmedAt, CreatedAt: s.CreatedAt}
	if s.IdemKey != "" {
		m.IdemAccount = &m.AccountID
	}
	return m
}

func (r supportRepo) Create(ctx context.Context, s *d.Support) error {
	return mapErr(r.db.WithContext(ctx).Create(fromSupport(s)).Error)
}

func (r supportRepo) get(db *gorm.DB, q string, args ...any) (*d.Support, error) {
	var m supportModel
	if err := db.Where(q, args...).First(&m).Error; err != nil {
		return nil, mapErr(err)
	}
	x := toSupport(&m)
	return &x, nil
}

func (r supportRepo) ByID(ctx context.Context, id string) (*d.Support, error) {
	return r.get(r.db.WithContext(ctx), "id = ?", id)
}
func (r supportRepo) ByIDForUpdate(ctx context.Context, id string) (*d.Support, error) {
	return r.get(forUpdate(r.db.WithContext(ctx)), "id = ?", id)
}
func (r supportRepo) ByProviderRef(ctx context.Context, ref string) (*d.Support, error) {
	return r.get(r.db.WithContext(ctx), "provider_ref = ?", ref)
}
func (r supportRepo) ByIdemKey(ctx context.Context, acc, key string) (*d.Support, error) {
	return r.get(r.db.WithContext(ctx), "idem_account = ? AND idem_key = ?", acc, key)
}

func (r supportRepo) Update(ctx context.Context, s *d.Support) error {
	m := fromSupport(s)
	m.UpdatedAt = time.Now().UTC()
	return mapErr(r.db.WithContext(ctx).Model(&supportModel{ID: s.ID}).Select("*").Omit("id", "account_id", "created_at", "idem_key", "idem_account").Updates(m).Error)
}

func (r supportRepo) filtered(ctx context.Context, acc string, f ports.SupportFilter) *gorm.DB {
	q := r.db.WithContext(ctx).Model(&supportModel{}).Where("account_id = ?", acc)
	if f.Channel != "" {
		q = q.Where("channel = ?", string(f.Channel))
	}
	if f.Status != "" {
		if f.Status == d.SupportFailed {
			q = q.Where("status IN ?", []string{string(d.SupportFailed), string(d.SupportExpired)})
		} else {
			q = q.Where("status = ?", string(f.Status))
		}
	}
	if s := strings.TrimSpace(f.Query); s != "" {
		like := "%" + likeEscape(strings.ToLower(s)) + "%"
		digits := strings.Map(func(r rune) rune {
			if r >= '0' && r <= '9' {
				return r
			}
			return -1
		}, s)
		cond := "lower(supporter_name) LIKE ? OR lower(message) LIKE ? OR lower(ref) LIKE ?"
		args := []any{like, like, like}
		if len(digits) >= 3 {
			cond += " OR payer_phone LIKE ?"
			args = append(args, "%"+digits+"%")
		}
		q = q.Where(cond, args...)
	}
	return q
}

func (r supportRepo) List(ctx context.Context, acc string, f ports.SupportFilter) ([]d.Support, int64, error) {
	var total int64
	if err := r.filtered(ctx, acc, f).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var ms []supportModel
	q := r.filtered(ctx, acc, f).Order("created_at DESC").Offset(f.Offset)
	if f.Limit > 0 {
		q = q.Limit(f.Limit)
	}
	if err := q.Find(&ms).Error; err != nil {
		return nil, 0, err
	}
	out := make([]d.Support, len(ms))
	for i := range ms {
		out[i] = toSupport(&ms[i])
	}
	return out, total, nil
}

func (r supportRepo) ChannelCounts(ctx context.Context, acc string) (map[d.Channel]int64, error) {
	var rows []struct {
		Channel string
		N       int64
	}
	if err := r.db.WithContext(ctx).Model(&supportModel{}).Select("channel, count(*) AS n").Where("account_id = ?", acc).Group("channel").Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := map[d.Channel]int64{d.ChannelMpesa: 0, d.ChannelEmola: 0, d.ChannelCard: 0}
	for _, x := range rows {
		out[d.Channel(x.Channel)] = x.N
	}
	return out, nil
}

func (r supportRepo) CountUnread(ctx context.Context, acc string) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&supportModel{}).Where("account_id = ? AND read_at IS NULL AND status = ?", acc, string(d.SupportConfirmed)).Count(&n).Error
	return n, err
}

func (r supportRepo) SetRead(ctx context.Context, id string, at *time.Time) error {
	return r.db.WithContext(ctx).Model(&supportModel{ID: id}).Update("read_at", at).Error
}

func (r supportRepo) AddSplits(ctx context.Context, splits []d.SupportSplit) error {
	if len(splits) == 0 {
		return nil
	}
	ms := make([]supportSplitModel, len(splits))
	for i, s := range splits {
		ms[i] = supportSplitModel{SupportID: s.SupportID, MemberID: s.MemberID, Name: s.Name, Pct: s.Pct, Amount: int64(s.Amount)}
	}
	return mapErr(r.db.WithContext(ctx).Create(&ms).Error)
}

func (r supportRepo) Splits(ctx context.Context, id string) ([]d.SupportSplit, error) {
	var ms []supportSplitModel
	if err := r.db.WithContext(ctx).Where("support_id = ?", id).Order("pct DESC").Find(&ms).Error; err != nil {
		return nil, err
	}
	out := make([]d.SupportSplit, len(ms))
	for i, m := range ms {
		out[i] = d.SupportSplit{SupportID: m.SupportID, MemberID: m.MemberID, Name: m.Name, Pct: m.Pct, Amount: d.Money(m.Amount)}
	}
	return out, nil
}

func (r supportRepo) ExpireStale(ctx context.Context, before time.Time) (int64, error) {
	res := r.db.WithContext(ctx).Model(&supportModel{}).Where("status = ? AND created_at < ?", string(d.SupportPending), before).
		Updates(map[string]any{"status": string(d.SupportExpired), "fail_reason": "O pedido expirou", "updated_at": time.Now().UTC()})
	return res.RowsAffected, res.Error
}

func (r supportRepo) Unsettled(ctx context.Context, since time.Time, limit int) ([]d.Support, error) {
	var rows []supportModel
	err := r.db.WithContext(ctx).
		Where("status IN ? AND provider_ref IS NOT NULL AND provider_ref <> '' AND channel IN ? AND created_at >= ?",
			[]string{string(d.SupportPending), string(d.SupportExpired)}, []string{string(d.ChannelMpesa), string(d.ChannelEmola)}, since).
		Order("created_at").Limit(limit).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]d.Support, 0, len(rows))
	for i := range rows {
		out = append(out, toSupport(&rows[i]))
	}
	return out, nil
}

type withdrawalRepo struct{ db *gorm.DB }

func toWithdrawal(m *withdrawalModel) d.Withdrawal {
	return d.Withdrawal{ID: m.ID, Number: m.Number, AccountID: m.AccountID, MemberID: m.MemberID, Amount: d.Money(m.Amount), DestKind: d.PayoutKind(m.DestKind),
		DestNumber: m.DestNumber, DestBank: m.DestBank, DestHolder: m.DestHolder, Status: d.WithdrawalStatus(m.Status), Reason: m.Reason, Note: m.Note, TransferRef: m.TransferRef,
		DueAt: m.DueAt, DecidedAt: m.DecidedAt, PaidAt: m.PaidAt, DecidedBy: m.DecidedBy, CreatedAt: m.CreatedAt}
}

func (r withdrawalRepo) Create(ctx context.Context, w *d.Withdrawal) error {
	m := &withdrawalModel{ID: w.ID, AccountID: w.AccountID, MemberID: w.MemberID, Amount: int64(w.Amount), DestKind: string(w.DestKind), DestNumber: w.DestNumber,
		DestBank: w.DestBank, DestHolder: w.DestHolder, Status: string(w.Status), DueAt: w.DueAt, CreatedAt: w.CreatedAt}
	if err := r.db.WithContext(ctx).Omit("number").Clauses(clause.Returning{Columns: []clause.Column{{Name: "number"}}}).Create(m).Error; err != nil {
		return mapErr(err)
	}
	w.Number = m.Number
	return nil
}

func (r withdrawalRepo) get(db *gorm.DB, id string) (*d.Withdrawal, error) {
	var m withdrawalModel
	if err := db.Where("id = ?", id).First(&m).Error; err != nil {
		return nil, mapErr(err)
	}
	x := toWithdrawal(&m)
	return &x, nil
}

func (r withdrawalRepo) ByID(ctx context.Context, id string) (*d.Withdrawal, error) {
	return r.get(r.db.WithContext(ctx), id)
}
func (r withdrawalRepo) ByIDForUpdate(ctx context.Context, id string) (*d.Withdrawal, error) {
	return r.get(forUpdate(r.db.WithContext(ctx)), id)
}

func (r withdrawalRepo) Update(ctx context.Context, w *d.Withdrawal) error {
	return mapErr(r.db.WithContext(ctx).Model(&withdrawalModel{ID: w.ID}).Updates(map[string]any{
		"status": string(w.Status), "reason": w.Reason, "note": w.Note, "transfer_ref": w.TransferRef,
		"decided_at": w.DecidedAt, "paid_at": w.PaidAt, "decided_by": w.DecidedBy, "updated_at": time.Now().UTC(),
	}).Error)
}

func (r withdrawalRepo) List(ctx context.Context, f ports.WithdrawalFilter) ([]d.Withdrawal, error) {
	q := r.db.WithContext(ctx).Model(&withdrawalModel{}).Order("created_at DESC")
	if f.AccountID != "" {
		q = q.Where("account_id = ?", f.AccountID)
	}
	if f.MemberID != "" {
		q = q.Where("member_id = ?", f.MemberID)
	}
	if len(f.Statuses) > 0 {
		ss := make([]string, len(f.Statuses))
		for i, s := range f.Statuses {
			ss[i] = string(s)
		}
		q = q.Where("status IN ?", ss)
	}
	var ms []withdrawalModel
	if err := q.Limit(500).Find(&ms).Error; err != nil {
		return nil, err
	}
	out := make([]d.Withdrawal, len(ms))
	for i := range ms {
		out[i] = toWithdrawal(&ms[i])
	}
	return out, nil
}

type inviteRepo struct{ db *gorm.DB }

func toInvite(m *inviteModel) *d.Invite {
	return &d.Invite{Code: m.Code, Note: m.Note, MaxUses: m.MaxUses, Uses: m.Uses, FeeBPS: d.BPS(m.FeeBPS), Status: d.InviteStatus(m.Status), CreatedAt: m.CreatedAt}
}

func (r inviteRepo) Create(ctx context.Context, i *d.Invite) error {
	return mapErr(r.db.WithContext(ctx).Create(&inviteModel{Code: i.Code, Note: i.Note, MaxUses: i.MaxUses, Uses: i.Uses, FeeBPS: int(i.FeeBPS), Status: string(i.Status), CreatedAt: i.CreatedAt}).Error)
}

func (r inviteRepo) get(db *gorm.DB, code string) (*d.Invite, error) {
	var m inviteModel
	if err := db.Where("code = ?", code).First(&m).Error; err != nil {
		return nil, mapErr(err)
	}
	return toInvite(&m), nil
}

func (r inviteRepo) ByCode(ctx context.Context, c string) (*d.Invite, error) {
	return r.get(r.db.WithContext(ctx), c)
}
func (r inviteRepo) ByCodeForUpdate(ctx context.Context, c string) (*d.Invite, error) {
	return r.get(forUpdate(r.db.WithContext(ctx)), c)
}

func (r inviteRepo) Update(ctx context.Context, i *d.Invite) error {
	return mapErr(r.db.WithContext(ctx).Model(&inviteModel{Code: i.Code}).Updates(map[string]any{"uses": i.Uses, "status": string(i.Status), "note": i.Note, "updated_at": time.Now().UTC()}).Error)
}

func (r inviteRepo) List(ctx context.Context) ([]d.Invite, error) {
	var ms []inviteModel
	if err := r.db.WithContext(ctx).Order("created_at DESC").Limit(500).Find(&ms).Error; err != nil {
		return nil, err
	}
	out := make([]d.Invite, len(ms))
	for i := range ms {
		out[i] = *toInvite(&ms[i])
	}
	return out, nil
}

type proposalRepo struct{ db *gorm.DB }

func toProposal(m *proposalModel) *d.SplitProposal {
	p := &d.SplitProposal{ID: m.ID, AccountID: m.AccountID, Status: d.ProposalStatus(m.Status), CreatedBy: m.CreatedBy, CreatedAt: m.CreatedAt}
	_ = json.Unmarshal(m.Splits, &p.Splits)
	return p
}

func (r proposalRepo) Create(ctx context.Context, p *d.SplitProposal, apps []d.SplitApproval) error {
	raw, err := json.Marshal(p.Splits)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&proposalModel{ID: p.ID, AccountID: p.AccountID, Splits: raw, Status: string(p.Status), CreatedBy: p.CreatedBy, CreatedAt: p.CreatedAt}).Error; err != nil {
			return mapErr(err)
		}
		for _, a := range apps {
			if err := tx.Create(&approvalModel{ProposalID: a.ProposalID, MemberID: a.MemberID, TokenHash: a.TokenHash}).Error; err != nil {
				return mapErr(err)
			}
		}
		return nil
	})
}

func (r proposalRepo) PendingByAccount(ctx context.Context, acc string) (*d.SplitProposal, error) {
	var m proposalModel
	if err := r.db.WithContext(ctx).Where("account_id = ? AND status = ?", acc, string(d.ProposalPending)).Order("created_at DESC").First(&m).Error; err != nil {
		return nil, mapErr(err)
	}
	return toProposal(&m), nil
}

func (r proposalRepo) ByID(ctx context.Context, id string) (*d.SplitProposal, error) {
	var m proposalModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error; err != nil {
		return nil, mapErr(err)
	}
	return toProposal(&m), nil
}

func (r proposalRepo) Update(ctx context.Context, p *d.SplitProposal) error {
	return r.db.WithContext(ctx).Model(&proposalModel{ID: p.ID}).Updates(map[string]any{"status": string(p.Status), "updated_at": time.Now().UTC()}).Error
}

func (r proposalRepo) Approvals(ctx context.Context, pid string) ([]d.SplitApproval, error) {
	var ms []approvalModel
	if err := r.db.WithContext(ctx).Where("proposal_id = ?", pid).Find(&ms).Error; err != nil {
		return nil, err
	}
	out := make([]d.SplitApproval, len(ms))
	for i, m := range ms {
		out[i] = d.SplitApproval{ProposalID: m.ProposalID, MemberID: m.MemberID, TokenHash: m.TokenHash, ApprovedAt: m.ApprovedAt}
	}
	return out, nil
}

func (r proposalRepo) ApprovalByToken(ctx context.Context, h string) (*d.SplitApproval, error) {
	var m approvalModel
	if err := r.db.WithContext(ctx).Where("token_hash = ?", h).First(&m).Error; err != nil {
		return nil, mapErr(err)
	}
	return &d.SplitApproval{ProposalID: m.ProposalID, MemberID: m.MemberID, TokenHash: m.TokenHash, ApprovedAt: m.ApprovedAt}, nil
}

func (r proposalRepo) Approve(ctx context.Context, pid, mid string, at time.Time) error {
	return r.db.WithContext(ctx).Model(&approvalModel{}).Where("proposal_id = ? AND member_id = ?", pid, mid).Update("approved_at", at).Error
}

type auditRepo struct{ db *gorm.DB }

func (r auditRepo) Add(ctx context.Context, e *d.AuditEntry) error {
	return r.db.WithContext(ctx).Create(&auditModel{ID: e.ID, AccountID: e.AccountID, Scope: e.Scope, Text: e.Text, CreatedAt: e.CreatedAt}).Error
}

func (r auditRepo) List(ctx context.Context, acc, scope string, limit int) ([]d.AuditEntry, error) {
	var ms []auditModel
	if err := r.db.WithContext(ctx).Where("account_id = ? AND scope = ?", acc, scope).Order("created_at DESC").Limit(limit).Find(&ms).Error; err != nil {
		return nil, err
	}
	out := make([]d.AuditEntry, len(ms))
	for i, m := range ms {
		out[i] = d.AuditEntry{ID: m.ID, AccountID: m.AccountID, Scope: m.Scope, Text: m.Text, CreatedAt: m.CreatedAt}
	}
	return out, nil
}

type resetRepo struct{ db *gorm.DB }

func toReset(m *resetModel) *d.PasswordReset {
	return &d.PasswordReset{ID: m.ID, UserID: m.UserID, CodeHash: m.CodeHash, ExpiresAt: m.ExpiresAt, Attempts: m.Attempts, ResetTokenHash: deref(m.ResetTokenHash),
		VerifiedAt: m.VerifiedAt, UsedAt: m.UsedAt, CreatedAt: m.CreatedAt}
}

func (r resetRepo) Create(ctx context.Context, x *d.PasswordReset) error {
	return r.db.WithContext(ctx).Create(&resetModel{ID: x.ID, UserID: x.UserID, CodeHash: x.CodeHash, ExpiresAt: x.ExpiresAt, CreatedAt: x.CreatedAt}).Error
}

func (r resetRepo) Latest(ctx context.Context, uid string) (*d.PasswordReset, error) {
	var m resetModel
	if err := r.db.WithContext(ctx).Where("user_id = ?", uid).Order("created_at DESC").First(&m).Error; err != nil {
		return nil, mapErr(err)
	}
	return toReset(&m), nil
}

func (r resetRepo) ByTokenHash(ctx context.Context, h string) (*d.PasswordReset, error) {
	var m resetModel
	if err := forUpdate(r.db.WithContext(ctx)).Where("reset_token_hash = ?", h).First(&m).Error; err != nil {
		return nil, mapErr(err)
	}
	return toReset(&m), nil
}

func (r resetRepo) Update(ctx context.Context, x *d.PasswordReset) error {
	return r.db.WithContext(ctx).Model(&resetModel{ID: x.ID}).Updates(map[string]any{
		"attempts": x.Attempts, "reset_token_hash": strPtr(x.ResetTokenHash), "verified_at": x.VerifiedAt, "used_at": x.UsedAt, "expires_at": x.ExpiresAt,
	}).Error
}

type waitlistRepo struct{ db *gorm.DB }

func (r waitlistRepo) Join(ctx context.Context, email string) (int64, error) {
	db := r.db.WithContext(ctx)
	if err := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&waitlistModel{Email: email, CreatedAt: time.Now().UTC()}).Error; err != nil {
		return 0, err
	}
	var m waitlistModel
	if err := db.Where("email = ?", email).First(&m).Error; err != nil {
		return 0, mapErr(err)
	}
	var pos int64
	err := db.Model(&waitlistModel{}).Where("id <= ?", m.ID).Count(&pos).Error
	return pos, err
}

type ledgerRepo struct{ db *gorm.DB }

func (r ledgerRepo) Add(ctx context.Context, e ports.LedgerEntry) error {
	return r.db.WithContext(ctx).Create(&ledgerModel{MemberID: e.MemberID, AccountID: e.AccountID, Kind: e.Kind, Amount: int64(e.Amount), Ref: e.Ref, CreatedAt: e.CreatedAt}).Error
}

func (r userRepo) RecordLoginFailure(ctx context.Context, id string, max int, lockUntil time.Time) error {
	return r.db.WithContext(ctx).Exec(`UPDATE users SET
		locked_until = CASE WHEN failed_logins + 1 >= ? THEN ?::timestamptz ELSE locked_until END,
		failed_logins = CASE WHEN failed_logins + 1 >= ? THEN 0 ELSE failed_logins + 1 END
		WHERE id = ?`, max, lockUntil, max, id).Error
}

func (r userRepo) ClearLoginFailures(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Model(&userModel{}).Where("id = ? AND (failed_logins <> 0 OR locked_until IS NOT NULL)", id).
		Updates(map[string]any{"failed_logins": 0, "locked_until": nil}).Error
}

func (r resetRepo) Since(ctx context.Context, uid string, since time.Time) (int64, int64, error) {
	var row struct{ Codes, Attempts int64 }
	err := r.db.WithContext(ctx).Model(&resetModel{}).Select("COUNT(*) AS codes, COALESCE(SUM(attempts), 0) AS attempts").
		Where("user_id = ? AND created_at >= ?", uid, since).Scan(&row).Error
	return row.Codes, row.Attempts, err
}

type sessionRepo struct{ db *gorm.DB }

func (r sessionRepo) Create(ctx context.Context, s *d.Session) error {
	return r.db.WithContext(ctx).Create(&sessionModel{ID: s.ID, UserID: s.UserID, ExpiresAt: s.ExpiresAt, CreatedAt: s.CreatedAt}).Error
}

func (r sessionRepo) ByID(ctx context.Context, id string) (*d.Session, error) {
	var m sessionModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error; err != nil {
		return nil, mapErr(err)
	}
	return &d.Session{ID: m.ID, UserID: m.UserID, ExpiresAt: m.ExpiresAt, RevokedAt: m.RevokedAt, CreatedAt: m.CreatedAt}, nil
}

func (r sessionRepo) Revoke(ctx context.Context, id string, at time.Time) error {
	return r.db.WithContext(ctx).Model(&sessionModel{}).Where("id = ? AND revoked_at IS NULL", id).Update("revoked_at", at).Error
}

func (r sessionRepo) RevokeUser(ctx context.Context, uid string, at time.Time) error {
	return r.db.WithContext(ctx).Model(&sessionModel{}).Where("user_id = ? AND revoked_at IS NULL", uid).Update("revoked_at", at).Error
}

func (r sessionRepo) Purge(ctx context.Context, before time.Time) (int64, error) {
	res := r.db.WithContext(ctx).Where("expires_at < ?", before).Delete(&sessionModel{})
	return res.RowsAffected, res.Error
}

func (r supportRepo) PushesSince(ctx context.Context, phone string, since time.Time) (int64, error) {
	var n *int64
	err := r.db.WithContext(ctx).Model(&supportModel{}).Select("SUM(pushes)").Where("payer_phone = ? AND created_at >= ?", phone, since).Scan(&n).Error
	if err != nil || n == nil {
		return 0, err
	}
	return *n, nil
}
