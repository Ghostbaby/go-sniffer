package client

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	SnifferBackendUpstream = "sniffer-backend"
)

type BaseClient struct {
	HTTP       *http.Client
	Endpoint   string
	Transport  *http.Transport
	Token      string
	User       string
	MetricInfo *struct {
		Name      string // 系统名称
		MatchUrl  string // 访问路径
		ClassType string // 业务类型 up: 上游，down:下游，callback:回调
	}
	Headers []Header // headers
}

type Header struct {
	Key   string
	Value string
}

func NewBaseClient(url string, timeout time.Duration) *BaseClient {
	hClient := &http.Client{
		Transport: &http.Transport{},
	}
	if timeout > 0 {
		hClient.Timeout = timeout
	}

	client := &BaseClient{
		HTTP:      hClient,
		Endpoint:  url,
		Transport: &http.Transport{},
	}

	return client
}

// WithMetricInfo 指标上报
func (c *BaseClient) WithMetricInfo(sysName, typ, matchUrl string) *BaseClient {
	c.MetricInfo = &struct {
		Name      string
		MatchUrl  string
		ClassType string
	}{
		Name:      sysName,
		ClassType: typ,
		MatchUrl:  matchUrl,
	}
	return c
}

func (c *BaseClient) WithHeader(key string, value string) *BaseClient {
	if c.Headers == nil {
		c.Headers = []Header{{Key: key, Value: value}}
		return c
	}
	c.Headers = append(c.Headers, Header{Key: key, Value: value})
	return c
}

func (c *BaseClient) WithTokenAndUser(token, user string) *BaseClient {
	c.Token = token
	c.User = user
	return c
}

func (c *BaseClient) Get(ctx context.Context, pathWithQuery string, out interface{}) error {
	var err error
	_, err = c.request(ctx, http.MethodGet, pathWithQuery, nil, out, nil)
	return err
}

func (c *BaseClient) Put(ctx context.Context, pathWithQuery string, in, out interface{}) error {
	var err error
	_, err = c.request(ctx, http.MethodPut, pathWithQuery, in, out, nil)
	return err
}

func (c *BaseClient) Post(ctx context.Context, pathWithQuery string, in, out interface{}) error {
	var err error
	_, err = c.request(ctx, http.MethodPost, pathWithQuery, in, out, nil)
	return err
}

func (c *BaseClient) Delete(ctx context.Context, pathWithQuery string, in, out interface{}) error {
	var err error
	_, err = c.request(ctx, http.MethodDelete, pathWithQuery, in, out, nil)
	return err
}

func (c *BaseClient) Import(ctx context.Context, pathWithQuery string, in []byte, out interface{}) error {
	_, err := c.request(ctx, http.MethodPost, pathWithQuery, nil, out, in)
	return err
}

func (c *BaseClient) Exporter(ctx context.Context, pathWithQuery string) ([]byte, error) {
	return c.request(ctx, http.MethodGet, pathWithQuery, nil, nil, nil)
}

func (c *BaseClient) request(
	ctx context.Context,
	method string,
	pathWithQuery string,
	requestObj,
	responseObj interface{},
	rawRequestObj []byte,
) (raw []byte, err error) {
	var body io.Reader = http.NoBody
	if requestObj != nil {
		var outData []byte
		outData, err = json.Marshal(requestObj)
		if err != nil {
			return
		}
		body = bytes.NewBuffer(outData)
	}

	if rawRequestObj != nil {
		body = bytes.NewBuffer(rawRequestObj)
	}

	var request *http.Request
	request, err = http.NewRequest(method, Joins(c.Endpoint, pathWithQuery), body)
	if err != nil {
		return
	}

	if rawRequestObj == nil {
		request.Header.Add("Content-Type", "application/json")
	} else {
		request.Header.Add("Content-Length", strconv.Itoa(len(rawRequestObj)))
	}
	// 注入headers
	for _, obj := range c.Headers {
		request.Header.Add(obj.Key, obj.Value)
	}
	if len(c.Token) != 0 {
		request.Header.Add("X-Token", c.Token)
	}

	if len(c.User) != 0 {
		request.Header.Add("X-User", c.User)
	}

	var resp *http.Response
	resp, err = c.doRequest(ctx, request)
	if err != nil {
		return
	}

	defer resp.Body.Close()

	if responseObj != nil {
		err = json.NewDecoder(resp.Body).Decode(responseObj)
		if err != nil {
			return
		}
		return
	}

	raw, err = io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	return
}

func (c *BaseClient) doRequest(context context.Context, request *http.Request) (*http.Response, error) {
	withContext := request.WithContext(context)

	response, err := c.HTTP.Do(withContext)
	if err != nil {
		return response, err
	}

	err = checkError(response)
	return response, err
}

func checkError(response *http.Response) error {
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		data, _ := ioutil.ReadAll(response.Body)
		return errors.New(
			// fmt.Sprintf("请求执行失败, 返回码 %d error: %s", response.StatusCode, string(data)))
			string(data))
	}
	return nil
}

func (c *BaseClient) Close() {
	if c.Transport != nil {
		// When the http transport goes out of scope, the underlying goroutines responsible
		// for handling keep-alive connections are not closed automatically.
		// Since this client gets recreated frequently we would effectively be leaking goroutines.
		// Let's make sure this does not happen by closing idle connections.
		c.Transport.CloseIdleConnections()
	}
}

func (c *BaseClient) Equal(c2 *BaseClient) bool {
	// handle nil case
	if c2 == nil && c != nil {
		return false
	}

	// compare endpoint and user creds
	return c.Endpoint == c2.Endpoint
}

func Joins(args ...string) string {
	var str strings.Builder
	for _, arg := range args {
		str.WriteString(arg)
	}
	return str.String()
}

func AddQueryParam(params, param string) string {
	if !strings.Contains(params, "?") {
		params = params + "?" + param
		return params
	}

	return params + "&" + param
}

// Response 返回结构, 和paas保持一致
type Response struct {
	Success   bool        `json:"success"`
	ErrorCode int         `json:"errorCode"`
	ErrorMsg  string      `json:"errorMsg"`
	Data      interface{} `json:"data,omitempty"`
}

func ResponseOk(c *gin.Context, data interface{}) {
	resp := &Response{
		Success:   true,
		ErrorCode: http.StatusOK,
		ErrorMsg:  "",
		Data:      data,
	}
	c.JSON(http.StatusOK, resp)
}

func ResponseErrorCode(c *gin.Context, errCode int, errMsg string) {
	resp := &Response{
		Success:   false,
		ErrorCode: errCode,
		ErrorMsg:  errMsg,
		Data:      nil,
	}
	c.JSON(http.StatusOK, resp)
}

// ResponseErrorCodeWithHttpCode 同一错误返回体，将业务错误码与http状态码传入统一封装
func ResponseErrorCodeWithHttpCode(c *gin.Context, errCode int, errMsg string, httpCode int) {
	resp := &Response{
		Success:   false,
		ErrorCode: errCode,
		ErrorMsg:  errMsg,
		Data:      nil,
	}
	c.JSON(httpCode, resp)
	c.Abort()
}
