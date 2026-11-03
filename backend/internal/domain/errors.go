package domain

import "errors"

type Error struct {
	Code    string
	Message string
	Kind    Kind
}

type Kind int

const (
	KindInvalid Kind = iota + 1
	KindNotFound
	KindConflict
	KindUnauthorized
	KindForbidden
	KindUnavailable
	KindTooMany
)

func (e *Error) Error() string { return e.Code + ": " + e.Message }

func newErr(kind Kind, code, msg string) *Error { return &Error{Code: code, Message: msg, Kind: kind} }

func Invalid(code, msg string) *Error     { return newErr(KindInvalid, code, msg) }
func NotFound(code, msg string) *Error    { return newErr(KindNotFound, code, msg) }
func Conflict(code, msg string) *Error    { return newErr(KindConflict, code, msg) }
func Forbidden(code, msg string) *Error   { return newErr(KindForbidden, code, msg) }
func Unavailable(code, msg string) *Error { return newErr(KindUnavailable, code, msg) }

var (
	ErrNotFound           = NotFound("not_found", "Não encontrado.")
	ErrUnauthorized       = newErr(KindUnauthorized, "unauthorized", "Sessão inválida ou expirada.")
	ErrForbidden          = Forbidden("forbidden", "Sem permissão para esta ação.")
	ErrInvalidCredentials = newErr(KindUnauthorized, "invalid_credentials", "Email ou palavra-passe incorretos.")
	ErrTooManyAttempts    = newErr(KindTooMany, "too_many_attempts", "Demasiadas tentativas. Tenta mais tarde.")
	ErrStaleState         = Conflict("stale_state", "O estado mudou entretanto. Atualiza e tenta de novo.")
)

func AsError(err error) (*Error, bool) {
	var e *Error
	ok := errors.As(err, &e)
	return e, ok
}
