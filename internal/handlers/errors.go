package handlers

import "errors"

func errRequired(field string) error {
	return errors.New(field + " is required")
}

func errInvalidType(field string) error {
	return errors.New(field + " is invalid")
}
