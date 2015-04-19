// asynchttpclient project asynchttpclient.go
package asynchttpclient

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
)

// An AsyncHttpClient is an async http client. Its zero value (DefautAsyncHttpClient)
// will start unlimit go threads.
type Client struct {
	// Client specifies the original http.Client. If nil, http.DefaultClient is used.
	Client *http.Client

	// Max concurrency can run at the same time. If zero, concurrency is unlimited.
	Concurrency int

	ticketsJar goTickets
	ticketInit sync.Once
}

func (c *Client) client() *http.Client {
	if c.Client == nil {
		return http.DefaultClient
	}
	return c.Client
}

func (c *Client) initTicket() {
	if c.Concurrency > 0 {
		tickets, err := NewGoTicket(c.Concurrency)
		if err == nil {
			c.ticketsJar = tickets
		}
	}
}

func (c *Client) tickets() goTickets {
	if c.Concurrency > 0 && c.ticketsJar == nil {
		c.ticketInit.Do(c.initTicket)
	}
	return c.ticketsJar
}

func (c *Client) takeTicket() {
	if c.Concurrency > 0 {
		c.tickets().Take()
		return
	}
}

func (c *Client) returnTicket() {
	if c.Concurrency > 0 {
		c.tickets().Return()
		return
	}
}

func respHandler(c *Client, callback func(error, *http.Response)) {
	if p := recover(); p != nil {
		err, ok := interface{}(p).(error)
		if ok {
			callback(err, nil)
		} else {
			errMsg := fmt.Sprintf("%v", p)
			callback(errors.New(errMsg), nil)
		}
	}

	c.returnTicket()
}

func asyncAction(c *Client, callback func(error, *http.Response), action func() (*http.Response, error)) {
	c.takeTicket()

	defer respHandler(c, callback)

	resp, httpErr := action()
	callback(httpErr, resp)
}

func (c *Client) Get(url string, callback func(error, *http.Response)) {
	go asyncAction(c, callback, func() (*http.Response, error) {
		return c.client().Get(url)
	})
}

func (c *Client) Head(url string, callback func(error, *http.Response)) {
	go asyncAction(c, callback, func() (*http.Response, error) {
		return c.client().Head(url)
	})
}

func (c *Client) Post(url string, bodyType string, body io.Reader, callback func(error, *http.Response)) {
	go asyncAction(c, callback, func() (*http.Response, error) {
		return c.client().Post(url, bodyType, body)
	})
}

func (c *Client) PostForm(url string, data url.Values, callback func(error, *http.Response)) {
	go asyncAction(c, callback, func() (*http.Response, error) {
		return c.client().PostForm(url, data)
	})
}

func (c *Client) Do(req *http.Request, callback func(error, *http.Response)) {
	go asyncAction(c, callback, func() (*http.Response, error) {
		return c.client().Do(req)
	})
}
