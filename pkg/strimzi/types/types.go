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
	ClusterOperator 	*ClusterOperator
	Resources			*Resources
	Producer   		    *Producer
	Images				*Images
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
	//AppLabel            string
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
	Namespace      string
	LivenessStream string
	LivenessStreamCleanup string
}

// Kafka
type Kafka struct {

	KafkaPartitionLeaderKill   string
	KafkaInstancesName         string
	// how to connect to kafka from liveness probe pods.
	Port                       string
	Service 				   string
	// where to check Kafka pods as default check
	Label   				   string
	 // Deprecated moved to App
	//Namespace 				   string

}

type Strimzi struct {
	Client 	*experimentClientSet.ExampleV1Alpha1Client
	StrimziKafkaClusterName string
	InternalListenerPortNumber int
	InternalListenerName string

}

type Topic struct {
	// Topic
	Name              string
	ReplicationFactor string
	MinInSyncReplica  string

}

type Producer struct {
	MessageCount   string
	MessageDelayMs string
	// values "all", "0", "1"
	Acks 	       string

}

type Consumer struct {
	TimeoutMs    string
	MessageCount string
}

type Images struct {
	KafkaImage 		string
	ProducerImage 	string
}

type ClusterOperator struct{
	// if and where  to check operator
	Namespace string
	Label     string
}

type Resources struct {
	ConfigMaps  string
	Secrets     string
	Services    string
	Resources   []KubernetesResource
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
