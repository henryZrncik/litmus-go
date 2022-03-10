package liveness

import (
	"fmt"
	"github.com/litmuschaos/litmus-go/pkg/clients"
	"github.com/litmuschaos/litmus-go/pkg/log"
	experimentTypes "github.com/litmuschaos/litmus-go/pkg/strimzi/types"
	strimziJobs "github.com/litmuschaos/litmus-go/pkg/strimzi/utils/jobs"
	"github.com/litmuschaos/litmus-go/pkg/utils/common"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"regexp"
	"strings"
)

const (
	obtainTopicLeaderJobPrefix  = "strimzi-liveness-topic-leader"
	topicJobNamePrefix          = "strimzi-liveness-topic-"
	producerJobNamePrefix = "strimzi-liveness-producer-"
	consumerJobNamePrefix = "strimzi-liveness-consumer-"
)

// LivenessStream generates kafka liveness pod, which continuously validate the liveness of kafka brokers
// and derive the kafka topic leader(candidate for the deletion)
func LivenessStream(experimentsDetails *experimentTypes.ExperimentDetails, clients clients.ClientSets) error {
	// Generate a random string as suffix to topic name and liveness image in case name was not provided
	experimentsDetails.Control.RunID = common.GetRunID()
	log.InfoWithValues("[Liveness]: liveness checks runs with ID  ", logrus.Fields{
		"runID": "liveness-"+ experimentsDetails.Control.RunID,
	})

	if experimentsDetails.Topic.Name == "" {
		experimentsDetails.Topic.Name = "topic-" + experimentsDetails.Control.RunID
	}

	// Create topic with specified configuration (i.e., replication factor, name, host)
	log.InfoWithValues("[Liveness]: Creating the kafka liveness topic:", logrus.Fields{
		"Kafka Topic Name": experimentsDetails.Topic.Name,
	})
	if err := createTopic(experimentsDetails, clients); err != nil {
		return err
	}

	if err := createProducer(experimentsDetails,clients); err != nil {
		return err
	}

	if err := createConsumer(experimentsDetails,clients); err != nil {
		return err
	}

	return nil
}

// createTopic creates the kafka liveness pod
func createTopic(experimentsDetails *experimentTypes.ExperimentDetails, clients clients.ClientSets) error {

	command := fmt.Sprintf("kafka-topics --bootstrap-server %s:%s --topic %s --create --partitions 1 --replication-factor %s --config min.insync.replicas=%s",
		experimentsDetails.Kafka.Service,
		experimentsDetails.Kafka.Port,
		experimentsDetails.Topic.Name,
		experimentsDetails.Topic.ReplicationFactor,
		experimentsDetails.Topic.MinInSyncReplica,
	)
	jobName := topicJobNamePrefix + experimentsDetails.Control.RunID
	cmdImage := experimentsDetails.Images.KafkaImage
	jobNamespace := experimentsDetails.App.Namespace

	// create Job that will create Topic
	log.Infof("[liveness]: creation of topic %s, in order to use it for liveness stream", experimentsDetails.Topic.Name)
	if err := strimziJobs.ExecKube(jobName, cmdImage, jobNamespace, command, nil, clients); err != nil {
		return err
	}

	// wait for topic to be created/ fail.
	if err:= strimziJobs.WaitForExecPod(jobName, jobNamespace, experimentsDetails.Control.Timeout, experimentsDetails.Control.Delay, clients, "Creation of the test topic"); err != nil {
		return err
	}

	return nil
}

func createProducer(experimentsDetails *experimentTypes.ExperimentDetails, clients clients.ClientSets) error {
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
	}
	jobName := producerJobNamePrefix + experimentsDetails.Control.RunID

	if err := strimziJobs.ExecKube(jobName, experimentsDetails.Images.ProducerImage, experimentsDetails.App.Namespace, "", envVariables, clients); err != nil {
		return err
	}

	return nil

}


func createConsumer(experimentsDetails *experimentTypes.ExperimentDetails, clients clients.ClientSets) error {
	command := fmt.Sprintf("kafka-console-consumer --bootstrap-server %s:%s --topic %s --from-beginning  --max-messages %s --timeout-ms %s",
		experimentsDetails.Kafka.Service,
		experimentsDetails.Kafka.Port,
		experimentsDetails.Topic.Name,
		experimentsDetails.Consumer.MessageCount,
		experimentsDetails.Consumer.TimeoutMs,
	)
	jobName := consumerJobNamePrefix + experimentsDetails.Control.RunID
	cmdImage := experimentsDetails.Images.KafkaImage
	jobNamespace := experimentsDetails.App.Namespace

	// create Job that will create Topic
	if err := strimziJobs.ExecKube(jobName, cmdImage, jobNamespace, command, nil, clients); err != nil {
		return err
	}

	return nil

}


func VerifyLivenessStream(experimentsDetails *experimentTypes.ExperimentDetails, clients clients.ClientSets) error {
	jobName := consumerJobNamePrefix + experimentsDetails.Control.RunID

	// consumer either did not start or we have timeout problem which will be reported
	if err:= strimziJobs.WaitForExecPod(jobName, experimentsDetails.App.Namespace, experimentsDetails.Control.Timeout, experimentsDetails.Control.Delay, clients, "Finishing of consumer process"); err != nil {
	}

	logs, err := strimziJobs.GetJobLogs(jobName, experimentsDetails.App.Namespace, clients)
	if err != nil {
		return err
	}

	re := regexp.MustCompile(`Processed\D*(\d+).*`)

	if re.MatchString(logs){
		match := re.FindStringSubmatch(logs)
		var actualNumberOfConsumedMessages string = match[1]
		if actualNumberOfConsumedMessages != experimentsDetails.Consumer.MessageCount {
			log.Errorf("[Info]: consumers did not consume expected number of messages: %s/%s", experimentsDetails.Consumer.MessageCount, actualNumberOfConsumedMessages)
			return errors.New(fmt.Sprintf("failed to consume all messages. Expected: %s, Obtained: %s",
				experimentsDetails.Consumer.MessageCount,
				actualNumberOfConsumedMessages,
			))
		}
		// if logs parsed successfully and counts match, all went well.
		log.Infof("[Info]: consumers consumed expected number of messages: %s", experimentsDetails.Consumer.MessageCount)
		return nil
	}

	return errors.New("unable to parse output from consumer container")

}

// GetPartitionLeaderInstanceName get partition Leader, returns name of instance that is partition leader of given topic (its 1st partition)
func GetPartitionLeaderInstanceName(experimentsDetails *experimentTypes.ExperimentDetails, clients clients.ClientSets) (string,error) {
	command := fmt.Sprintf("kafka-topics --topic %s --describe --bootstrap-server %s:%s | grep -o 'Leader: [^[:space:]]*' | awk '{print $2}'",
		experimentsDetails.Topic.Name,
		experimentsDetails.Kafka.Service,
		experimentsDetails.Kafka.Port,
	)
	jobName := obtainTopicLeaderJobPrefix + experimentsDetails.Control.RunID

	if err := strimziJobs.ExecKube(jobName, experimentsDetails.Images.KafkaImage, experimentsDetails.App.Namespace, command, nil, clients); err != nil {
		return "", err
	}

	if err := strimziJobs.WaitForExecPod(jobName, experimentsDetails.App.Namespace, 30, experimentsDetails.Control.Delay, clients, "Obtaining partition leader info"); err != nil {
		return "", err
	}

	partitionLeaderId, err := strimziJobs.GetJobLogs(jobName, experimentsDetails.App.Namespace, clients)
	if err != nil {
		return "", err
	}

	log.Info("[Liveness]: Determine the leader broker pod name")
	podList, err := clients.KubeClient.CoreV1().Pods(experimentsDetails.App.Namespace).List(metav1.ListOptions{LabelSelector: experimentsDetails.Kafka.Label})
	if err != nil {
		return "", errors.Errorf("unable to find the pods with matching labels, err: %v", err)
	}
	//
	for _, pod := range podList.Items {
		if strings.ContainsAny(pod.Name, partitionLeaderId) {
			log.Infof("[Info]: Leader kafka instance for test topic is %s", experimentsDetails.Kafka.KafkaInstancesName)
			return pod.Name, nil
		}
	}

	return "", errors.Errorf("no kafka pod found with %v partition leader ID", partitionLeaderId)

}


