package main

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/palourde/gunsen/check"
	"github.com/palourde/gunsen/plugin"
)

type config struct {
	redirectOK bool
	timeout    int
	url        string
}

func main() {
	// Create a plugin
	plgn := plugin.New("check-http")

	c := &config{}

	// Instantiate the configuration flags
	plgn.Flags().BoolVarP(&c.redirectOK, "redirect-ok", "r", false, "Accept redirection")
	plgn.Flags().StringVarP(&c.url, "url", "u", "", "URL to connect to")
	plgn.Flags().IntVarP(&c.timeout, "timeout", "t", 15, "Time limit, in seconds, for the request")

	// Run the plugin
	plgn.Run(run(c))
}

func run(c *config) plugin.RunFunc {
	return func(cmd plugin.Command, args []string) error {
		client := prepareClient(c)

		resp := initiateRequest(c, client)

		handleResponse(c, resp)

		return nil
	}
}

func handleResponse(c *config, resp *http.Response) {
	status := statusLine(resp.StatusCode)

	if resp.StatusCode >= http.StatusOK && resp.StatusCode <= http.StatusIMUsed {
		// ~200
		check.OK(status)
	} else if resp.StatusCode >= http.StatusMultipleChoices && resp.StatusCode <= http.StatusPermanentRedirect {
		// ~300
		if c.redirectOK {
			check.OK(status)
		}

		// A redirection was not expected
		check.Warning(status + ": unexpected redirection")
	}

	check.Critical(status)
}

func initiateRequest(c *config, client *http.Client) *http.Response {
	resp, err := client.Get(c.url)
	if err != nil {
		// If we have an error, verify if it's a timeout
		if err, ok := err.(net.Error); ok && err.Timeout() {
			check.Critical(fmt.Sprintf("Request exceeded timeout of %d seconds", c.timeout))
		}

		// Unknown error
		check.Critical("Request error: " + err.Error())
	}

	return resp
}

func prepareClient(c *config) *http.Client {
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
