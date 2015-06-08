package pulse

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

//Tests if we can fetch a file from S3 or not...
func TestCurlS3(t *testing.T) {
	req := &CurlRequest{
		Path:     "/tb-minion/latest",
		Endpoint: "s3.amazonaws.com",
		Host:     "s3.amazonaws.com",
		Ssl:      true,
	}
	resp := CurlImpl(req)
	if resp.Err != "" {
		t.Error(resp.Err)
	}
	if resp.Status != 200 {
		t.Error("Status should be 200... got ", resp.Status)
	}
}

//Tests if we can override host header...
func TestCurlInvalidS3(t *testing.T) {
	req := &CurlRequest{
		Path:     "/tb-minion/latest",
		Endpoint: "s3.amazonaws.com",
		Host:     "www.turbobytes.com", //Bogus Host header not configured with S3
		Ssl:      false,
	}
	resp := CurlImpl(req)
	if resp.Err != "" {
		t.Error(resp.Err)
	}
	if resp.Status != 404 {
		t.Error("Status should be 404... got ", resp.Status)
	}
}

//Tests if a local url is being blocked correctly or not...
func TestCurlLocalBlock(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello, client")
	}))
	defer ts.Close()
	url, _ := url.Parse(ts.URL)
	req := &CurlRequest{
		Path:     "/tb-minion/latest",
		Endpoint: url.Host,
		Host:     url.Host,
		Ssl:      false,
	}
	resp := CurlImpl(req)
	if resp.Err != securityerr.Error() {
		t.Error("Security err should have been raised")
	}
}
