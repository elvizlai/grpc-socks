/**
 * Copyright 2015-2017, Wothing Co., Ltd.
 * All rights reserved.
 *
 * Created by elvizlai on 2017/12/1 14:07.
 */

package lib

import (
	"context"
	"crypto/tls"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"golang.org/x/net/http2"
)

var client = &http.Client{}

func TestName(t *testing.T) {
	ts := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	t.Logf("%#v", ts)

	err := http2.ConfigureTransport(ts)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%#v", ts)

	client.Transport = ts

	req, err := http.NewRequest("GET", "https://47.74.156.21:10465", nil)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cFun := context.WithTimeout(req.Context(), time.Second*2)
	defer cFun()

	req = req.WithContext(ctx)

	t.Log(req)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(resp)

	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(string(data))

}
