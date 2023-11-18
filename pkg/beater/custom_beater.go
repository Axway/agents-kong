package beater

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/google/uuid"

	agentErrors "github.com/Axway/agent-sdk/pkg/util/errors"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
	"github.com/Axway/agent-sdk/pkg/util/log"

	config "github.com/Axway/agents-kong/pkg/config/traceability"
	"github.com/Axway/agents-kong/pkg/processor"
)

type httpLogBeater struct {
	done   chan struct{}
	client beat.Client
	logger log.FieldLogger
}

// New creates an instance of kong_traceability_agent.
func New(*beat.Beat, *common.Config) (beat.Beater, error) {
	b := &httpLogBeater{
		done:   make(chan struct{}),
		logger: log.NewFieldLogger().WithComponent("httpLogBeater").WithPackage("beater"),
	}

	// Validate that all necessary services are up and running. If not, return error
	if hc.RunChecks() != hc.OK {
		b.logger.Error("not all services are running")
		return nil, agentErrors.ErrInitServicesNotReady
	}

	return b, nil
}

// Run starts kong_traceability_agent.
func (b *httpLogBeater) Run(beater *beat.Beat) error {
	b.logger.Info("kong_traceability_agent is running! Hit CTRL-C to stop it.")

	var err error
	b.client, err = beater.Publisher.Connect()
	if err != nil {
		return err
	}

	http.HandleFunc(config.GetAgentConfig().HttpLogPluginConfig.Path,
		func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				b.logger.Trace("received a non post request")
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}

			// http://10.129.216.201.nip.io:9000/requestlogs

			ctx := context.WithValue(context.Background(), processor.CtxTransactionID, uuid.NewString())
			logData, err := io.ReadAll(r.Body)
			defer r.Body.Close()

			if err != nil {
				b.logger.WithError(err).Error("reading request body")
			}

			w.WriteHeader(200)

			eventProcessor, err := processor.NewEventProcessor(ctx, logData)
			if err == nil {
				go b.process(eventProcessor)
			}
		},
	)

	/* Start a new HTTP server in a separate Go routine that will be the target
	   for the HTTP Log plugin. It should write events it gets to eventChannel */
	go func() {
		err := http.ListenAndServe(fmt.Sprintf(":%d", config.GetAgentConfig().HttpLogPluginConfig.Port), nil)
		if err != nil {
			b.logger.WithError(err).Fatalf("unable to start the HTTP Server")
		}
		b.logger.WithField("port", config.GetAgentConfig().HttpLogPluginConfig.Port).WithField("path", config.GetAgentConfig().HttpLogPluginConfig.Path).Info("started HTTP server")
	}()

	<-b.done
	return nil
}

// Stop stops kong_traceability_agent.
func (b *httpLogBeater) Stop() {
	b.client.Close()
	close(b.done)
}

func (b *httpLogBeater) process(eventProcessor *processor.EventProcessor) {
	eventsToPublish, err := eventProcessor.Process()
	if err == nil {
		b.client.PublishAll(eventsToPublish)
	}
}
