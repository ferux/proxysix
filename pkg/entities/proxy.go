package entities

import (
	"fmt"
	"time"
)

type Error string

func (e Error) Error() string {
	return string(e)
}

const (
	ErrWrongProxyType Error = "wrong proxy type"
)

type ProxyType uint8

const (
	ProxyTypeUnknown ProxyType = iota
	ProxyTypeHTTP
	ProxyTypeSocks
)

func (t ProxyType) MarshalText() ([]byte, error) {
	switch t {
	case ProxyTypeHTTP:
		return []byte("http"), nil
	case ProxyTypeSocks:
		return []byte("socks"), nil
	default:
		return nil, fmt.Errorf("type %d: %w", t, ErrWrongProxyType)
	}
}

func ParseProxyType(value string) (ProxyType, error) {
	switch value {
	case "http":
		return ProxyTypeHTTP, nil
	case "socks":
		return ProxyTypeSocks, nil
	default:
		return ProxyTypeUnknown, fmt.Errorf("type %s: %w", value, ErrWrongProxyType)
	}
}

type Proxy struct {
	ID          string
	Host        string
	Port        uint16
	User        string
	Password    Sensitive[string]
	Type        ProxyType
	Country     string
	Date        time.Time
	ExpireDate  time.Time
	Description string
	Active      bool
}

func (p *Proxy) ProxyURL() string {
	var scheme string
	switch p.Type {
	case ProxyTypeHTTP:
		scheme = "http"
	case ProxyTypeSocks:
		scheme = "socks5"
	}

	return fmt.Sprintf("%s://%s:%s@%s:%d", scheme, p.User, p.Password.value, p.Host, p.Port)
}

func NewSensitive[T any](value T) Sensitive[T] {
	return Sensitive[T]{
		value: value,
	}
}

type Sensitive[T any] struct {
	value T
	allow bool
}

func (s Sensitive[T]) MarshalText() (text []byte, err error) {
	if s.allow {
		return []byte(fmt.Sprintf("%v", s.value)), nil
	}

	return []byte("***"), nil
}

func (s *Sensitive[T]) Allow() {
	s.allow = true
}
