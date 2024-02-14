package kong

import (
	"encoding/json"
)

type OAuthPluginConfig struct {
	HideCredentials               bool     `json:"hide_credentials,omitempty"`
	PersistentRefreshToken        bool     `json:"persistent_refresh_token,omitempty"`
	ProvisionKey                  string   `json:"provision_key,omitempty"`
	RefreshTokenTTL               int64    `json:"refresh_token_ttl,omitempty"`
	TokenExpiration               int64    `json:"token_expiration,omitempty"`
	AcceptHTTPIfAlreadyTerminated bool     `json:"accept_http_if_already_terminated,omitempty"`
	AuthHeaderName                string   `json:"auth_header_name,omitempty"`
	MandatoryScope                bool     `json:"mandatory_scope,omitempty"`
	Scopes                        []string `json:"scopes,omitempty"`
	PKCE                          string   `json:"pkce,omitempty"`
	ReuseRefreshToken             bool     `json:"reuse_refresh_token,omitempty"`
	EnablePasswordGrant           bool     `json:"enable_password_grant,omitempty"`
	EnableClientCredentials       bool     `json:"enable_client_credentials,omitempty"`
	GlobalCredentials             bool     `json:"global_credentials,omitempty"`
	Anonymous                     string   `json:"anonymous,omitempty"`
	EnableImplicitGrant           bool     `json:"enable_implicit_grant,omitempty"`
	EnableAuthorizationCode       bool     `json:"enable_authorization_code,omitempty"`
}

func NewOAuthPluginConfigFromMap(mapData map[string]interface{}) (*OAuthPluginConfig, error) {
	// Convert map to json string
	jsonStr, err := json.Marshal(mapData)
	if err != nil {
		return nil, err
	}

	config := &OAuthPluginConfig{}
	if err := json.Unmarshal(jsonStr, config); err != nil {
		return nil, err
	}

	return config, nil
}

type KeyAuthPluginConfig struct {
	KeyInQuery      bool     `json:"key_in_query,omitempty"`
	KeyInHeader     bool     `json:"key_in_header,omitempty"`
	KeyNames        []string `json:"key_names,omitempty"`
	Anonymous       string   `json:"anonymous,omitempty"`
	RunOnPreflight  bool     `json:"run_on_preflight,omitempty"`
	HideCredentials bool     `json:"hide_credentials,omitempty"`
	KeyInBody       bool     `json:"key_in_body,omitempty"`
}

func NewKeyAuthPluginConfigFromMap(mapData map[string]interface{}) (*KeyAuthPluginConfig, error) {
	// Convert map to json string
	jsonStr, err := json.Marshal(mapData)
	if err != nil {
		return nil, err
	}

	config := &KeyAuthPluginConfig{}
	if err := json.Unmarshal(jsonStr, config); err != nil {
		return nil, err
	}

	return config, nil
}

type BasicAuthPluginConfig struct {
	Anonymous       string `json:"anonymous,omitempty"`
	HideCredentials bool   `json:"hide_credentials,omitempty"`
}

func NewBasicAuthPluginConfigFromMap(mapData map[string]interface{}) (*BasicAuthPluginConfig, error) {
	// Convert map to json string
	jsonStr, err := json.Marshal(mapData)
	if err != nil {
		return nil, err
	}

	config := &BasicAuthPluginConfig{}
	if err := json.Unmarshal(jsonStr, config); err != nil {
		return nil, err
	}

	return config, nil
}
