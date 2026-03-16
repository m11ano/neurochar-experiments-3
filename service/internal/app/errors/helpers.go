package errors

import (
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ExtractError - ищет ближайшую ошибку AppError
func ExtractError(err error) (*AppError, bool) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr, true
	}

	return nil, false
}

// NearestHints - ищет ближайший непустой слайс подсказок
func NearestHints(err error) ([]string, bool) {
	for {
		if err == nil {
			return nil, false
		}
		if appErr, ok := err.(*AppError); ok {
			if len(appErr.hints) > 0 {
				return appErr.hints, true
			}
			err = appErr.Unwrap()
			continue
		}
		if chainErr, ok := err.(*ChainError); ok {
			err = chainErr.Unwrap()
			continue
		}
		err = errors.Unwrap(err)
	}
}

// NearestErrMsg - ищет только сообщение об ошибке в AppError
func NearestErrMsg(err error) (string, bool) {
	for {
		if err == nil {
			return "", false
		}
		if appErr, ok := err.(*AppError); ok {
			if len(appErr.errMsg) > 0 {
				return appErr.errMsg, true
			}
			err = appErr.Unwrap()
			continue
		}
		if chainErr, ok := err.(*ChainError); ok {
			err = chainErr.Unwrap()
			continue
		}
		err = errors.Unwrap(err)
	}
}

// NearestError - ищет либо сообщение об ошибке в Error, либо текст сторонней ошибки
func NearestError(err error) (string, bool) {
	for {
		if err == nil {
			return "", false
		}
		if appErr, ok := err.(*AppError); ok {
			if len(appErr.errMsg) > 0 {
				return appErr.errMsg, true
			}
			err = appErr.Unwrap()
			continue
		}
		if chainErr, ok := err.(*ChainError); ok {
			err = chainErr.Unwrap()
			continue
		}
		if err.Error() != "" {
			return err.Error(), true
		}
		err = errors.Unwrap(err)
	}
}

// WithHints - добавляет подсказки к error.
// Ищет ближайший AppError и копируется. Если не найдено - создается копия ErrInternal
func WithHints(err error, hints ...string) *AppError {
	var foundAppErr *AppError

	lookingErr := err
	for lookingErr != nil {

		if appErr, ok := lookingErr.(*AppError); ok {
			foundAppErr = appErr
			break
		}
		if chainErr, ok := lookingErr.(*ChainError); ok {
			lookingErr = chainErr.Unwrap()
			continue
		}
		lookingErr = errors.Unwrap(lookingErr)
	}

	var cp *AppError
	if foundAppErr != nil {
		cp = foundAppErr.Copy()
	} else {
		cp = ErrInternal.Copy()
	}
	cp.parent = err
	cp.hints = hints

	return cp
}

// ToGrpcStatus - преобразует error в GrpcStatus
func ToGrpcStatus(err error) error {
	if err == nil {
		return nil
	}

	appErr, ok := ExtractError(err)
	if !ok {
		st, ok := status.FromError(err)
		if !ok {
			return status.Error(codes.Unknown, err.Error())
		}

		return st.Err()
	}

	hint := ""
	hints, ok := NearestHints(err)
	if ok {
		hint = strings.Join(hints, "; ")
	}

	code := codes.Unknown
	textCode := appErr.Meta().TextCode

	switch appErr.Meta().Code {
	case errorCodeBadRequest:
		code = codes.InvalidArgument
	case errorCodeForbidden:
		code = codes.PermissionDenied
	case errorCodeNotFound:
		code = codes.NotFound
	case errorCodeConflict:
		code = codes.Aborted
	case errorTooManyRequests:
		code = codes.ResourceExhausted
	case errorCodeInternal:
		code = codes.Internal
	}

	detail := &errdetails.ErrorInfo{
		Reason: textCode,
	}

	st, stErr := status.New(code, hint).WithDetails(detail)
	if stErr != nil {
		return status.Error(codes.Unknown, stErr.Error())
	}

	return st.Err()
}

// CheckIsTxСoncurrentExec - проверяет, является ли ошибка pgx о конкурентном выполнении транзакции
func CheckIsTxСoncurrentExec(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && (pgErr.Code == "40001" || pgErr.Code == "25P02") {
		return true
	}
	return errors.Is(err, ErrTxСoncurrentExec)
}

// ConvertPgxToAppErr - конвертирует ошибку pgx в ошибку приложения
func ConvertPgxToAppErr(err error) (error, bool) {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrStoreNoRows.WithWrap(err), true
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "40001":
			return ErrTxСoncurrentExec.WithWrap(err), true
		case "25P02":
			return ErrTxСoncurrentExec.WithWrap(err), true
		case "23505":
			return ErrStoreUniqueViolation.WithWrap(err).WithDetail("column", true, pgErr.ColumnName), true
		case "23503":
			return ErrStoreForeignKeyViolation.WithWrap(err).WithDetail("column", true, pgErr.ColumnName), true
		case "23502":
			return ErrStoreNotNullViolation.WithWrap(err).WithDetail("column", true, pgErr.ColumnName), true
		case "23514":
			return ErrStoreCheckViolation.WithWrap(err).WithDetail("constraint", true, pgErr.ConstraintName), true
		case "23001":
			return ErrStoreRestrictViolation.WithWrap(err).WithDetail("constraint", true, pgErr.ConstraintName), true
		case "23000":
			return ErrStoreIntegrityViolation.WithWrap(err).WithDetail("constraint", true, pgErr.ConstraintName), true
		default:
			return ErrInternal.WithWrap(err), false
		}
	}
	return err, false
}
