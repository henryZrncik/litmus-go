package jobs

import (
	"github.com/litmuschaos/litmus-go/pkg/log"
	topicv1 "github.com/litmuschaos/litmus-go/pkg/strimzi/client"
	"github.com/litmuschaos/litmus-go/pkg/strimzi/client/clientset"
	experimentTypes "github.com/litmuschaos/litmus-go/pkg/strimzi/types"
	"github.com/litmuschaos/litmus-go/pkg/utils/common"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

// waitForTopicCreation check according to Topic Operator that topic is mapped from CR to kafka brokers
func waitForTopicCreation(topicName, appNs string, timeout, retryBackOffTime int, strimziClient clientset.StrimziV1AlphaClient) error {
	ChaosStartTimeStamp := time.Now()
	duration := int(time.Since(ChaosStartTimeStamp).Seconds())

	for duration < timeout {
		topicObject, err := getTopic(topicName, appNs, strimziClient)
		if err != nil {
			return err
		}

		if topicObject.Status.Conditions != nil {
			for _, condition := range topicObject.Status.Conditions {
				// if status of condition is true, topic is mapped correctly.
				if condition.Status == "True" {
					return nil
				}
			}
		}
		// wait for another retry
		common.WaitForDuration(retryBackOffTime)
		duration = int(time.Since(ChaosStartTimeStamp).Seconds())
	}
	return errors.Errorf("timeout while waiting for topic %s creation", topicName)
}

func CreateTopic(exp experimentTypes.ExperimentDetails, strimziClient clientset.StrimziV1AlphaClient) error{
	log.InfoWithValues("[Liveness]: Creating the kafka liveness topic:", logrus.Fields{
		"Topic Name": exp.Topic.Name,
		"Replication Factor": exp.Topic.ReplicationFactor,
		"Min In Sync Replica": exp.Topic.MinInSyncReplica,
	})
	err := createTopic(
		exp.Topic.Name,
		exp.App.Namespace,
		exp.Strimzi.StrimziKafkaClusterName,
		exp.Topic.ReplicationFactor,
		1,
		exp.Topic.MinInSyncReplica,
		strimziClient,
	)

	err = waitForTopicCreation(
		exp.Topic.Name,
		exp.App.Namespace,
		exp.Control.Timeout,
		exp.Control.Delay,
		strimziClient,
	)
	if err != nil {
		return err
	}

	return err
}

func DeleteTopic(topicName string, appNs string, strimziClient clientset.StrimziV1AlphaClient) error{
	return strimziClient.KafkaTopic(appNs).Delete(topicName, metav1.DeleteOptions{})
}

// getTopic returns topic by name and namespace, if not found return given
func getTopic(topicName string, appNs string, strimziClient clientset.StrimziV1AlphaClient) (*topicv1.KafkaTopic, error){
	topicObject, err := strimziClient.KafkaTopic(appNs).Get(topicName, metav1.GetOptions{} )
	if err != nil {
		return nil, err
	}
	return topicObject, nil
}

func createTopic(topicName, appNs string, clusterName string, replicas, partitions int, MinInsyncReplicas string, strimziClient clientset.StrimziV1AlphaClient) error {
	topicPayload := topicv1.KafkaTopic{
		ObjectMeta: metav1.ObjectMeta{
			Name: topicName,
			Labels: map[string]string{
				// label must match cluster to which topic should be created
				"strimzi.io/cluster": clusterName,
				"app.kubernetes.io/part-of": "litmus",
			},
		},

		Spec: topicv1.KafkaTopicCrSpecification{
			Replicas: replicas,
			TopicName: topicName,
			Partitions: partitions,
			TopicConf: topicv1.TopicConf{
				MinInsyncReplicas: MinInsyncReplicas,
			},
		},
	}
	_, err := strimziClient.KafkaTopic(appNs).Create(&topicPayload, metav1.CreateOptions{})
	return err
}
