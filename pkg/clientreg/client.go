package clientreg

import (
	"fmt"
	"net/http"

	"github.com/Axway/agent-sdk/pkg/apic/auth"
	"github.com/Axway/agents-kong/pkg/clientreg/client"
	"github.com/Axway/agents-kong/pkg/clientreg/client/applications"
	"github.com/Axway/agents-kong/pkg/clientreg/models"
	"github.com/go-openapi/runtime"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"

	"github.com/mitchellh/mapstructure"
)

// Client perfoms operations agains a micro gateway controller service.
type Client struct {
	cr *client.Clientreg
}

type runtimeHeaderSetter struct {
	runtime.ClientRequest
}

/* #nosec G104 */
func (rhs runtimeHeaderSetter) SetHeader(key, value string) {
	rhs.SetHeaderParam(key, value) // error can be safely ignored because header will not be set and not be detrimental
}

func apicAuthClientInfoWriter(aa *auth.ApicAuth) runtime.ClientAuthInfoWriter {
	return runtime.ClientAuthInfoWriterFunc(func(req runtime.ClientRequest, reg strfmt.Registry) error {
		return aa.Authenticate(runtimeHeaderSetter{req})
	})
}

// NewClient creates a new microgateway controller.
func NewClient(host string, basePath string, scheme string, httpClient *http.Client, aa *auth.ApicAuth) *Client {
	rt := httptransport.NewWithClient(host, basePath, []string{scheme}, httpClient)
	if aa != nil {
		rt.DefaultAuthentication = apicAuthClientInfoWriter(aa)
	}
	cr := client.New(rt, nil)

	return &Client{
		cr: cr,
	}
}

func (c *Client) GetAppProfile(appID string, profileName string) (*models.JWTKeyProfile, error) {
	q := "name==" + profileName
	r, err := c.cr.Applications.GetProfilesForApplication(
		applications.NewGetProfilesForApplicationParams().
			WithApplicationID(appID).
			WithQuery(&q), nil)

	if err != nil {
		return nil, fmt.Errorf("failed to get application profile: %w", err)
	}

	// if r.Error() != "" {
	// 	return nil, fmt.Errorf("failed to get application profile: %s", r.Error())
	// }

	if len(r.GetPayload()) == 0 {
		return nil, nil
	}

	profile := &models.JWTKeyProfile{}

	if err := mapstructure.Decode(r.GetPayload()[0], profile); err != nil {
		return nil, fmt.Errorf("failed to get application profile: %w", err)
	}

	return profile, nil
}
