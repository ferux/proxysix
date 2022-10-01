package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ferux/proxysix/pkg/entities"
)

const (
	defaultAddr = "https://proxy6.net"
)

type ProxyState uint8

const (
	ProxyStateUnknown ProxyState = iota
	ProxyStateActive
	ProxyStateExpired
	ProxyStateExpiring
	ProxyStateAll
)

func ParseProxyState(value string) (state ProxyState, err error) {
	switch strings.ToLower(value) {
	case ProxyStateAll.String():
		state = ProxyStateAll
	case ProxyStateExpiring.String():
		state = ProxyStateExpiring
	case ProxyStateExpired.String():
		state = ProxyStateExpired
	case ProxyStateActive.String():
		state = ProxyStateActive
	default:
		return state, newParseError("state", "unsupported value "+value)
	}

	return state, nil
}

func (s ProxyState) MarshalText() string {
	return s.String()
}

func (s ProxyState) String() string {
	switch s {
	case ProxyStateAll:
		return "all"
	case ProxyStateExpiring:
		return "expiring"
	case ProxyStateExpired:
		return "expired"
	case ProxyStateActive:
		return "active"
	default:
		return ""
	}
}

type Client interface {
	ListProxies(ctx context.Context, params ListProxyParams) ([]entities.Proxy, error)
}

type httpClient struct {
	addr string
	key  string

	hclient *http.Client
	logF    GetLogger
}

type options struct {
}

type optionF func(*httpClient) error

func WithAddr(addr string) optionF {
	return func(hc *httpClient) error {
		if addr == "" {
			return newGeneralError("addr is empty")
		}

		hc.addr = addr

		return nil
	}
}

func WithLoggerFunc(logFunc GetLogger) optionF {
	return func(hc *httpClient) error {
		if logFunc == nil {
			return newGeneralError("get logger func is nil")
		}

		hc.logF = logFunc

		return nil
	}
}

func NewHTTPClient(_ context.Context, apiKey string, opts ...optionF) (*httpClient, error) {
	hc := &httpClient{
		addr:    defaultAddr,
		key:     apiKey,
		hclient: http.DefaultClient,
		logF:    GetLoggerNoop,
	}

	for idx, opt := range opts {
		if err := opt(hc); err != nil {
			return nil, fmt.Errorf("applying %d opt: %w", idx, err)
		}
	}

	return hc, nil
}

type proxy struct {
	ID          string `json:"id"`
	Host        string `json:"host"`
	Port        string `json:"port"`
	User        string `json:"user"`
	Pass        string `json:"pass"`
	Type        string `json:"type"`
	Country     string `json:"country"`
	Date        string `json:"date"`
	DateEnd     string `json:"date_end"`
	Unixtime    int64  `json:"unixtime"`
	UnixtimeEnd int64  `json:"unixtime_end"`
	Descr       string `json:"descr"`
	Active      string `json:"active"`
}

type baseResponse struct {
	Status  string `json:"status"`
	ErrorID int    `json:"error_id"`
	Error   string `json:"error"`
}

func (b baseResponse) isError() bool {
	return b.Status == "no" || b.ErrorID > 0 || b.Error != ""
}

type proxyResponse struct {
	baseResponse

	ListCount int
	List      []proxy
}

type ListProxyParams struct {
	Descr string
	State ProxyState
}

func (c *httpClient) ListProxies(ctx context.Context, params ListProxyParams) ([]entities.Proxy, error) {
	resp, err := c.loadAllProxies(ctx, params.State, params.Descr)
	if err != nil {
		return nil, err
	}

	proxies, err := mapProxyResponse(resp)
	if err != nil {
		return nil, err
	}

	return proxies, nil
}

func (c *httpClient) loadAllProxies(ctx context.Context, state ProxyState, desc string) (response proxyResponse, err error) {
	const pathTemplate = "/api/{api_key}/getproxy"

	reqURL := c.addr + strings.ReplaceAll(pathTemplate, "{api_key}", c.key)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return response, fmt.Errorf("making request: %w", err)
	}

	values := make(url.Values, 3)
	values.Set("state", state.String())
	values.Set("descr", desc)
	values.Set("nokey", "")

	request.URL.RawQuery = values.Encode()

	log := c.logF(ctx)

	err = doRequest(c.hclient, request, log, &response)
	if err != nil {
		return response, err
	}

	if response.isError() {
		return response, RequestError{
			code: response.ErrorID,
			msg:  response.Error,
		}
	}

	return response, nil
}

func doRequest(c *http.Client, req *http.Request, log Logger, dst any) error {
	log.Debug("sending request", LogField("req_url", req.URL.String()))

	begin := time.Now()

	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("doing request: %w", err)
	}

	body, _ := ioutil.ReadAll(resp.Body)
	log.Info(
		"request processed",
		LogField("code", resp.StatusCode),
		LogField("duration", time.Since(begin)),
		LogField("body", string(body)),
	)

	if resp.StatusCode >= http.StatusBadRequest {
		// body, _ := ioutil.ReadAll(resp.Body)

		return RequestError{
			code: resp.StatusCode,
			msg:  string(body),
		}
	}

	// err = json.NewDecoder(resp.Body).Decode(dst)
	err = json.Unmarshal(body, dst)
	if err != nil {
		return fmt.Errorf("decoding json: %w", err)
	}

	return nil
}

func mapProxyResponse(response proxyResponse) ([]entities.Proxy, error) {
	proxies := make([]entities.Proxy, 0, response.ListCount)
	for _, item := range response.List {
		proxy, err := mapProxyToEntity(item)
		if err != nil {
			return nil, err
		}

		proxies = append(proxies, proxy)
	}

	return proxies, nil
}

func mapProxyToEntity(proxy proxy) (entities.Proxy, error) {
	port, err := strconv.Atoi(proxy.Port)
	if err != nil {
		return entities.Proxy{}, fmt.Errorf("parsing port: %w", err)
	}

	proxyType, err := entities.ParseProxyType(proxy.Type)
	if err != nil {
		return entities.Proxy{}, fmt.Errorf("parsing proxy type: %w", err)
	}

	entityProxy := entities.Proxy{
		ID:          proxy.ID,
		Host:        proxy.Host,
		Port:        uint16(port),
		User:        proxy.User,
		Password:    entities.NewSensitive(proxy.Pass),
		Type:        proxyType,
		Country:     proxy.Country,
		Date:        time.Unix(proxy.Unixtime, 0),
		ExpireDate:  time.Unix(proxy.UnixtimeEnd, 0),
		Description: proxy.Descr,
		Active:      strings.EqualFold(proxy.Active, "1"),
	}

	return entityProxy, nil
}
