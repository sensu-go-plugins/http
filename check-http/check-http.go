package main

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/palourde/gunsen/plugin"
)

// CheckHTTP ...
type CheckHTTP struct {
	cmd plugin.Command

	redirectOK bool
	timeout    int
	url        string
}

func main() {
	// Initialize our check
	c := &CheckHTTP{
		cmd: plugin.NewCommand("CheckHTTP"),
	}

	// Instantiate the configuration flags
	c.cmd.Flags().BoolVarP(&c.redirectOK, "redirect-ok", "r", false, "Accept redirection")
	c.cmd.Flags().StringVarP(&c.url, "url", "u", "", "URL to connect to")
	c.cmd.Flags().IntVarP(&c.timeout, "timeout", "t", 15, "Time limit, in seconds, for the request")

	// Execute the check
	plugin.Execute(c)
}

// Command ...
func (c *CheckHTTP) Command() plugin.Command {
	return c.cmd
}

// Run ...
func (c *CheckHTTP) Run() error {
	// Validate the provided configuration
	if c.url == "" {
		return &plugin.Exit{Msg: "No URL specified", Status: plugin.Unknown}
	}

	client := c.prepareClient()
	resp, err := c.initiateRequest(client)
	if err != nil {
		return err
	}

	return c.handleResponse(resp)
}

func (c *CheckHTTP) handleResponse(resp *http.Response) error {
	status := statusLine(resp.StatusCode)

	if resp.StatusCode >= http.StatusOK && resp.StatusCode <= http.StatusIMUsed {
		// ~200
		return &plugin.Exit{Msg: status, Status: plugin.OK}
	} else if resp.StatusCode >= http.StatusMultipleChoices && resp.StatusCode <= http.StatusPermanentRedirect {
		// ~300
		if c.redirectOK {
			return &plugin.Exit{Msg: status, Status: plugin.OK}
		}

		// A redirection was not expected
		return &plugin.Exit{
			Msg:    status + ": unexpected redirection",
			Status: plugin.Warning,
		}
	}

	return &plugin.Exit{Msg: status, Status: plugin.Critical}
}

func (c *CheckHTTP) initiateRequest(client *http.Client) (*http.Response, error) {
	resp, err := client.Get(c.url)
	if err != nil {
		// If we have an error, verify if it's a timeout
		if err, ok := err.(net.Error); ok && err.Timeout() {
			return nil, &plugin.Exit{
				Msg:    fmt.Sprintf("Request exceeded timeout of %d seconds", c.timeout),
				Status: plugin.Critical,
			}
		}

		// Unknown error
		return nil, &plugin.Exit{
			Msg:    "Request error: " + err.Error(),
			Status: plugin.Critical,
		}
	}

	return resp, nil
}

func (c *CheckHTTP) prepareClient() *http.Client {
	t := time.Duration(c.timeout) * time.Second
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: t,
	}

	return client
}

// statusLine returns a string that contains the status code and status text
func statusLine(code int) string {
	return fmt.Sprintf("%d %s", code, http.StatusText(code))
}
