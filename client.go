package fxiaoke

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/tidwall/gjson"
)

const (
	BaseURL = "https://open.fxiaoke.com"

	EndpointGetToken    = "/cgi/corpAccessToken/get/V2"
	responseSuccessCode = 0
)

var (
	defaultTimeout = time.Second * time.Duration(10)
	defaultBackOff = &backoff.ExponentialBackOff{
		InitialInterval:     200 * time.Millisecond,
		RandomizationFactor: backoff.DefaultRandomizationFactor,
		Multiplier:          backoff.DefaultMultiplier,
		MaxInterval:         5 * time.Second,
		MaxElapsedTime:      20 * time.Second,
		Clock:               backoff.SystemClock,
	}
)

type Client struct {
	client *BackOffClient

	appID         string
	appSecret     string
	permanentCode string
	corpID        string
	userID        string

	token            string
	tokenExpire      int64
	tokenRefreshTime int64
}

func NewClient(appID, appSecret, permanentCode, userID, corpID string) (*Client, error) {
	if appID == "" {
		return nil, errors.New("appID cannot be empty")
	}
	if appSecret == "" {
		return nil, errors.New("appSecret cannot be empty")
	}
	if permanentCode == "" {
		return nil, errors.New("permanentCode cannot be empty")
	}
	if userID == "" {
		return nil, errors.New("userID cannot be empty")
	}
	if corpID == "" {
		return nil, errors.New("corpID cannot be empty")
	}

	client := &Client{
		appID:         appID,
		appSecret:     appSecret,
		permanentCode: permanentCode,
		userID:        userID,
		corpID:        corpID,
		client: &BackOffClient{
			backOff: defaultBackOff,
			client: &http.Client{
				Timeout: defaultTimeout,
			},
		},
	}
	return client, nil
}

func (c *Client) Post(endpoint string, data map[string]interface{}, auth bool) (content string, err error) {
	if endpoint == "" {
		err = errors.New("endpoint cannot be empty")
		return
	}

	var buf *bytes.Buffer
	if data != nil {
		if auth {
			data["corpId"] = c.corpID
			data["currentOpenUserId"] = c.userID
			if err = c.RefreshAccessToken(); err != nil {
				return
			}
			data["corpAccessToken"] = c.token
		}
		buf = &bytes.Buffer{}
		err = json.NewEncoder(buf).Encode(data)
		if err != nil {
			return
		}
	}

	var req *http.Request
	url := concatURL(BaseURL, endpoint)
	req, err = http.NewRequest(http.MethodPost, url, buf)
	if err != nil {
		return
	}
	req.Header.Add("Content-Type", "application/json")

	var resp *http.Response
	resp, err = c.client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	var b []byte
	b, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	content = string(b)
	if resp.StatusCode != http.StatusOK || gjson.Get(content, "errorCode").Int() != responseSuccessCode {
		err = fmt.Errorf("Post %s err: %s, resp: %+v", url, content, resp)
	}
	return
}

func (c *Client) RawPost(endpoint string, data map[string]interface{}, auth bool) (*http.Response, error) {
	if endpoint == "" {
		return nil, errors.New("endpoint cannot be empty")
	}

	var buf *bytes.Buffer
	if data != nil {
		if auth {
			data["corpId"] = c.corpID
			data["currentOpenUserId"] = c.userID
			if err := c.RefreshAccessToken(); err != nil {
				return nil, err
			}
			data["corpAccessToken"] = c.token
		}
		buf = &bytes.Buffer{}
		if err := json.NewEncoder(buf).Encode(data); err != nil {
			return nil, err
		}
	}

	url := concatURL(BaseURL, endpoint)
	req, err := http.NewRequest(http.MethodPost, url, buf)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	return c.client.Do(req)
}

func (c *Client) RefreshAccessToken() error {
	currentTime := time.Now().Unix()
	if currentTime-c.tokenRefreshTime < c.tokenExpire {
		return nil
	}

	data := map[string]string{
		"appId":         c.appID,
		"appSecret":     c.appSecret,
		"permanentCode": c.permanentCode,
	}
	buf := &bytes.Buffer{}
	json.NewEncoder(buf).Encode(data)

	url := concatURL(BaseURL, EndpointGetToken)
	req, err := http.NewRequest(http.MethodPost, url, buf)
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	content := string(b)
	if resp.StatusCode != http.StatusOK || gjson.Get(content, "errorCode").Int() != responseSuccessCode {
		return fmt.Errorf("refresh corp access token failed: %s, resp: %+v", content, resp)
	}
	c.token = gjson.Get(content, "corpAccessToken").String()
	c.tokenExpire = gjson.Get(content, "expiresIn").Int()
	c.tokenRefreshTime = currentTime
	return nil
}

func concatURL(base, endpoint string) string {
	if !strings.HasPrefix(endpoint, "/") {
		endpoint = fmt.Sprintf("/%s", endpoint)
	}
	return fmt.Sprintf("%s%s", base, endpoint)
}

func GetEndpoint(objType, action string) (string, error) {
	switch objType {
	case ObjTypeCustom:
		return fmt.Sprintf("/cgi/crm/custom/v2/data/%s", action), nil
	case ObjTypePackage:
		return fmt.Sprintf("/cgi/crm/v2/data/%s", action), nil
	default:
		return "", fmt.Errorf("invalid obj type: %s", objType)
	}
}
