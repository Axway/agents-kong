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

	"github.com/Axway/agents-kong/pkg/traceability/config"
	"github.com/Axway/agents-kong/pkg/traceability/processor"
)

type httpLogBeater struct {
	client beat.Client
	logger log.FieldLogger
	server http.Server
}

// New creates an instance of kong_traceability_agent.
func New(*beat.Beat, *common.Config) (beat.Beater, error) {
	b := &httpLogBeater{
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

	mux := http.NewServeMux()
	mux.HandleFunc(config.GetAgentConfig().HttpLogPluginConfig.Path, b.HandleHello)

	// other handlers can be assigned to separate paths
	b.server = http.Server{Handler: mux, Addr: fmt.Sprintf(":%d", config.GetAgentConfig().HttpLogPluginConfig.Port)}
	b.logger.Fatal(b.server.ListenAndServe())

	return nil
}

func (b *httpLogBeater) HandleHello(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		b.logger.Trace("received a non post request")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	w.WriteHeader(200)

	logData, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		b.logger.WithError(err).Error("reading request body")
		return
	}

	go func(data []byte) {
		ctx := context.WithValue(context.Background(), processor.CtxTransactionID, uuid.NewString())

		eventProcessor, err := processor.NewEventsHandler(ctx, data)
		if err == nil {
			eventsToPublish, err := eventProcessor.Handle()
			if err == nil {
				b.client.PublishAll(eventsToPublish)
			}
		}
	}(logData)
}

// Stop stops kong_traceability_agent.
func (b *httpLogBeater) Stop() {
	b.server.Shutdown(context.Background())
}