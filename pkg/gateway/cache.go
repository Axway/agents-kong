package gateway

import (
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/cache"
)

// If the item is cached, return true
func setCachedService(kongServiceId string, kongServiceName string, hash string, centralName string) bool {
	specCache := cache.GetCache()
	item, err := specCache.Get(kongServiceId)
	// if there is an error, then the item is not in the cache
	if err != nil {
		cachedService := CachedService{
			kongServiceId:   kongServiceId,
			kongServiceName: kongServiceName,
			hash:            hash,
			centralName:     centralName,
		}
		specCache.Set(kongServiceId, cachedService)
		return false
	}

	if item != nil {
		if cachedService, ok := item.(CachedService); ok {
			if cachedService.kongServiceId == kongServiceId && cachedService.hash == hash {
				cachedService.centralName = centralName
				cachedService.kongServiceName = kongServiceName
				specCache.Set(kongServiceId, cachedService)
				return true
			} else {
				cachedService.kongServiceName = kongServiceName
				cachedService.hash = hash
				specCache.Set(kongServiceId, cachedService)
			}
		}
	}
	return false
}

func initCache(centralAPIServices []*v1alpha1.APIService) {
	clearCache()
	for _, apiSvc := range centralAPIServices {
		setCachedService(apiSvc.Attributes[kongServiceID], apiSvc.Title, apiSvc.Attributes[kongHash], apiSvc.Name)
	}
}

func clearCache() {
	cache := cache.GetCache()
	for _, key := range cache.GetKeys() {
		cache.Delete(key)
	}
}
