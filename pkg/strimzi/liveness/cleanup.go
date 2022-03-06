package liveness

import (
	"github.com/litmuschaos/litmus-go/pkg/clients"
	"github.com/litmuschaos/litmus-go/pkg/log"
	experimentTypes "github.com/litmuschaos/litmus-go/pkg/strimzi/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

) 

// Cleanup deletes all additional pods and jobs created by experiment.
func Cleanup(experimentsDetails *experimentTypes.ExperimentDetails, clients clients.ClientSets)  error {
	log.Infof("[Liveness]: Cleanup")

	err := clients.KubeClient.CoreV1().Pods(experimentsDetails.Kafka.Namespace).DeleteCollection(
		&metav1.DeleteOptions{},
		metav1.ListOptions{LabelSelector: "app=kafka-liveness"},
	)
	if err != nil {
		log.Warnf("[Liveness]: Cleanup, problem while deleting pods: %s", err)
		return err
	}

	err = clients.KubeClient.BatchV1().Jobs(experimentsDetails.Kafka.Namespace).DeleteCollection(
		&metav1.DeleteOptions{},
		metav1.ListOptions{LabelSelector: "app=kafka-liveness"},
	)
	if err != nil {
		log.Warnf("[Liveness]: Cleanup, problem while deleting jobs: %s", err)
		return err
	}

	return nil

}