package kong

import "net/http"

type DoRequest interface {
	Do(req *http.Request) (*http.Response, error)
}

type DocumentObjects struct {
	Data []DocumentObject `json:"data,omitempty"`
	Next string           `json:"next,omitempty"`
}

type DocumentObject struct {
	CreatedAt int    `json:"created_at,omitempty"`
	ID        string `json:"id,omitempty"`
	Path      string `json:"path,omitempty"`
	Service   struct {
		ID string `json:"id,omitempty"`
	} `json:"service,omitempty"`
}
