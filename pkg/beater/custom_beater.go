package beater

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	traceabilityconfig "github.com/Axway/agents-kong/pkg/config/traceability"

	"github.com/Axway/agents-kong/pkg/processor"

	agentErrors "github.com/Axway/agent-sdk/pkg/util/errors"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

type customLogBeater struct {
	done           chan struct{}
	eventProcessor *processor.EventProcessor
	client         beat.Client
	eventChannel   chan string
}

// New creates an instance of kong_traceability_agent.
func New(*beat.Beat, *common.Config) (beat.Beater, error) {
	bt := &customLogBeater{
		done:         make(chan struct{}),
		eventChannel: make(chan string),
	}

	bt.eventProcessor = processor.NewEventProcessor()

	// Validate that all necessary services are up and running. If not, return error
	if hc.RunChecks() != hc.OK {
		return nil, agentErrors.ErrInitServicesNotReady
	}

	return bt, nil
}

// Run starts kong_traceability_agent.
func (bt *customLogBeater) Run(b *beat.Beat) error {
	logp.Info("kong_traceability_agent is running! Hit CTRL-C to stop it.")

	var err error
	bt.client, err = b.Publisher.Connect()
	if err != nil {
		return err
	}

	http.HandleFunc(fmt.Sprintf("%s", traceabilityconfig.GetAgentConfig().HTTPLogPluginConfig.Path),
		func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}

			body, err := ioutil.ReadAll(r.Body)
			defer r.Body.Close()

			if err != nil {
				fmt.Errorf("Error while reading request body: %s", err)
			}

			w.WriteHeader(200)
			bt.processAndDispatchEvent(string(body))
		})

	/* Start a new HTTP server in a separate Go routine that will be the target
	   for the HTTP Log plugin. It should write events it gets to eventChannel */
	go func() {
		if err := http.ListenAndServe(fmt.Sprintf(":%d", traceabilityconfig.GetAgentConfig().HTTPLogPluginConfig.Port),
			nil); err != nil {
			log.Fatalf("Unable to start the HTTP Server: %s", err)
		}
		fmt.Printf("Started HTTP server on port %d to receive request logs", traceabilityconfig.GetAgentConfig().HTTPLogPluginConfig.Port)
	}()

	for {
		select {
		case <-bt.done:
			return nil
		}
	}
}

// Stop stops kong_traceability_agent.
func (bt *customLogBeater) Stop() {
	bt.client.Close()
	close(bt.done)
}

func (bt *customLogBeater) processAndDispatchEvent(logEvent string) {
	eventsToPublish := bt.eventProcessor.ProcessRaw([]byte(logEvent))
	if eventsToPublish != nil {
		bt.client.PublishAll(eventsToPublish)
	}
}
