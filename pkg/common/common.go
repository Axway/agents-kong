package common

const (
	AttrServiceId = "serviceId"
	AttrRouteId   = "routeId"
	AttrChecksum  = "checksum"
	AttrAppID     = "kongApplicationId"

	AttrCredentialID = "kongCredentialId"

	AclGroup    = "amplify.group"
	Marketplace = "marketplace"
	// CorsField -
	CorsField    = "cors"
	ProvisionKey = "provision_key"

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
