package fxiaoke

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/cenkalti/backoff"
)

type BackOffClient struct {
	client  *http.Client
	backOff backoff.BackOff
	notify  func(error, time.Duration)
}

func (b *BackOffClient) Do(req *http.Request) (*http.Response, error) {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	req.Body.Close()

	var resp *http.Response
	op := func() (err error) {
		req.Body = ioutil.NopCloser(bytes.NewReader(body))
		resp, err = b.client.Do(req)
		if err != nil {
			err = fmt.Errorf("request err: %+v", err)
		}
		return err
	}
	return resp, backoff.RetryNotify(op, b.backOff, b.notify)
}
