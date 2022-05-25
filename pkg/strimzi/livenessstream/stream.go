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
func LivenessStream(experimentsDetails *experimentTypes.ExperimentDetails, clients clients.ClientSets) error {
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
		return err
	}
	// creation of producer Job
	if err := createProducer(experimentsDetails,clients); err != nil {
		return err
	}
	// creation of consumer Job
	if err := createConsumer(experimentsDetails,clients); err != nil {
		return err
	}

	// wait till pods of producer and consumer are truly created.
	producerJobName := producerJobNamePrefix + experimentsDetails.Control.RunID
	consumerJobName := consumerJobNamePrefix + experimentsDetails.Control.RunID
	log.Infof("[Wait] Waiting for preparation of producer")
	err := strimziLivenessUtils.WaitForJobStart(producerJobName, experimentsDetails.App.Namespace, 30, experimentsDetails.Control.Delay, clients)
	if err != nil {
		return err
	}
	log.Infof("[Wait] Waiting for preparation of consumer")
	err = strimziLivenessUtils.WaitForJobStart(consumerJobName, experimentsDetails.App.Namespace, 30, experimentsDetails.Control.Delay, clients)
	if err != nil {
		return err
	}

	// wait for actual start of pods (not just job noticing it)
	err = strimziLivenessUtils.WaitForRunningJob(producerJobName, experimentsDetails.App.Namespace, clients, experimentsDetails.Control.Delay)
	if err != nil {
		return err
	}
	err = strimziLivenessUtils.WaitForRunningJob(consumerJobName, experimentsDetails.App.Namespace, clients, experimentsDetails.Control.Delay)
	if err != nil {
		return err
	}

	return nil
}

// createProducer pass parameters to producer image job
func createProducer(exp *experimentTypes.ExperimentDetails, clients clients.ClientSets) error {
	log.InfoWithValues("[Liveness]: Creating the kafka producer:", logrus.Fields{
		"Kafka Service":                 exp.Kafka.Service,
		"Kafka Port":                    exp.Kafka.Port,
		"Message Creation Delay (Ms)":   exp.Producer.MessageDelayMs,
		"Acks":                          exp.Producer.Acks,
		"Message Delivery Timeout (Ms)": exp.Producer.MessageDeliveryTimeoutMs,
		"Request Timeout (Ms)":          exp.Producer.RequestTimeoutMs,
		"Message Produced Count":        exp.Consumer.MessageCount,
		"LOG_LEVEL":                     exp.Consumer.LogLevel,
	})


	var envVariables []corev1.EnvVar = []corev1.EnvVar{
		{
			Name:  "TOPIC",
			Value: exp.Topic.Name,
		},
		{
			Name: "BOOTSTRAP_SERVERS",
			Value: fmt.Sprintf("%s:%s", exp.Kafka.Service, exp.Kafka.Port),
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
			Name:  "DELAY_MS",
			Value: exp.Producer.MessageDelayMs,
		},
		{
			Name:  "MESSAGE_COUNT",
			Value: exp.Consumer.MessageCount,
		},
		{
			Name:  "PRODUCER_ACKS",
			Value: exp.Producer.Acks,
		},
		{
			Name: "ADDITIONAL_CONFIG",
			Value: fmt.Sprintf("delivery.timeout.ms=%s\nrequest.timeout.ms=%s", exp.Producer.MessageDeliveryTimeoutMs, exp.Producer.RequestTimeoutMs) ,
		},
		{
			Name: "BLOCKING_PRODUCER",
			Value: "true",
		},
		{
			Name:  "LOG_LEVEL",
			Value: exp.Consumer.LogLevel,
		},
	}
	// append acces envs if present
	if exp.Access.AuthorizationAndAuthenticationMethod == "tls" {
		extraVariables := getAccessEnvironmentVariables(exp.Access.ProducerKafkaUserName, exp.Access.ClusterCertificate)
		envVariables = append(envVariables, extraVariables...)
	}


	jobName := producerJobNamePrefix + exp.Control.RunID

	if err := strimziLivenessUtils.CreateJob(jobName, exp.Control.RunID, exp.Producer.ProducerImage, exp.App.Namespace, "", envVariables, clients); err != nil {
		return err
	}
	return nil

}


func createConsumer(exp *experimentTypes.ExperimentDetails, clients clients.ClientSets) error {
	log.InfoWithValues("[Liveness]: Creating the kafka consumer:", logrus.Fields{
		"Kafka service":            exp.Kafka.Service,
		"Kafka port":               exp.Kafka.Port,
		"Message  Count":           exp.Consumer.MessageCount,
		"Group Id":                 exp.Control.RunID,
		"Topic Name":               exp.Topic.Name,
		"LOG_LEVEL":                exp.Consumer.LogLevel,
		"Additional configuration": exp.Consumer.AdditionalConfig,
	})

	var envVariables []corev1.EnvVar = []corev1.EnvVar{
		{
			Name:  "TOPIC",
			Value: exp.Topic.Name,
		},
		{
			Name: "BOOTSTRAP_SERVERS",
			Value: fmt.Sprintf("%s:%s", exp.Kafka.Service, exp.Kafka.Port),
		},
		{
			Name: "GROUP_ID",
			Value: "litmus",
		},
		{
			Name:  "MESSAGE_COUNT",
			Value: exp.Consumer.MessageCount,
		},
		{
			Name:  "LOG_LEVEL",
			Value: exp.Consumer.LogLevel,
		},
		{
			Name:  "ADDITIONAL_CONFIG",
			Value: exp.Consumer.AdditionalConfig,
		},
	}

	// append acces envs if present
	if exp.Access.AuthorizationAndAuthenticationMethod == "tls" {
		extraVariables := getAccessEnvironmentVariables(exp.Access.ConsumerKafkaUserName, exp.Access.ClusterCertificate)
		envVariables = append(envVariables, extraVariables...)
	}

	jobName := consumerJobNamePrefix + exp.Control.RunID

	if err := strimziLivenessUtils.CreateJob(jobName, exp.Control.RunID, exp.Consumer.ConsumerImage, exp.App.Namespace, "", envVariables, clients); err != nil {
		return err
	}

	return nil

}


func VerifyLivenessStream(exp *experimentTypes.ExperimentDetails, clients clients.ClientSets) error {
	log.Infof("[Liveness]: stream verification")
	consumerJobName := consumerJobNamePrefix + exp.Control.RunID
	producerJobName := producerJobNamePrefix + exp.Control.RunID

	restDuration := exp.App.LivenessDuration
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

// returns list of environment variables for clients (Producer and Consumer based on choosen authorization and authentication) based
func getAccessEnvironmentVariables(kafkaUser, clusterCa string) []corev1.EnvVar {
	// supported TLS option

		var envVariables []corev1.EnvVar = []corev1.EnvVar{
			{
				Name: "CA_CRT",
				ValueFrom:  &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							clusterCa,
						},
						Key:                  "ca.crt",
						Optional:             nil,
					},
				},
			},
			{
				Name: "USER_CRT",
				ValueFrom:  &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							kafkaUser,
						},
						Key:                  "user.crt",
						Optional:             nil,
					},
				},
			},
			{
				Name: "USER_KEY",
				ValueFrom:  &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							kafkaUser,
						},
						Key:                  "user.key",
						Optional:             nil,
					},
				},
			},

		}
		return envVariables

}


