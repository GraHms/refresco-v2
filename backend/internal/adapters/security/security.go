package security

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	d "github.com/refresco/refresco-v2/backend/internal/domain"
	"github.com/refresco/refresco-v2/backend/internal/ports"
)

type Bcrypt struct{ Cost int }

func (b Bcrypt) Hash(pw string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(pw), b.Cost)
	return string(h), err
}

func (b Bcrypt) Compare(hash, pw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}

type JWT struct {
	Secret []byte
	TTL    time.Duration
	Issuer string
}

type claims struct {
	Role string `json:"role"`
	Ver  int    `json:"ver"`
	jwt.RegisteredClaims
}

func (j JWT) Issue(u *d.User, sessionID string) (string, time.Time, error) {
	exp := time.Now().Add(j.TTL)
	c := claims{Role: string(u.Role), Ver: u.TokenVersion, RegisteredClaims: jwt.RegisteredClaims{
		ID: sessionID, Subject: u.ID, Issuer: j.Issuer, Audience: jwt.ClaimStrings{j.Issuer}, IssuedAt: jwt.NewNumericDate(time.Now()), ExpiresAt: jwt.NewNumericDate(exp),
	}}
	s, err := jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(j.Secret)
	return s, exp, err
}

func (j JWT) Parse(tok string) (*ports.Claims, error) {
	var c claims
	t, err := jwt.ParseWithClaims(tok, &c, func(t *jwt.Token) (any, error) { return j.Secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithIssuer(j.Issuer), jwt.WithAudience(j.Issuer), jwt.WithExpirationRequired())
	if err != nil || !t.Valid {
		return nil, errors.New("invalid token")
	}
	if c.ID == "" {
		return nil, errors.New("token without session")
	}
	return &ports.Claims{SessionID: c.ID, UserID: c.Subject, Role: d.Role(c.Role), TokenVersion: c.Ver, ExpiresAt: c.ExpiresAt.Time}, nil
}

type SystemClock struct{}

func (SystemClock) Now() time.Time { return time.Now().UTC() }
