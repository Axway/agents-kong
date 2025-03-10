package common

import "context"

const (
	AttrServiceID   = "serviceID"
	AttrServiceName = "serviceName"
	AttrRouteName   = "routeName"
	AttrRouteID     = "routeID"
	AttrServiceTag  = "serviceTag"
	AttrChecksum    = "checksum"
	AttrAppID       = "kongApplicationId"

	AttrCredentialID = "kongCredentialID"
	AttrCredUpdater  = "kongCredentialUpdate"

	AclGroup    = "amplify.group"
	Marketplace = "marketplace"
	// CorsField -
	CorsField = "cors"

	// RedirectURLsField -
	RedirectURLsField = "redirectURLs"
	OauthServerField  = "oauthServer"

	OAuth2AuthType = "oauth2"

	ApplicationTypeField = "applicationType"
	// ClientTypeField -
	ClientTypeField = "clientType"
	AudienceField   = "audience"
	OauthScopes     = "oauthScopes"

	// plugins
	AclPlugin          = "acl"
	RateLimitingPlugin = "rate-limiting"
)

type ContextKeys string

func (c ContextKeys) String() string {
	return string(c)
}

const (
	ContextWorkspace ContextKeys = "workspace"
)

func GetStringValueFromCtx(ctx context.Context, key ContextKeys) string {
	ctxVal := ctx.Value(key)
	str, ok := ctxVal.(string)
	if !ok {
		return ""
	}
	return str
}
