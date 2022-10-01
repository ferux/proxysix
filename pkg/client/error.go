package client

import (
	"errors"
	"strconv"
)

func newGeneralError(msg string) error {
	return GeneralError{msg: msg}
}

type GeneralError struct {
	msg string
}

func (err GeneralError) Error() string {
	return err.msg
}

type RequestError struct {
	code int
	msg  string
}

func (err RequestError) Error() string {
	code := strconv.Itoa(err.code)
	if err.msg == "" {
		return "unexpected code " + code
	}

	return "unexpected code " + code + ": " + err.msg
}

func newParseError(name, reason string) ParseError {
	return ParseError{
		Name:   name,
		Reason: reason,
	}
}

type ParseError struct {
	Name   string
	Reason string
}

func (err ParseError) Error() string {
	return "parsing " + err.Name + ": " + err.Reason
}

func GetCode(err error) int {
	var rerr RequestError
	if errors.As(err, &rerr) {
		return rerr.code
	}

	return -1
}

func GetMessage(err error) string {
	var rerr RequestError
	if errors.As(err, &rerr) {
		return rerr.msg
	}

	var gerr GeneralError
	if errors.As(err, &gerr) {
		return gerr.msg
	}

	return ""
}
