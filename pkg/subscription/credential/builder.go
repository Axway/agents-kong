package credential

import (
	"github.com/Axway/agents-kong/pkg/common"
	"github.com/google/uuid"
	klib "github.com/kong/go-kong/kong"
)

type kongCredentialBuilder struct {
	consumer     *klib.Consumer
	id           *string
	name         *string
	createdAt    *int
	authKey      *string
	consumerTags []*string
	username     *string
	password     *string
	clientID     *string
	clientSecret *string
	clientType   *string
	redirectURIs []*string
	credMetadata map[string]interface{}
}

type credentialMetaData struct {
	cors            []string
	redirectURLs    []string
	oauthServerName string
	appType         string
	audience        string
}

func NewKongCredentialBuilder() *kongCredentialBuilder {
	// now := int(time.Now().Unix())
	b := &kongCredentialBuilder{
		// createdAt: &now,
	}
	return b
}

func (b *kongCredentialBuilder) WithConsumer(c *klib.Consumer) *kongCredentialBuilder {
	b.consumer = c
	return b
}

func (b *kongCredentialBuilder) WithID(id string) *kongCredentialBuilder {
	b.id = &id
	return b
}

// WithName adds a random UUID if passed an empty string
func (b *kongCredentialBuilder) WithName(name string) *kongCredentialBuilder {
	if name == "" {
		randomName := uuid.New().String()
		b.name = &randomName
		return b
	}
	b.name = &name
	return b
}

// WithAuthKey adds a random UUID if passed an empty string
func (b *kongCredentialBuilder) WithAuthKey(authKey string) *kongCredentialBuilder {
	if authKey == "" {
		randomAuthKey := uuid.New().String()
		b.authKey = &randomAuthKey
		return b
	}
	b.authKey = &authKey
	return b
}

func (b *kongCredentialBuilder) WithConsumerTags(consumerTags []*string) *kongCredentialBuilder {
	b.consumerTags = consumerTags
	return b
}

// WithUsername adds a random UUID if passed an empty string
func (b *kongCredentialBuilder) WithUsername(username string) *kongCredentialBuilder {
	if username == "" {
		randomUsername := uuid.New().String()
		b.username = &randomUsername
		return b
	}
	b.username = &username
	return b
}

// WithPassword adds a random UUID if passed an empty string
func (b *kongCredentialBuilder) WithPassword(password string) *kongCredentialBuilder {
	if password == "" {
		randomPass := uuid.New().String()
		b.password = &randomPass
		return b
	}
	b.password = &password
	return b
}

// WithClientID adds a random UUID if passed an empty string
func (b *kongCredentialBuilder) WithClientID(clientID string) *kongCredentialBuilder {
	if clientID == "" {
		randomID := uuid.New().String()
		b.clientID = &randomID
		return b
	}
	b.clientID = &clientID
	return b
}

// WithClientSecret adds a random UUID if passed an empty string
func (b *kongCredentialBuilder) WithClientSecret(clientSecret string) *kongCredentialBuilder {
	if clientSecret == "" {
		randomID := uuid.New().String()
		b.clientSecret = &randomID
		return b
	}
	b.clientSecret = &clientSecret
	return b
}

func (b *kongCredentialBuilder) WithProvData(provData map[string]interface{}) *kongCredentialBuilder {
	pData := getCredProvData(provData)
	if len(pData.cors) > 0 {
		var redirectUris []*string
		for i := range pData.cors {
			redirectUris = append(redirectUris, &pData.cors[i])
		}
		b.redirectURIs = redirectUris
	}
	b.clientType = &pData.appType

	return b
}

func (b *kongCredentialBuilder) ToOauth2() *klib.Oauth2Credential {
	return &klib.Oauth2Credential{
		Consumer:     b.consumer,
		Name:         b.name,
		ID:           b.id,
		CreatedAt:    b.createdAt,
		Tags:         b.consumerTags,
		ClientID:     b.clientID,
		ClientSecret: b.clientSecret,
		RedirectURIs: b.redirectURIs,
		ClientType:   b.clientType,
	}
}

func (b *kongCredentialBuilder) ToBasicAuth() *klib.BasicAuth {
	return &klib.BasicAuth{
		Consumer:  b.consumer,
		ID:        b.id,
		CreatedAt: b.createdAt,
		Tags:      b.consumerTags,
		Username:  b.username,
		Password:  b.password,
	}
}

func (b *kongCredentialBuilder) ToKeyAuth() *klib.KeyAuth {
	return &klib.KeyAuth{
		Consumer:  b.consumer,
		ID:        b.id,
		CreatedAt: b.createdAt,
		Key:       b.authKey,
		Tags:      b.consumerTags,
	}
}

func getCredProvData(credData map[string]interface{}) credentialMetaData {
	// defaults
	credMetaData := credentialMetaData{
		cors:         []string{"*"},
		redirectURLs: []string{},
		appType:      "Confidential",
		audience:     "",
	}

	// get cors from credential request
	if data, ok := credData[common.CorsField]; ok && data != nil {
		credMetaData.cors = []string{}
		for _, c := range data.([]interface{}) {
			credMetaData.cors = append(credMetaData.cors, c.(string))
		}
	}
	// get redirectURLs
	if data, ok := credData[common.RedirectURLsField]; ok && data != nil {
		credMetaData.redirectURLs = []string{}
		for _, u := range data.([]interface{}) {
			credMetaData.redirectURLs = append(credMetaData.redirectURLs, u.(string))
		}
	}
	// Oauth Server field
	if data, ok := credData[common.OauthServerField]; ok && data != nil {
		credMetaData.oauthServerName = data.(string)
	}
	// credential type field
	if data, ok := credData[common.ApplicationTypeField]; ok && data != nil {
		credMetaData.appType = data.(string)
	}

	return credMetaData
}
