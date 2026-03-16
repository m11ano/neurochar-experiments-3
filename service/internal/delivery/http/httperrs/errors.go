package httperrs

import (
	appErrors "github.com/m11ano/neurochar-experiments-3/service/internal/app/errors"
)

var ErrCantParseBody = appErrors.ErrBadRequest.Extend("cannot parse request body").WithHints("cannot parse request body")

var ErrValidation = appErrors.ErrBadRequest.Extend("validation")
