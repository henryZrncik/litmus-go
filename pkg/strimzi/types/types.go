package types

import (
	experimentClientSet "github.com/litmuschaos/litmus-go/pkg/strimzi/client/clientset"
	clientTypes "k8s.io/apimachinery/pkg/types"
)

type ExperimentDetails struct {
	Control				*Control
	App					*App
	Kafka				*Kafka
	Topic 				*Topic
	Resources			*Resources
	Producer   		    *Producer
	Consumer			*Consumer
	Strimzi				*Strimzi
}

// Control is for collecting all the experiment-related details
type Control struct {
	ExperimentName      string
	EngineName          string
	ChaosDuration       int
	ChaosInterval       string
	RampTime            int
	Force               bool
	ChaosLib            string
	ChaosServiceAccount string
	//AppNS               string
	AppLabel            string
	AppKind             string
	ChaosUID            clientTypes.UID
	InstanceID          string
	ChaosNamespace      string
	ChaosPodName        string
	Timeout             int
	Delay               int
	TargetPods          string
	PodsAffectedPerc    int
	Sequence            string
	LIBImagePullPolicy  string
	TargetContainer     string
	RunID				string

}

type App struct {
	Namespace                  string
	LivenessStream             string
	LivenessStreamJobsCleanup  string
	LivenessStreamTopicCleanup string
	LivenessDuration           int
	LivenessImage			   string
}

// Kafka
type Kafka struct {

	KafkaPartitionLeaderKill string
	KafkaInstancesName       string
	// how to connect to kafka from liveness probe pods.
	Port                     string
	Service 				 string
}

type Strimzi struct {
	Client 	*experimentClientSet.StrimziV1AlphaClient
	StrimziKafkaClusterName    string
	InternalListenerPortNumber int
	InternalListenerName       string

}

type Topic struct {
	// Topic
	Name              string
	ReplicationFactor int
	MinInSyncReplica  string

}

type Producer struct {
	ProducerImage            string
	// after each unit of ms next message will be sent
	MessageDelayMs           string
	// after unit of ms message will not be delivered and next one will be created. "delivery.timeout.ms" configuration from kafka
	MessageDeliveryTimeoutMs string
	// max time of waiting before trying to repeat request regarding sending message. "request.timeout.ms" configuration from kafka
	RequestTimeoutMs         string
	// values "all", "0", "1"
	Acks 	                 string

}

type Consumer struct {
	ConsumerImage    string
	MessageCount     string
	LogLevel         string
	AdditionalConfig string
}


type Resources struct {
	ConfigMaps string
	Secrets    string
	Services   string
	Resources  []KubernetesResource
}

type ResourceType string

const (
	ConfigMapResourceType ResourceType = "configuration map"
	SecretResourceType                 = "secret"
	ServiceResourceType                = "service"
)

type KubernetesResource struct {
	Name string
	Type ResourceType
}
