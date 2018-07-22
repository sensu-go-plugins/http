package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/sensu-go-plugins/gunsen/plugin"
)

// CheckHTTP represents
type CheckHTTP struct {
	cmd plugin.Command

	missingPattern string
	pattern        string
	redirectOK     bool
	responseCode   int
	timeout        int
	url            string
}

func main() {
	// Initialize our check
	c := &CheckHTTP{
		cmd: plugin.NewCommand("CheckHTTP"),
	}

	// Instantiate the configuration flags
	c.cmd.Flags().StringVarP(&c.missingPattern, "negquery", "n", "", "Query for pattern that must be absent in response body")
	c.cmd.Flags().StringVarP(&c.pattern, "query", "q", "", "Query for pattern that must exist in response body")
	c.cmd.Flags().BoolVarP(&c.redirectOK, "redirect-ok", "r", false, "Accept redirection")
	c.cmd.Flags().IntVar(&c.responseCode, "response-code", http.StatusOK, "Expected HTTP status code")
	c.cmd.Flags().IntVarP(&c.timeout, "timeout", "t", 15, "Time limit, in seconds, for the request")
	c.cmd.Flags().StringVarP(&c.url, "url", "u", "", "URL to connect to")

	// Execute the check
	plugin.Execute(c)
}

// Command returns the plugin command
func (c *CheckHTTP) Command() plugin.Command {
	return c.cmd
}

// Run executes the plugin
func (c *CheckHTTP) Run() error {
	// Validate the provided configuration
	if c.url == "" {
		return &plugin.Exit{Msg: "no URL specified", Status: plugin.Unknown}
	}

	if c.pattern != "" && c.missingPattern != "" {
		return &plugin.Exit{
			Msg:    "--query and --negquery can not be used simultaneously",
			Status: plugin.Unknown,
		}
	}

	// Perform the request
	client := c.prepareClient()
	resp, err := c.initiateRequest(client)
	if err != nil {
		return err
	}

	return c.handleResponse(resp)
}

func (c *CheckHTTP) handleResponse(resp *http.Response) error {
	responseCode := statusLine(resp.StatusCode)

	// Verify if we are expecting something else than a 200 OK status
	if c.responseCode != http.StatusOK && c.responseCode != 0 {
		if c.responseCode == resp.StatusCode {
			// The response code corresponds to the expected one, now verify the
			// response body
			return c.verifyBody(resp)
		}
		return &plugin.Exit{
			Msg:    fmt.Sprintf("expected HTTP status %s, got %s", statusLine(c.responseCode), responseCode),
			Status: plugin.Critical,
		}
	}

	if resp.StatusCode >= http.StatusOK && resp.StatusCode <= http.StatusIMUsed {
		// Verify the response body
		return c.verifyBody(resp)
	} else if resp.StatusCode >= http.StatusMultipleChoices && resp.StatusCode <= http.StatusPermanentRedirect {
		// ~300
		if c.redirectOK {
			// The redirection was expected, now verify the response body
			return c.verifyBody(resp)
		}

		// A redirection was not expected
		return &plugin.Exit{
			Msg:    responseCode + ": unexpected redirection",
			Status: plugin.Warning,
		}
	}

	return &plugin.Exit{Msg: responseCode, Status: plugin.Critical}
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

func (c *CheckHTTP) verifyBody(resp *http.Response) error {
	responseCode := statusLine(resp.StatusCode)

	// Determine if we have a pattern that must be present or absent
	pattern := c.pattern
	if c.missingPattern != "" {
		pattern = c.missingPattern
	}

	if pattern != "" {
		// Get the response body
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return &plugin.Exit{Msg: err.Error(), Status: plugin.Critical}
		}

		contentLength := len(body)

		if strings.Contains(string(body), pattern) {
			// Determine the status based on whether it must be absent or present
			status := plugin.OK
			if c.missingPattern != "" {
				status = plugin.Critical
			}

			return &plugin.Exit{
				Msg:    fmt.Sprintf("%s found /%s/ in %d bytes", responseCode, pattern, contentLength),
				Status: status,
			}
		}

		// Determine the status based on whether it must be absent or present
		status := plugin.Critical
		if c.missingPattern != "" {
			status = plugin.OK
		}

		return &plugin.Exit{
			Msg:    fmt.Sprintf("did not found /%s/ in %d bytes", pattern, contentLength),
			Status: status,
		}
	}

	return &plugin.Exit{Msg: responseCode, Status: plugin.OK}
}

// statusLine returns a string that contains the status code and status text
func statusLine(code int) string {
	return fmt.Sprintf("%d %s", code, http.StatusText(code))
}
