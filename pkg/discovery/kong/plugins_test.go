package kong_test

import (
	"context"
	"testing"

	klib "github.com/kong/go-kong/kong"

	"github.com/Axway/agents-kong/pkg/discovery/kong"
)

type pluginsMock []*klib.Plugin

var truePlugin = true

func (pm pluginsMock) ListAll(_ context.Context) ([]*klib.Plugin, error) {
	return pm, nil
}

func p(id, name string) *klib.Plugin {
	return &klib.Plugin{
		ID:      &id,
		Name:    &name,
		Enabled: &truePlugin,
	}
}

func pwr(id, name, routeID string) *klib.Plugin {
	return &klib.Plugin{
		ID:   &id,
		Name: &name,
		Route: &klib.Route{
			ID: &routeID,
		},
		Enabled: &truePlugin,
	}
}

func pws(id, name, serviceID string) *klib.Plugin {
	return &klib.Plugin{
		ID:   &id,
		Name: &name,
		Service: &klib.Service{
			ID: &serviceID,
		},
		Enabled: &truePlugin,
	}
}

func pwrs(id, name, routeID, serviceID string) *klib.Plugin {
	return &klib.Plugin{
		ID:   &id,
		Name: &name,
		Route: &klib.Route{
			ID: &routeID,
		},
		Service: &klib.Service{
			ID: &serviceID,
		},
		Enabled: &truePlugin,
	}
}

func TestGetEffectivePlugins(t *testing.T) {
	var routeID = "routeID"
	var serviceID = "serviceID"
	testCases := []struct {
		name        string
		plugins     []*klib.Plugin
		expectedIds map[string]interface{}
	}{{
		"one plugin with routeID",
		[]*klib.Plugin{p("1", "api-key")},
		map[string]interface{}{"1": nil},
	}, {
		"plugin on route takes precedence",
		[]*klib.Plugin{
			p("1", "api-key"),
			pwr("2", "api-key", routeID),
			pws("3", "api-key", serviceID),
		},
		map[string]interface{}{"2": nil},
	}, {
		"plugin on service takes precedence",
		[]*klib.Plugin{
			p("1", "api-key"),
			pws("3", "api-key", serviceID),
		},
		map[string]interface{}{"3": nil},
	}, {
		"multiple plugin types still correct",
		[]*klib.Plugin{
			p("1", "api-key"),
			pws("2", "api-key", serviceID),
			pwr("3", "acl", routeID),
			pws("4", "acl", serviceID),
			pwrs("5", "acl", "otherRoute", serviceID),
		},
		map[string]interface{}{"2": nil, "3": nil},
	}}

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {

			plugins := kong.Plugins{pluginsMock(tc.plugins)}

			res, err := plugins.GetEffectivePlugins(routeID, serviceID)
			if err != nil {
				t.Fatalf("Failed due: %s", err)
			}

			for _, plugin := range res {
				if _, ok := tc.expectedIds[*plugin.ID]; !ok {
					t.Error("unexpected plugin with id:", *plugin.ID)
					continue
				}

				delete(tc.expectedIds, *plugin.ID)
			}

			if len(tc.expectedIds) != 0 {
				for k := range tc.expectedIds {
					t.Error("missing plugin with id:", k)
				}
			}
		})
	}
}
