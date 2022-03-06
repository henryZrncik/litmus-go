package cluster

import (
	"github.com/litmuschaos/litmus-go/pkg/clients"
	"github.com/litmuschaos/litmus-go/pkg/log"
	"github.com/litmuschaos/litmus-go/pkg/status"
	experimentTypes "github.com/litmuschaos/litmus-go/pkg/strimzi/types"
)

// ClusterHealthCheck checks health of the kafka cluster
func ClusterHealthCheck(experimentsDetails *experimentTypes.ExperimentDetails, clients clients.ClientSets) error {
	// Checking Kafka pods status
	log.Info("[Status]: Verify that all the kafka pods are running")
	if err := status.CheckApplicationStatus(experimentsDetails.Kafka.Namespace, experimentsDetails.Kafka.Label, experimentsDetails.Control.Timeout, experimentsDetails.Control.Delay, clients); err != nil {
		return err
	}
	return nil
}
