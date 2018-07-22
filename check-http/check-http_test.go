package main

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sensu-go-plugins/gunsen/plugin"
)

func verifyExitCode(t *testing.T, got error, want int) {
	t.Helper()

	if got != nil {
		if exit, ok := got.(*plugin.Exit); ok {
			if exit.Status != want {
				t.Errorf("exit status = %v, want %v", exit.Status, want)
			}
		} else {
			t.Error("exit could not be asserted")
		}
	} else {
		t.Error("exit was nil")
	}
}

func TestHandleResponse(t *testing.T) {
	type fields struct {
		redirectOK   bool
		responseCode int
	}
	tests := []struct {
		name       string
		fields     fields
		resp       *http.Response
		wantStatus int
	}{
		{
			name:       "200 OK",
			resp:       &http.Response{StatusCode: http.StatusOK},
			wantStatus: plugin.OK,
		},
		{
			name:       "400 Bad Request",
			resp:       &http.Response{StatusCode: http.StatusBadRequest},
			wantStatus: plugin.Critical,
		},
		{
			name:       "Redirection not allowed",
			resp:       &http.Response{StatusCode: http.StatusMovedPermanently},
			wantStatus: plugin.Warning,
		},
		{
			name: "Expected 200 OK",
			fields: fields{
				responseCode: http.StatusOK,
			},
			resp:       &http.Response{StatusCode: http.StatusOK},
			wantStatus: plugin.OK,
		},
		{
			name: "Unexpected 200 OK",
			fields: fields{
				responseCode: http.StatusMovedPermanently,
			},
			resp:       &http.Response{StatusCode: http.StatusOK},
			wantStatus: plugin.Critical,
		},
		{
			name: "Expected 301 Moved Permanently",
			fields: fields{
				responseCode: http.StatusMovedPermanently,
			},
			resp:       &http.Response{StatusCode: http.StatusMovedPermanently},
			wantStatus: plugin.OK,
		},
		{
			name: "Unexpected 301 Moved Permanently",
			fields: fields{
				responseCode: http.StatusBadRequest,
			},
			resp:       &http.Response{StatusCode: http.StatusMovedPermanently},
			wantStatus: plugin.Critical,
		},
		{
			name: "Redirection allowed",
			fields: fields{
				redirectOK: true,
			},
			resp:       &http.Response{StatusCode: http.StatusMovedPermanently},
			wantStatus: plugin.OK,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &CheckHTTP{
				redirectOK:   tt.fields.redirectOK,
				responseCode: tt.fields.responseCode,
			}
			exit := c.handleResponse(tt.resp)
			verifyExitCode(t, exit, tt.wantStatus)
		})
	}
}

func TestInitiateRequest(t *testing.T) {
	okHandler := func(w http.ResponseWriter, r *http.Request) {
		return
	}

	panicHandler := func(w http.ResponseWriter, r *http.Request) {
		panic("panic handler!")
	}

	timeoutHandler := func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
	}

	type fields struct {
		timeout int
	}
	tests := []struct {
		name    string
		fields  fields
		handler http.HandlerFunc
		wantErr bool
	}{
		{
			name:    "Successful request",
			handler: okHandler,
			wantErr: false,
		},
		{
			name: "Timeout",
			fields: fields{
				timeout: 1,
			},
			handler: timeoutHandler,
			wantErr: true,
		},
		{
			name:    "Request error",
			handler: panicHandler,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(tt.handler))
			defer ts.Close()

			c := &CheckHTTP{
				timeout: tt.fields.timeout,
				url:     ts.URL,
			}
			defer func() {
				if err := recover(); err != nil {
					//fmt.Println(err)
				}
			}()
			client := c.prepareClient()

			_, err := c.initiateRequest(client)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckHTTP.initiateRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestVerifyBody(t *testing.T) {
	type fields struct {
		missingPattern string
		pattern        string
	}
	tests := []struct {
		name       string
		fields     fields
		resp       *http.Response
		wantStatus int
	}{
		{
			name: "Required pattern is present",
			fields: fields{
				pattern: "foo",
			},
			resp: &http.Response{
				Body:       ioutil.NopCloser(bytes.NewReader([]byte("foobar"))),
				StatusCode: http.StatusOK,
			},
			wantStatus: plugin.OK,
		},
		{
			name: "Required pattern is missing",
			fields: fields{
				pattern: "qux",
			},
			resp: &http.Response{
				Body:       ioutil.NopCloser(bytes.NewReader([]byte("foobar"))),
				StatusCode: http.StatusOK,
			},
			wantStatus: plugin.Critical,
		},
		{
			name: "Disallowed pattern is present",
			fields: fields{
				missingPattern: "foo",
			},
			resp: &http.Response{
				Body:       ioutil.NopCloser(bytes.NewReader([]byte("foobar"))),
				StatusCode: http.StatusOK,
			},
			wantStatus: plugin.Critical,
		},
		{
			name: "Disallowed pattern is missing",
			fields: fields{
				missingPattern: "qux",
			},
			resp: &http.Response{
				Body:       ioutil.NopCloser(bytes.NewReader([]byte("foobar"))),
				StatusCode: http.StatusOK,
			},
			wantStatus: plugin.OK,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &CheckHTTP{
				missingPattern: tt.fields.missingPattern,
				pattern:        tt.fields.pattern,
			}
			exit := c.verifyBody(tt.resp)
			verifyExitCode(t, exit, tt.wantStatus)
		})
	}
}
