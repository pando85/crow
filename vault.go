package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/vault/api"
	"golang.org/x/net/http2"
)

type SecretMsgStorer interface {
	Store(string, ttl string) (token string, err error)
	Get(token string) (msg string, err error)
}

type vault struct {
	address string
	prefix  string
	token   string
}

// NewVault creates a vault client to talk with underline vault server
func newVault(address string, prefix string, token string) vault {
	return vault{address, prefix, token}
}

func (v vault) Store(msg string, ttl string) (token string, err error) {
	// Default TTL
	if ttl == "" {
		ttl = "48h"
	}

	// Verify duration
	d, err := time.ParseDuration(ttl)
	if err != nil {
		return "", fmt.Errorf("cannot parse duration %v", err)
	}

	// validate duration length
	if d > 168*time.Hour || d == 0*time.Hour {
		return "", fmt.Errorf("cannot set ttl to infinte or more than 7 days %v", err)
	}

	t, err := v.createOneTimeToken(ttl)
	if err != nil {
		return "", err
	}

	if v.writeMsgToVault(t, msg) != nil {
		return "", err
	}
	return t, nil
}

func (v vault) createOneTimeToken(ttl string) (string, error) {
	fmt.Println("Info: creating message with ttl: ", ttl)

	c, err := v.newVaultClient()
	if err != nil {
		return "", err
	}
	t := c.Auth().Token()

	var notRenewable bool
	s, err := t.Create(&api.TokenCreateRequest{
		Metadata:       map[string]string{"name": "placeholder"},
		ExplicitMaxTTL: ttl,
		NumUses:        2, //1 to create 2 to get
		Renewable:      &notRenewable,
	})
	if err != nil {
		return "", err
	}

	return s.Auth.ClientToken, nil
}

func (v vault) newVaultClient() (*api.Client, error) {
	config := &api.Config{
		Address:      "https://127.0.0.1:8200",
		HttpClient:   cleanhttp.DefaultPooledClient(),
		Timeout:      time.Second * 60,
		MinRetryWait: time.Millisecond * 1000,
		MaxRetryWait: time.Millisecond * 1500,
		MaxRetries:   2,
		Backoff:      retryablehttp.LinearJitterBackoff,
	}

	transport := config.HttpClient.Transport.(*http.Transport)
	transport.TLSHandshakeTimeout = 10 * time.Second
	insecureSkipVerify := false
	confInsecureSkipVerify := os.Getenv("VAULT_TLS_INSECURE_SKIP_VERIFY")
	if len(confInsecureSkipVerify) != 0 {
		insecureSkipVerifyBool, err := strconv.ParseBool(confInsecureSkipVerify)
		insecureSkipVerify = insecureSkipVerifyBool
		if err != nil {
			log.Fatal(err)
			return nil, err
		}
	}

	transport.TLSClientConfig = &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: insecureSkipVerify,
	}
	if err := http2.ConfigureTransport(transport); err != nil {
		config.Error = err
		return nil, err
	}

	if err := config.ReadEnvironment(); err != nil {
		config.Error = err
		return nil, err
	}

	// Ensure redirects are not automatically followed
	// Note that this is sane for the API client as it has its own
	// redirect handling logic (and thus also for command/meta),
	// but in e.g. http_test actual redirect handling is necessary
	config.HttpClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		// Returning this value causes the Go net library to not close the
		// response body and to nil out the error. Otherwise retry clients may
		// try three times on every redirect because it sees an error from this
		// function (to prevent redirects) passing through to it.
		return http.ErrUseLastResponse
	}

	c, err := api.NewClient(config)
	if err != nil {
		return nil, err
	}

	if v.token != "" {
		c.SetToken(v.token)
	}

	if v.address == "" {
		return c, nil
	}

	err = c.SetAddress(v.address)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (v vault) writeMsgToVault(token, msg string) error {
	c, err := v.newVaultClientWithToken(token)
	if err != nil {
		return err
	}

	raw := map[string]interface{}{"msg": msg}

	_, err = c.Logical().Write("/" + v.prefix + token, raw)

	return err
}

func (v vault) Get(token string) (msg string, err error) {
	c, err := v.newVaultClientWithToken(token)
	if err != nil {
		return "", err
	}

	r, err := c.Logical().Read(v.prefix + token)
	if err != nil {
		return "", err
	}
	return r.Data["msg"].(string), nil
}

func (v vault) newVaultClientWithToken(token string) (*api.Client, error) {
	c, err := v.newVaultClient()
	if err != nil {
		return nil, err
	}
	c.SetToken(token)
	return c, nil
}
