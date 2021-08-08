package fxiaoke

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/tidwall/gjson"
)

const (
	BaseURL = "https://open.fxiaoke.com"
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
	url := path.Join(BaseURL, endpoint)
	req, err = http.NewRequest(http.MethodPost, url, buf)
	if err != nil {
		return
	}

	var resp *http.Response
	resp, err = c.client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	var b []byte
	b, err = ioutil.ReadAll(req.Body)
	if err != nil {
		return
	}
	content = string(b)
	if resp.StatusCode != http.StatusOK || gjson.Get(content, "errorCode").Int() != 0 {
		err = fmt.Errorf("request err: [%d] %s", resp.StatusCode, content)
	}
	return
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

	url := path.Join(BaseURL, "/cgi/corpAccessToken/get/V2")
	req, err := http.NewRequest(http.MethodPost, url, buf)
	if err != nil {
		return err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return err
	}
	content := string(b)
	if gjson.Get(content, "errorCode").Int() != 0 {
		return fmt.Errorf("refresh corp access token failed: %s", content)
	}
	c.token = gjson.Get(content, "corpAccessToken").String()
	c.tokenExpire = gjson.Get(content, "expiresIn").Int()
	c.tokenRefreshTime = currentTime
	return nil
}
