package utils

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New(validator.WithRequiredStructEnabled())

// ValidateStruct runs struct-tag validation and returns a readable message.
func ValidateStruct(s interface{}) error {
	if err := validate.Struct(s); err != nil {
		var verrs validator.ValidationErrors
		if errors.As(err, &verrs) && len(verrs) > 0 {
			ve := verrs[0]
			return fmt.Errorf("field %q failed on %q", ve.Field(), ve.Tag())
		}
		return err
	}
	return nil
}

// ValidURL enforces http/https and the 2048-char cap (§10.2); validator's
// `url` tag alone would accept ftp:// and other schemes.
func ValidURL(raw string) error {
	if raw == "" {
		return fmt.Errorf("url is required")
	}
	if len(raw) > 2048 {
		return fmt.Errorf("url exceeds 2048 characters")
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return fmt.Errorf("url must use http or https")
	}
	if u.Host == "" {
		return fmt.Errorf("url must include a host")
	}
	return nil
}
