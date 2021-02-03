package gateway

import (
	"fmt"
	"strings"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agent-sdk/pkg/notify"
	log "github.com/Axway/agent-sdk/pkg/util/log"
)

// ValidateSubscription - Callback for validating the subscription to be processed
func (a *Client) ValidateSubscription(subscription apic.Subscription) bool {
	return true
}

// ProcessSubscribe - Callback for processing subscription
func (a *Client) ProcessSubscribe(subscription apic.Subscription) {
	log.Info("Processing subscription")

}

// ProcessUnsubscribe - Callback to unsubscribe
func (a *Client) ProcessUnsubscribe(subscription apic.Subscription) {
	log.Info("Processing unsubscribe")

}

func (a *Client) sendSubscriptionNotification(subscription apic.Subscription, key string, newState apic.SubscriptionState, message string) {

	cfg := agent.GetCentralConfig()
	// Verify that at least 1 notification type was set.  If none was set, then do not attempt to gather user info or send notification
	if len(cfg.GetSubscriptionConfig().GetNotificationTypes()) == 0 {
		log.Debug("No subscription notifications are configured.")
		return
	}

	apicClient := agent.GetCentralClient()
	catalogItemName, _ := apicClient.GetCatalogItemName(subscription.GetCatalogItemID())

	createdUserID := subscription.GetCreatedUserID()
	// Check to see if the id is a DOSA account.  If it is, return.  We cannot get user information from a DOSA account
	if strings.Contains(createdUserID, "DOSA_") {
		log.Errorf("Subscription id '%s' is not valid for getting platform user information", createdUserID)
		return
	}

	recipient, err := apicClient.GetUserEmailAddress(createdUserID)
	if err != nil {
		log.Errorf("Could not send notification via smtp server.  %s", err.Error())
		return
	}

	catalogItemURL := fmt.Sprintf(cfg.GetURL()+"/catalog/explore/%s", subscription.GetCatalogItemID())

	subNotification := notify.NewSubscriptionNotification(recipient, message, newState)
	subNotification.SetCatalogItemInfo(subscription.GetCatalogItemID(), catalogItemName, catalogItemURL)
	subNotification.SetAPIKeyInfo(key, "Ocp-Apim-Subscription-Key")

	// Set the authtemplate to apikey so the subscription body template fills in correctly
	subNotification.SetAuthorizationTemplate(notify.Apikeys)

	err = subNotification.NotifySubscriber(recipient)
	if err != nil {
		log.Errorf("Error hit sending notification to subscriber: %s", err.Error())
	}
}
