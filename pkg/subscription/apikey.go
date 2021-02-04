package subscription

import (
	"github.com/kong/go-kong/kong"
)

type apiKey struct{}

const name = "kong-apikey"

func (*apiKey) AppliesTo(*kong.Route) {
}
