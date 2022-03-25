package livenessstream

import (
	"fmt"
	"github.com/litmuschaos/litmus-go/pkg/clients"
	"github.com/litmuschaos/litmus-go/pkg/log"
	experimentTypes "github.com/litmuschaos/litmus-go/pkg/strimzi/types"
	strimziJobs "github.com/litmuschaos/litmus-go/pkg/strimzi/utils/livenessjobs"
	"github.com/litmuschaos/litmus-go/pkg/utils/common"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"

	"time"
)

const (
	obtainTopicLeaderJobPrefix  = "strimzi-liveness-topic-leader"
	topicJobNamePrefix          = "strimzi-liveness-topic-"
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
	if err := createTopic(experimentsDetails, clients); err != nil {
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
	err := strimziJobs.WaitForRunningJob(producerJobName, experimentsDetails.App.Namespace, clients, 2)
	if err != nil {
		return nil, err
	}
	log.Infof("[Wait] Waiting for preparation of consumer")
	err = strimziJobs.WaitForRunningJob(consumerJobName, experimentsDetails.App.Namespace, clients, 2)
	if err != nil {
		return nil, err
	}
	timeNow := time.Now()
	return &timeNow, nil
}

// createTopic creates the kafka liveness pod
func createTopic(experimentsDetails *experimentTypes.ExperimentDetails, clients clients.ClientSets) error {
	log.InfoWithValues("[Liveness]: Creating the kafka liveness topic:", logrus.Fields{
		"Topic Name": experimentsDetails.Topic.Name,
		"Replication Factor": experimentsDetails.Topic.ReplicationFactor,
		"Min In Sync Replica": experimentsDetails.Topic.MinInSyncReplica,
	})

	command := fmt.Sprintf("kafka-topics --bootstrap-server %s:%s --topic %s --create --partitions 1 --replication-factor %s --config min.insync.replicas=%s",
		experimentsDetails.Kafka.Service,
		experimentsDetails.Kafka.Port,
		experimentsDetails.Topic.Name,
		experimentsDetails.Topic.ReplicationFactor,
		experimentsDetails.Topic.MinInSyncReplica,
	)
	jobName := topicJobNamePrefix + experimentsDetails.Control.RunID
	cmdImage := experimentsDetails.Consumer.ConsumerImage
	jobNamespace := experimentsDetails.App.Namespace

	// create Job that will create Topic
	if err := strimziJobs.ExecKube(jobName, experimentsDetails.Control.RunID, cmdImage, jobNamespace, command, nil, clients); err != nil {
		return  err
	}

	// wait for topic to be created/ fail
	log.Info("[Wait]: Waiting for creation of the test topic")
	if err:= strimziJobs.WaitForExecPod(jobName, jobNamespace, experimentsDetails.Control.Timeout, experimentsDetails.Control.Delay, clients, "Creation of the test topic"); err != nil {
		return err
	}


	return  nil
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
		"Message Produced Count": experimentsDetails.Producer.MessageCount,
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
			Name: "MESSAGES_PER_TRANSACTION",
			Value: "1",
		},
		{
			Name: "DELAY_MS",
			Value: experimentsDetails.Producer.MessageDelayMs,
		},
		{
			Name: "MESSAGE_COUNT",
			Value: experimentsDetails.Producer.MessageCount,
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
	}
	jobName := producerJobNamePrefix + experimentsDetails.Control.RunID

	if err := strimziJobs.ExecKube(jobName, experimentsDetails.Control.RunID, experimentsDetails.Producer.ProducerImage, experimentsDetails.App.Namespace, "", envVariables, clients); err != nil {
		return err
	}

	return nil

}


func createConsumer(experimentsDetails *experimentTypes.ExperimentDetails, clients clients.ClientSets) error {
	//log.InfoWithValues("[Liveness]: Creating the kafka consumer:", logrus.Fields{
	//	"Kafka service": experimentsDetails.Kafka.Service,
	//	"Kafka port": experimentsDetails.Kafka.Port,
	//	"Message Produced Count": experimentsDetails.Consumer.MessageCount,
	//	"Consumer Timeout (Ms)": experimentsDetails.Consumer.TimeoutMs,
	//	"Retry (Ms)": experimentsDetails.Consumer.ConsumerRetryBackOffMs,
	//
	//})
	//
	//
	//command := fmt.Sprintf("kafka-console-consumer --property print.timestamp=true --bootstrap-server %s:%s --topic %s --from-beginning  --max-messages %d --timeout-ms %s",
	//	experimentsDetails.Kafka.Service,
	//	experimentsDetails.Kafka.Port,
	//	experimentsDetails.Topic.Name,
	//	experimentsDetails.Consumer.MessageCount,
	//	experimentsDetails.Consumer.TimeoutMs,
	//)




	log.InfoWithValues("[Liveness]: Creating the kafka consumer:", logrus.Fields{
		"Kafka service": experimentsDetails.Kafka.Service,
		"Kafka port": experimentsDetails.Kafka.Port,
		"Message Produced Count": experimentsDetails.Consumer.MessageCount,
		"Group Id": experimentsDetails.Control.RunID,
		"Topic Name": experimentsDetails.Topic.Name,
		//"Retry (Ms)": experimentsDetails.Consumer.ConsumerRetryBackOffMs,
		//"Reconnect (Ms)": experimentsDetails.Consumer.ConsumerReconnectBackOffMs,
		"LOG_LEVEL": "INFO",
		"Consumer Timeout EXTRA FOR NOW (Ms)": experimentsDetails.Consumer.TimeoutMs,
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
			Name: "MESSAGES_PER_TRANSACTION",
			Value: "1",
		},
		{
			Name: "DELAY_MS",
			Value: experimentsDetails.Producer.MessageDelayMs,
		},
		{
			Name:  "MESSAGE_COUNT",
			Value: fmt.Sprintf("%d",experimentsDetails.Consumer.MessageCount),
		},
		{
			Name: "LOG_LEVEL",
			Value: "INFO",
		},
		{
			Name: "ADDITIONAL_CONFIG",
			Value: fmt.Sprintf("retry.backoff.ms=%s\nreconnect.backoff.ms=%s",experimentsDetails.Consumer.RetryBackoffMs, experimentsDetails.Consumer.RetryBackoffMs) ,
		},
		{
			Name: "BLOCKING_PRODUCER",
			Value: "true",
		},
	}
	jobName := consumerJobNamePrefix + experimentsDetails.Control.RunID

	if err := strimziJobs.ExecKube(jobName, experimentsDetails.Control.RunID, "quay.io/strimzi-examples/java-kafka-consumer:latest", experimentsDetails.App.Namespace, "", envVariables, clients); err != nil {
		return err
	}

	return nil

}


func VerifyLivenessStream(experimentsDetails *experimentTypes.ExperimentDetails, clients clients.ClientSets, startedTime *time.Time) error {
	log.Infof("[Liveness]: stream verification")
	consumerJobName := consumerJobNamePrefix + experimentsDetails.Control.RunID
	producerJobName := producerJobNamePrefix + experimentsDetails.Control.RunID

	// consumer either did not start or we have timeout problem which will be reported
	log.InfoWithValues("[Wait]: Waiting for finish of consumer container, at most 1 timeout duration", logrus.Fields{
		"Timeout": experimentsDetails.Control.Timeout,
	})

	duration := int(time.Since(*startedTime).Seconds())

	// while duration of liveness wasn't reached "running" state means that we still wait
	for  {
		duration = int(time.Since(*startedTime).Seconds())
		consumerResult, _ := strimziJobs.GetJobResult(consumerJobName,experimentsDetails.App.Namespace,clients)
		producerResult, _ := strimziJobs.GetJobResult(producerJobName,experimentsDetails.App.Namespace,clients)
		// parse producer result
		repeatProducer, errProducer := strimziJobs.ParseJobResult(producerResult, duration < experimentsDetails.App.LivenessDuration)
		// consumer has 5 extra seconds to finish after
		repeatConsumer, errConsumer := strimziJobs.ParseJobResult(consumerResult, duration < (experimentsDetails.App.LivenessDuration + 5))

		if errProducer != nil {
			return errors.Errorf("Producer %v", errProducer.Error())
		}
		if errConsumer != nil {
			return errors.Errorf("Consumer %v", errConsumer.Error())
		}
		if !repeatProducer && !repeatConsumer {
			return nil
		}
		common.WaitForDuration(experimentsDetails.Control.Delay)

	}
}





	// pokym som s casom nizsie pozeram ze running je ok

	// akonahle som  startedTime prekonal  clientTimeout


	//if err:= strimziJobs.WaitForExecPod(jobName, experimentsDetails.App.Namespace, experimentsDetails.Control.Timeout, experimentsDetails.Control.Delay, clients, "successful completion of consumer process"); err != nil {
	//	return err
	//}
	//
	//logs, err := strimziJobs.GetJobLogs(jobName, experimentsDetails.App.Namespace, clients)
	//if err != nil {
	//	return err
	//}
	//
	//re := regexp.MustCompile(`Processed\D*(\d+).*`)
	//
	//if re.MatchString(logs){
	//	match := re.FindStringSubmatch(logs)
	//	actualNumberOfConsumedMessages, _ := strconv.Atoi(match[1])
	//	if actualNumberOfConsumedMessages < experimentsDetails.Consumer.MessageCount {
	//		return errors.New(fmt.Sprintf("failed to consume expected number of messages. Expected:%d, Obtained:%d",
	//			experimentsDetails.Consumer.MessageCount,
	//			actualNumberOfConsumedMessages,
	//		))
	//	}
	//	// if logs parsed successfully and counts match, all went well.
	//	log.Infof("[Info]: consumers consumed expected number of messages: %d", experimentsDetails.Consumer.MessageCount)
	//	return nil
	//}
	//
	//return errors.New("unable to parse output from consumer container, number of processed messages is expected")


// GetPartitionLeaderInstanceName get partition Leader, returns name of instance that is partition leader of given topic (its 1st partition)
func GetPartitionLeaderInstanceName(experimentsDetails *experimentTypes.ExperimentDetails, clients clients.ClientSets) (string,error) {
	command := fmt.Sprintf("kafka-topics --topic %s --describe --bootstrap-server %s:%s | grep -o 'Leader: [^[:space:]]*' | awk '{print $2}'",
		experimentsDetails.Topic.Name,
		experimentsDetails.Kafka.Service,
		experimentsDetails.Kafka.Port,
	)
	jobName := obtainTopicLeaderJobPrefix + experimentsDetails.Control.RunID

	if err := strimziJobs.ExecKube(jobName, experimentsDetails.Control.RunID, experimentsDetails.Consumer.ConsumerImage, experimentsDetails.App.Namespace, command, nil, clients); err != nil {
		return "", err
	}

	if err := strimziJobs.WaitForExecPod(jobName, experimentsDetails.App.Namespace, experimentsDetails.Control.Timeout, experimentsDetails.Control.Delay, clients, "Obtaining partition leader info"); err != nil {
		return "", err
	}

	partitionLeaderId, err := strimziJobs.GetJobLogs(jobName, experimentsDetails.App.Namespace, clients)
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


