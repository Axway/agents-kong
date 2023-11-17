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
	done           chan struct{}
	eventProcessor *processor.EventProcessor
	client         beat.Client
	eventChannel   chan string
	logger         log.FieldLogger
}

// New creates an instance of kong_traceability_agent.
func New(*beat.Beat, *common.Config) (beat.Beater, error) {
	b := &httpLogBeater{
		done:         make(chan struct{}),
		eventChannel: make(chan string),
		logger:       log.NewFieldLogger().WithComponent("httpLogBeater").WithPackage("beater"),
	}

	b.eventProcessor = processor.NewEventProcessor()

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

			ctx := context.WithValue(context.Background(), processor.CtxTransactionID, uuid.NewString())
			logData, err := io.ReadAll(r.Body)
			defer r.Body.Close()

			if err != nil {
				b.logger.Error(fmt.Errorf("error while reading request body: %s", err))
			}

			w.WriteHeader(200)
			go b.processAndDispatchEvent(ctx, logData)
		})

	/* Start a new HTTP server in a separate Go routine that will be the target
	   for the HTTP Log plugin. It should write events it gets to eventChannel */
	go func() {
		if err := http.ListenAndServe(fmt.Sprintf(":%d", config.GetAgentConfig().HttpLogPluginConfig.Port),
			nil); err != nil {
			b.logger.Fatalf("Unable to start the HTTP Server: %s", err)
		}
		b.logger.Infof("Started HTTP server on port %d to receive request logs", config.GetAgentConfig().HttpLogPluginConfig.Port)
	}()

	<-b.done
	return nil
}

// Stop stops kong_traceability_agent.
func (bt *httpLogBeater) Stop() {
	bt.client.Close()
	close(bt.done)
}

func (bt *httpLogBeater) processAndDispatchEvent(ctx context.Context, logData []byte) {
	log := log.UpdateLoggerWithContext(ctx, bt.logger)
	log.WithField("data", logData).Trace("handling log data")
	eventsToPublish := bt.eventProcessor.ProcessRaw(ctx, logData)
	if eventsToPublish != nil {
		log.Trace("finished handling data")
		bt.client.PublishAll(eventsToPublish)
	}
}
