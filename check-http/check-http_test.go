package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sensu-go-plugins/gunsen/plugin"
)

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
			if exit := c.handleResponse(tt.resp); exit != nil {
				if exit, ok := exit.(*plugin.Exit); ok {
					if exit.Status != tt.wantStatus {
						t.Errorf("CheckHTTP.handleResponse() exit status = %v, wantStatus %v", exit.Status, tt.wantStatus)
					}
				} else {
					t.Error("CheckHTTP.handleResponse() exit could not be asserted")
				}
			} else {
				t.Error("CheckHTTP.handleResponse() exit was nil")
			}
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
