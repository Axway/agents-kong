package beater

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/google/uuid"

	"github.com/Axway/agent-sdk/pkg/agent"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/transaction/metric"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
	"github.com/Axway/agent-sdk/pkg/util/log"

	"github.com/Axway/agents-kong/pkg/traceability/config"
	"github.com/Axway/agents-kong/pkg/traceability/processor"
)

type httpLogBeater struct {
	client       beat.Client
	logger       log.FieldLogger
	server       http.Server
	processing   sync.WaitGroup
	shutdownDone sync.WaitGroup
}

// New creates an instance of kong_traceability_agent.
func New(*beat.Beat, *common.Config) (beat.Beater, error) {
	b := &httpLogBeater{
		logger:       log.NewFieldLogger().WithComponent("httpLogBeater").WithPackage("beater"),
		processing:   sync.WaitGroup{},
		shutdownDone: sync.WaitGroup{},
	}

	for {
		if hc.RunChecks() == hc.OK {
			break
		}
		// Validate that all necessary services are up and running. If not, try in 5 seconds
		b.logger.Error("waiting for all services to be running")
		time.Sleep(5 * time.Second)
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
	agent.RegisterShutdownHandler(b.shutdownHandler)

	mux := http.NewServeMux()
	mux.HandleFunc(config.GetAgentConfig().KongGatewayCfg.Logs.HTTP.Path, b.HandleEvent)

	// other handlers can be assigned to separate paths
	b.server = http.Server{Handler: mux, Addr: fmt.Sprintf(":%d", config.GetAgentConfig().KongGatewayCfg.Logs.HTTP.Port)}
	b.server.ListenAndServe()

	// wait for the shutdown process to finish prior to exit
	b.shutdownDone.Add(1)
	b.shutdownDone.Wait()

	return nil
}

func (b *httpLogBeater) HandleEvent(w http.ResponseWriter, r *http.Request) {
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

	b.processing.Add(1)
	go func(data []byte) {
		defer b.processing.Done()
		ctx := context.WithValue(context.Background(), processor.CtxTransactionID, uuid.NewString())

		eventProcessor, err := processor.NewEventsHandler(ctx, data)
		if err == nil {
			eventsToPublish := eventProcessor.Handle()
			b.client.PublishAll(eventsToPublish)
		}
	}(logData)
}

func (b *httpLogBeater) shutdownHandler() {
	b.logger.Info("waiting for current processing to finish")
	defer b.shutdownDone.Done()

	// wait for all processing to finish
	b.processing.Wait()

	// publish the metrics and usage
	b.logger.Info("publishing cached metrics and usage")
	metric.GetMetricCollector().ShutdownPublish()

	// clean the agent resource, if necessary
	b.cleanResource()
}

func (b *httpLogBeater) cleanResource() {
	// if pod name is empty do nothing further
	podName := os.Getenv("POD_NAME")
	if podName == "" {
		b.logger.Debug("not cleaning the agent resource, does not seem to be a kubernetes pod")
		return
	}

	// check if this agent resource reported an error
	if agent.GetStatus() == agent.AgentFailed || agent.GetStatus() == agent.AgentUnhealthy {
		b.logger.Debug("not cleaning the agent resource, agent not gracefully stopping")
		return
	}

	// check that this is not the last agent resource to be removed
	agentRes := management.NewTraceabilityAgent(config.GetAgentConfig().CentralCfg.GetAgentName(), config.GetAgentConfig().CentralCfg.GetEnvironmentName())
	res, err := agent.GetCentralClient().GetResources(agentRes)
	if len(res) > 1 && err == nil {
		b.logger.Info("cleaning the agent resource")
		// cleanup the agent resource
		agent.GetCentralClient().DeleteResourceInstance(agentRes)
	}
}

// Stop stops kong_traceability_agent.
func (b *httpLogBeater) Stop() {
	b.server.Shutdown(context.Background())
}
