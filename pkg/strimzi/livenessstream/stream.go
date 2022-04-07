package livenessstream

import (
	"fmt"
	"github.com/litmuschaos/litmus-go/pkg/clients"
	"github.com/litmuschaos/litmus-go/pkg/log"
	experimentTypes "github.com/litmuschaos/litmus-go/pkg/strimzi/types"
	strimziLivenessUtils "github.com/litmuschaos/litmus-go/pkg/strimzi/utils/liveness"
	"github.com/litmuschaos/litmus-go/pkg/utils/common"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
	"time"
)

const (
	obtainTopicLeaderJobPrefix  = "strimzi-liveness-topic-leader-"
	producerJobNamePrefix = "strimzi-liveness-producer-"
	consumerJobNamePrefix = "strimzi-liveness-consumer-"
)

// LivenessStream generates kafka liveness pod, which continuously validate the liveness of kafka brokers
// and derive the kafka topic leader(candidate for the deletion)
//
// returns error if ocured and time when liveness streams (producer and consumers actually start)
func LivenessStream(experimentsDetails *experimentTypes.ExperimentDetails, clients clients.ClientSets) (*time.Time, error) {
	// Generate a random string as suffix to topic name and liveness image in case name was not provided
	experimentsDetails.Control.RunID = common.GetRunID()
	log.InfoWithValues("[Liveness]: liveness stream starts", logrus.Fields{
		"runID": "liveness-"+ experimentsDetails.Control.RunID,
	})

	if experimentsDetails.Topic.Name == "" {
		experimentsDetails.Topic.Name = "topic-" + experimentsDetails.Control.RunID
	}

	// Create topic with specified configuration (i.e., replication factor, name, host)
	if err :=  strimziLivenessUtils.CreateTopic(*experimentsDetails, *experimentsDetails.Strimzi.Client); err != nil {
		return nil, err
	}
	// creation of producer Job
	if err := createProducer(experimentsDetails,clients); err != nil {
		return nil, err
	}
	// creation of consumer Job
	if err := createConsumer(experimentsDetails,clients); err != nil {
		return nil, err
	}

	// wait till pods of producer and consumer are truly created.
	producerJobName := producerJobNamePrefix + experimentsDetails.Control.RunID
	consumerJobName := consumerJobNamePrefix + experimentsDetails.Control.RunID
	log.Infof("[Wait] Waiting for preparation of producer")
	err := strimziLivenessUtils.WaitForJobStart(producerJobName, experimentsDetails.App.Namespace, 30, experimentsDetails.Control.Delay, clients)
	if err != nil {
		return nil, err
	}
	log.Infof("[Wait] Waiting for preparation of consumer")
	err = strimziLivenessUtils.WaitForJobStart(consumerJobName, experimentsDetails.App.Namespace, 30, experimentsDetails.Control.Delay, clients)
	if err != nil {
		return nil, err
	}

	// wait for actual start of pods (not just job noticing it)
	err = strimziLivenessUtils.WaitForRunningJob(producerJobName, experimentsDetails.App.Namespace, clients, experimentsDetails.Control.Delay)
	if err != nil {
		return nil, err
	}
	err = strimziLivenessUtils.WaitForRunningJob(consumerJobName, experimentsDetails.App.Namespace, clients, experimentsDetails.Control.Delay)
	if err != nil {
		return nil, err
	}


	timeNow := time.Now()
	return &timeNow, nil
}

// createProducer pass parameters to producer image job
func createProducer(experimentsDetails *experimentTypes.ExperimentDetails, clients clients.ClientSets) error {
	log.InfoWithValues("[Liveness]: Creating the kafka producer:", logrus.Fields{
		"Kafka Service": experimentsDetails.Kafka.Service,
		"Kafka Port": experimentsDetails.Kafka.Port,
		"Message Creation Delay (Ms)": experimentsDetails.Producer.MessageDelayMs,
		"Acks": experimentsDetails.Producer.Acks,
		"Message Delivery Timeout (Ms)": experimentsDetails.Producer.MessageDeliveryTimeoutMs,
		"Request Timeout (Ms)": experimentsDetails.Producer.RequestTimeoutMs,
		"Message Produced Count": experimentsDetails.Consumer.MessageCount,
		"LOG_LEVEL": experimentsDetails.Consumer.LogLevel,
	})

	var envVariables []corev1.EnvVar = []corev1.EnvVar{
		{
			Name: "TOPIC",
			Value: experimentsDetails.Topic.Name,
		},
		{
			Name: "BOOTSTRAP_SERVERS",
			Value: fmt.Sprintf("%s:%s",experimentsDetails.Kafka.Service, experimentsDetails.Kafka.Port),
		},
		//{
		//	Name: "GROUP_ID",
		//	Value: "litmus",
		//},
		{
			Name: "MESSAGES_PER_TRANSACTION",
			Value: "1",
		},
		{
			Name: "DELAY_MS",
			Value: experimentsDetails.Producer.MessageDelayMs,
		},
		{
			Name: "MESSAGE_COUNT",
			Value: experimentsDetails.Consumer.MessageCount,
		},
		{
			Name: "PRODUCER_ACKS",
			Value: experimentsDetails.Producer.Acks,
		},
		{
			Name: "ADDITIONAL_CONFIG",
			Value: fmt.Sprintf("delivery.timeout.ms=%s\nrequest.timeout.ms=%s",experimentsDetails.Producer.MessageDeliveryTimeoutMs, experimentsDetails.Producer.RequestTimeoutMs) ,
		},
		{
			Name: "BLOCKING_PRODUCER",
			Value: "true",
		},
		{
			Name: "LOG_LEVEL",
			Value: experimentsDetails.Consumer.LogLevel,
		},
	}
	jobName := producerJobNamePrefix + experimentsDetails.Control.RunID

	if err := strimziLivenessUtils.CreateJob(jobName, experimentsDetails.Control.RunID, experimentsDetails.Producer.ProducerImage, experimentsDetails.App.Namespace, "", envVariables, clients); err != nil {
		return err
	}
	return nil

}


func createConsumer(experimentsDetails *experimentTypes.ExperimentDetails, clients clients.ClientSets) error {
	log.InfoWithValues("[Liveness]: Creating the kafka consumer:", logrus.Fields{
		"Kafka service": experimentsDetails.Kafka.Service,
		"Kafka port": experimentsDetails.Kafka.Port,
		"Message  Count": experimentsDetails.Consumer.MessageCount,
		"Group Id": experimentsDetails.Control.RunID,
		"Topic Name": experimentsDetails.Topic.Name,
		"LOG_LEVEL": experimentsDetails.Consumer.LogLevel,
		"Additional configuration": experimentsDetails.Consumer.AdditionalConfig,
	})

	var envVariables []corev1.EnvVar = []corev1.EnvVar{
		{
			Name: "TOPIC",
			Value: experimentsDetails.Topic.Name,
		},
		{
			Name: "BOOTSTRAP_SERVERS",
			Value: fmt.Sprintf("%s:%s",experimentsDetails.Kafka.Service, experimentsDetails.Kafka.Port),
		},
		{
			Name: "GROUP_ID",
			Value: "litmus",
		},
		{
			Name:  "MESSAGE_COUNT",
			Value: experimentsDetails.Consumer.MessageCount,
		},
		{
			Name: "LOG_LEVEL",
			Value: experimentsDetails.Consumer.LogLevel,
		},
		{
			Name: "ADDITIONAL_CONFIG",
			Value: experimentsDetails.Consumer.AdditionalConfig,
		},
	}
	jobName := consumerJobNamePrefix + experimentsDetails.Control.RunID

	if err := strimziLivenessUtils.CreateJob(jobName, experimentsDetails.Control.RunID, experimentsDetails.Consumer.ConsumerImage, experimentsDetails.App.Namespace, "", envVariables, clients); err != nil {
		return err
	}

	return nil

}


func VerifyLivenessStream(exp *experimentTypes.ExperimentDetails, clients clients.ClientSets, startedTime *time.Time) error {
	log.Infof("[Liveness]: stream verification")
	consumerJobName := consumerJobNamePrefix + exp.Control.RunID
	producerJobName := producerJobNamePrefix + exp.Control.RunID

	duration := int(time.Since(*startedTime).Seconds())

	restDuration := 0
	if duration < exp.App.LivenessDuration {
		restDuration = exp.App.LivenessDuration  - duration
	}

	log.Infof("[Wait]: Waiting for finish of producer and consumer container. Duration of liveness stream is %d/%d",duration, exp.App.LivenessDuration)
	log.Infof("[Info]: Waiting for end of liveness stream for additional %d seconds",restDuration)
	// while duration of liveness wasn't reached "running" state means that we still wait

	err := strimziLivenessUtils.WaitForJobEnd(producerJobName, exp.App.Namespace, restDuration, exp.Control.Delay, clients)
	if err != nil {
		return errors.Errorf("Producer container. %s",err.Error())
	}
	// consumer gets 5 extra seconds to finish possible last second message
	err = strimziLivenessUtils.WaitForJobEnd(consumerJobName, exp.App.Namespace, restDuration + 5, exp.Control.Delay, clients)
	if err != nil {
		return errors.Errorf("Consumer container. %s",err.Error())
	}
	log.Infof("[Liveness]: Producer and Consumer finished successfully")
	return nil
}


// GetPartitionLeaderInstanceName get partition Leader, returns name of instance that is partition leader of given topic (its 1st partition)
func GetPartitionLeaderInstanceName(experimentsDetails *experimentTypes.ExperimentDetails, clients clients.ClientSets) (string,error) {
	command := fmt.Sprintf("kafka-topics --topic %s --describe --bootstrap-server %s:%s | grep -o 'Leader: [^[:space:]]*' | awk '{print $2}'",
		experimentsDetails.Topic.Name,
		experimentsDetails.Kafka.Service,
		experimentsDetails.Kafka.Port,
	)
	jobName := obtainTopicLeaderJobPrefix + experimentsDetails.Control.RunID

	if err := strimziLivenessUtils.CreateJob(jobName, experimentsDetails.Control.RunID, experimentsDetails.App.LivenessImage, experimentsDetails.App.Namespace, command, nil, clients); err != nil {
		return "", err
	}

	if err := strimziLivenessUtils.WaitForJobEnd(jobName, experimentsDetails.App.Namespace, experimentsDetails.Control.Timeout, experimentsDetails.Control.Delay, clients); err != nil {
		return "", err
	}

	partitionLeaderId, err := strimziLivenessUtils.GetJobLogs(jobName, experimentsDetails.App.Namespace, clients)
	if err != nil {
		return "", err
	}

	log.Info("[Liveness]: Determine the leader broker pod name")
	podList, err := clients.KubeClient.CoreV1().Pods(experimentsDetails.App.Namespace).List(metav1.ListOptions{LabelSelector: experimentsDetails.Control.AppLabel})
	if err != nil {
		return "", errors.Errorf("unable to find the pods with matching labels, err: %v", err)
	}
	//
	for _, pod := range podList.Items {
		if strings.ContainsAny(pod.Name, partitionLeaderId) {
			return pod.Name, nil
		}
	}
	return "", errors.Errorf("no kafka pod found with %v partition leader ID", partitionLeaderId)
}


