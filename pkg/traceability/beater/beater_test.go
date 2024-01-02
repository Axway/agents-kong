package beater

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Axway/agent-sdk/pkg/agent"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/mock"
	corecfg "github.com/Axway/agent-sdk/pkg/config"

	"github.com/Axway/agents-kong/pkg/traceability/config"
)

func Test_httpLogBeater_cleanResource(t *testing.T) {
	testCases := map[string]struct {
		podName            string
		agentStatus        string
		numTAs             int
		getResErr          bool
		expectGetCalled    bool
		expectDeleteCalled bool
	}{
		"no pod name, nothing deleted": {
			agentStatus: agent.AgentRunning,
		},
		"agent in failed status, nothing deleted": {
			podName:     "pod",
			agentStatus: agent.AgentFailed,
		},
		"only one ta exists, nothing deleted": {
			podName:         "pod",
			agentStatus:     agent.AgentRunning,
			numTAs:          1,
			expectGetCalled: true,
		},
		"error returned getting resources, nothing deleted": {
			podName:         "pod",
			agentStatus:     agent.AgentRunning,
			numTAs:          2,
			getResErr:       true,
			expectGetCalled: true,
		},
		"all validations pass, delete agent res": {
			podName:            "pod",
			agentStatus:        agent.AgentRunning,
			numTAs:             2,
			expectGetCalled:    true,
			expectDeleteCalled: true,
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			os.Setenv("POD_NAME", tc.podName)
			config.SetAgentConfig(&config.AgentConfig{CentralCfg: &corecfg.CentralConfiguration{}})
			agent.UpdateStatus(tc.agentStatus, "")

			beater, err := New(nil, nil)
			assert.Nil(t, err)
			assert.NotNil(t, beater)
			httpBeater := beater.(*httpLogBeater)
			getResCalled := false
			deleteResCalled := false

			mockAPICClient := &mock.Client{
				GetResourcesMock: func(ri v1.Interface) ([]v1.Interface, error) {
					getResCalled = true
					assert.NotNil(t, ri)
					if tc.getResErr {
						return nil, fmt.Errorf("error")
					}
					tas := []v1.Interface{}
					for i := 0; i < tc.numTAs; i++ {
						tas = append(tas, management.NewTraceabilityAgent("name", "env"))
					}
					return tas, nil
				},
				DeleteResourceInstanceMock: func(ri v1.Interface) error {
					deleteResCalled = true
					assert.NotNil(t, ri)
					return nil
				},
			}
			agent.InitializeForTest(mockAPICClient)

			httpBeater.cleanResource()
			assert.Equal(t, tc.expectGetCalled, getResCalled)
			assert.Equal(t, tc.expectDeleteCalled, deleteResCalled)
		})
	}
}
