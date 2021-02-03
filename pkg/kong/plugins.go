package kong

import (
	"context"

	klib "github.com/kong/go-kong/kong"
)

type Plugins struct {
	PluginLister
}

type PluginLister interface {
	ListAll(ctx context.Context) ([]*klib.Plugin, error)
}

// determines the most specific
func mostSpecific(p1, p2 *klib.Plugin) *klib.Plugin {
	if p1 == nil {
		return p2
	}

	if p2 == nil {
		return p1
	}

	if p1.Service != nil && p1.Route != nil { // can't be more specific than this
		return p1
	}
	if p2.Service != nil && p2.Route != nil { // can't be more specific than this
		return p2
	}

	if p1.Route != nil { // route is more specific
		return p1
	}

	if p2.Route != nil {
		return p2
	}

	if p1.Service != nil { //
		return p1
	}

	if p2.Service != nil { //
		return p2
	}

	return p2
}

// GetEffectivePlugins determines the effective plugin configuration for the route/service combination.
// Returns a map containg effective Plugin configuration grouped by plugin type.
func (p *Plugins) GetEffectivePlugins(routeID, serviceID string) (map[string]*klib.Plugin, error) {
	plugins, err := p.ListAll(context.Background())
	if err != nil {
		return nil, err
	}

	pmap := map[string]*klib.Plugin{}

	for _, plugin := range plugins {
		if (plugin.Route != nil && (plugin.Route.ID == nil || *plugin.Route.ID != routeID)) ||
			(plugin.Service != nil && (plugin.Service.ID == nil || *plugin.Service.ID != serviceID)) {
			continue
		}

		pmap[*plugin.Name] = mostSpecific(pmap[*plugin.Name], plugin)
	}

	return pmap, nil
}
