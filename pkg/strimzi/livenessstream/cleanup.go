package livenessstream

import (
	"fmt"
	"github.com/litmuschaos/litmus-go/pkg/clients"
	"github.com/litmuschaos/litmus-go/pkg/log"
	experimentTypes "github.com/litmuschaos/litmus-go/pkg/strimzi/types"
	strimziLivenessUtils "github.com/litmuschaos/litmus-go/pkg/strimzi/utils/liveness"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	topicCleanJobNamePrefix          = "strimzi-liveness-topic-clean-"
)

// JobsCleanup deletes all additional pods and jobs created by experiment.
func JobsCleanup(experimentsDetails *experimentTypes.ExperimentDetails, clients clients.ClientSets)  error {
	log.Info("[Liveness-Cleanup]: Deleting the kafka liveness jobs (post-chaos cleanup)")
	// delete pods for sure
	clients.KubeClient.CoreV1().Pods(experimentsDetails.App.Namespace).DeleteCollection(
		&metav1.DeleteOptions{},
		metav1.ListOptions{LabelSelector: fmt.Sprintf("app=kafka-liveness,run=%s", experimentsDetails.Control.RunID)},
	)

	// delete jobs (which include pods as well)
	err := clients.KubeClient.BatchV1().Jobs(experimentsDetails.App.Namespace).DeleteCollection(
		&metav1.DeleteOptions{},
		metav1.ListOptions{LabelSelector: fmt.Sprintf("app=kafka-liveness,run=%s", experimentsDetails.Control.RunID)},
	)
	if err != nil {
		log.Warnf("[Liveness-Cleanup]: Problem while deleting jobs: %s", err)
		return err
	}

	return nil

}

// TopicCleanup creates the kafka liveness pod
func TopicCleanup(exp *experimentTypes.ExperimentDetails) error {
	log.InfoWithValues("[Liveness-Cleanup]: (post-chaos cleanup) Deleting the kafka liveness topic:", logrus.Fields{
		"Topic Name": exp.Topic.Name,
	})
	return strimziLivenessUtils.DeleteTopic(exp.Topic.Name,exp.App.Namespace,*exp.Strimzi.Client)

	//command := fmt.Sprintf("kafka-topics --bootstrap-server %s:%s --delete --topic %s",
	//	experimentsDetails.Kafka.Service,
	//	experimentsDetails.Kafka.Port,
	//	experimentsDetails.Topic.Name,
	//)
	//jobName := topicCleanJobNamePrefix + experimentsDetails.Control.RunID
	//cmdImage := experimentsDetails.Consumer.ConsumerImage
	//jobNamespace := experimentsDetails.App.Namespace

	// delete topic

	//if err := strimziLivenessUtils.CreateJob(jobName, experimentsDetails.Control.RunID, cmdImage, jobNamespace, command, nil, clients); err != nil {
	//	return err
	//}
	//
	//// wait for topic to be created/ fail
	//log.Info("[Wait]: Waiting for Deletion of the test topic")
	//if err:= strimziLivenessUtils.WaitForExecPod(jobName, jobNamespace, 30, experimentsDetails.Control.Delay, clients, "deleting the liveness topic"); err != nil {
	//	log.Warnf("[Wait]: Error While Waiting for Deletion of the test topic: %s. Please consider delete.topic.enable option in your cluster ",err)
	//	return err
	//}
	//
	//return nil
}