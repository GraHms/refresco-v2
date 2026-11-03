package domain

import (
	"strings"
	"time"
)

type Role string

const (
	RoleCreator    Role = "creator"
	RoleSuperAdmin Role = "superadmin"
)

type User struct {
	ID           string
	Email        string
	Phone        string
	Name         string
	PasswordHash string
	Role         Role
	TokenVersion int
	CreatedAt    time.Time
}

type AccountStatus string

const (
	AccountActive    AccountStatus = "active"
	AccountPending   AccountStatus = "pending"
	AccountSuspended AccountStatus = "suspended"
)

type ChannelSet struct {
	Mpesa bool `json:"mpesa"`
	Emola bool `json:"emola"`
	Card  bool `json:"card"`
}

func (c ChannelSet) Has(ch Channel) bool {
	switch ch {
	case ChannelMpesa:
		return c.Mpesa
	case ChannelEmola:
		return c.Emola
	case ChannelCard:
		return c.Card
	}
	return false
}

func (c ChannelSet) Any() bool { return c.Mpesa || c.Emola || c.Card }

var SocialNetworks = []string{"youtube", "spotify", "instagram", "tiktok", "facebook", "x"}

var Categories = []string{"Podcast", "Música", "Arte e ilustração", "Comédia", "Educação", "Vídeo e streaming"}

type Account struct {
	ID          string
	OwnerID     string
	Handle      string
	Name        string
	Category    string
	Bio         string
	Thanks      string
	Amounts     []Money
	DefaultIdx  int
	AllowCustom bool
	MinAmount   Money
	Socials     map[string]string
	Channels    ChannelSet
	FeeBPS      BPS
	FeeNote     string
	FeeUntil    *time.Time
	Status      AccountStatus
	StatusNote  string
	InviteCode  string
	AutoSplit   bool
	CreatedAt   time.Time
}

func (a *Account) EffectiveFee(now time.Time) BPS {
	if a.FeeUntil != nil && now.After(*a.FeeUntil) {
		return DefaultFeeBPS
	}
	return a.FeeBPS
}

func (a *Account) Negotiated(now time.Time) bool { return a.EffectiveFee(now) != DefaultFeeBPS }

const (
	MaxPresetAmount = 1_000_000 * MT
	MinCustomFloor  = 10 * MT
)

// ValidatePage checks the creator-editable settings.
func (a *Account) ValidatePage() error {
	a.Name = strings.TrimSpace(a.Name)
	if l := len([]rune(a.Name)); l < 2 || l > 40 {
		return Invalid("invalid_name", "O nome da página deve ter entre 2 e 40 caracteres.")
	}
	if !ValidHandle(a.Handle) {
		return Invalid("invalid_handle", "Usa 3 a 24 letras minúsculas, números ou pontos.")
	}
	if !contains(Categories, a.Category) {
		return Invalid("invalid_category", "Categoria inválida.")
	}
	if len([]rune(a.Bio)) > 200 || len([]rune(a.Thanks)) > 200 {
		return Invalid("text_too_long", "A bio e a mensagem de agradecimento têm no máximo 200 caracteres.")
	}
	if len(a.Amounts) != 4 {
		return Invalid("invalid_amounts", "Define quatro valores sugeridos.")
	}
	for _, v := range a.Amounts {
		if v < MinCustomFloor || v > MaxPresetAmount || v%MT != 0 {
			return Invalid("invalid_amounts", "Os valores sugeridos devem estar entre 10 e 1.000.000 MT.")
		}
	}
	if a.DefaultIdx < 0 || a.DefaultIdx > 3 {
		return Invalid("invalid_default", "Valor sugerido inválido.")
	}
	if a.MinAmount < MinCustomFloor || a.MinAmount > MaxPresetAmount || a.MinAmount%MT != 0 {
		return Invalid("invalid_min", "O mínimo deve estar entre 10 e 1.000.000 MT.")
	}
	if !a.Channels.Any() {
		return Invalid("no_channel", "Ativa pelo menos uma forma de pagamento.")
	}
	clean := map[string]string{}
	for _, k := range SocialNetworks {
		v := strings.TrimSpace(a.Socials[k])
		if v == "" {
			continue
		}
		if len(v) > 80 || strings.ContainsAny(v, " <>\"'`") {
			return Invalid("invalid_social", "Utilizador de rede social inválido: "+k+".")
		}
		clean[k] = v
	}
	a.Socials = clean
	return nil
}

type MemberStatus string

const (
	MemberActive  MemberStatus = "active"
	MemberInvited MemberStatus = "invited"
)

type Member struct {
	ID           string
	AccountID    string
	UserID       *string
	Name         string
	Role         string
	Pct          int
	Wallet       Channel
	WalletNumber string
	IsAdmin      bool
	Status       MemberStatus
	Balance      Money
	CreatedAt    time.Time
}

type PayoutKind string

const (
	PayoutMpesa PayoutKind = "mpesa"
	PayoutEmola PayoutKind = "emola"
	PayoutNIB   PayoutKind = "nib"
)

type PayoutAccount struct {
	ID        string
	MemberID  string
	Kind      PayoutKind
	Number    string
	BankName  string
	Holder    string
	IsMain    bool
	CreatedAt time.Time
}

var nibDigits = 21

func ValidateNIB(s string) string {
	d := nonDigit.ReplaceAllString(s, "")
	if len(d) != nibDigits {
		return ""
	}
	return d
}

type SupportStatus string

const (
	SupportPending   SupportStatus = "pending"
	SupportConfirmed SupportStatus = "confirmed"
	SupportFailed    SupportStatus = "failed"
	SupportExpired   SupportStatus = "expired"
)

type Support struct {
	ID            string
	Ref           string
	AccountID     string
	Amount        Money
	FeeBPS        BPS
	Fee           Money
	Net           Money
	Preset        bool
	Channel       Channel
	PayerPhone    string
	CardBrand     string
	CardLast4     string
	SupporterName string
	SupporterNet  string
	Message       string
	Status        SupportStatus
	FailReason    string
	ProviderRef   string
	NextAction    string
	ExpiresAt     time.Time
	IdemKey       string
	ReadAt        *time.Time
	ConfirmedAt   *time.Time
	CreatedAt     time.Time
}

type SupportSplit struct {
	SupportID string
	MemberID  string
	Name      string
	Pct       int
	Amount    Money
}

type WithdrawalStatus string

const (
	WithdrawalRequested WithdrawalStatus = "requested"
	WithdrawalApproved  WithdrawalStatus = "approved"
	WithdrawalPaid      WithdrawalStatus = "paid"
	WithdrawalRejected  WithdrawalStatus = "rejected"
)

func (s WithdrawalStatus) Open() bool { return s == WithdrawalRequested || s == WithdrawalApproved }

const (
	MinWithdrawal = 500 * MT
	WithdrawalSLA = 72 * time.Hour
)

type Withdrawal struct {
	ID          string
	Number      int64
	AccountID   string
	MemberID    string
	Amount      Money
	DestKind    PayoutKind
	DestNumber  string
	DestBank    string
	DestHolder  string
	Status      WithdrawalStatus
	Reason      string
	Note        string
	TransferRef string
	DueAt       time.Time
	DecidedAt   *time.Time
	PaidAt      *time.Time
	DecidedBy   string
	CreatedAt   time.Time
}

func (w *Withdrawal) Code() string { return "LV-" + itoa(1000+w.Number) }

type InviteStatus string

const (
	InviteActive  InviteStatus = "active"
	InviteRevoked InviteStatus = "revoked"
)

type Invite struct {
	Code      string
	Note      string
	MaxUses   int
	Uses      int
	FeeBPS    BPS
	Status    InviteStatus
	CreatedAt time.Time
}

func (i *Invite) Usable() bool { return i.Status == InviteActive && i.Uses < i.MaxUses }

type MemberPct struct {
	MemberID string `json:"member_id"`
	Pct      int    `json:"pct"`
}

type ProposalStatus string

const (
	ProposalPending   ProposalStatus = "pending"
	ProposalApplied   ProposalStatus = "applied"
	ProposalCancelled ProposalStatus = "cancelled"
)

type SplitProposal struct {
	ID        string
	AccountID string
	Splits    []MemberPct
	Status    ProposalStatus
	CreatedBy string
	CreatedAt time.Time
}

type SplitApproval struct {
	ProposalID string
	MemberID   string
	TokenHash  string
	ApprovedAt *time.Time
}

func ValidateSplits(members []Member, splits []MemberPct) error {
	if len(splits) != len(members) {
		return Invalid("invalid_split", "A divisão tem de incluir todos os membros.")
	}
	ids := map[string]bool{}
	for _, m := range members {
		ids[m.ID] = true
	}
	total := 0
	for _, s := range splits {
		if !ids[s.MemberID] || s.Pct < 0 || s.Pct > 100 {
			return Invalid("invalid_split", "Divisão inválida.")
		}
		delete(ids, s.MemberID)
		total += s.Pct
	}
	if total != 100 {
		return Invalid("split_not_100", "A divisão tem de somar 100%.")
	}
	return nil
}

type AuditEntry struct {
	ID        string
	AccountID string
	Scope     string
	Text      string
	CreatedAt time.Time
}

type PasswordReset struct {
	ID             string
	UserID         string
	CodeHash       string
	ExpiresAt      time.Time
	Attempts       int
	ResetTokenHash string
	VerifiedAt     *time.Time
	UsedAt         *time.Time
	CreatedAt      time.Time
}

func contains(xs []string, v string) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
