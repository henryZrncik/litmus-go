package client

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)


type KafkaTopic struct {
	// kind
	metav1.TypeMeta   `json:",inline"`
	// al meta data: i.e., name, namespace ...
	metav1.ObjectMeta `json:"metadata,omitempty"`
	//
	Spec KafkaTopicCrSpecification `json:"spec"`
	// path: /status
	Status KafkaStatus `json:"status,omitempty"`
}


type KafkaTopicCrSpecification struct {
	TopicConf TopicConf `json:"config,omitempty"`
	Replicas  int       `json:"replicas,omitempty"`
	TopicName string `json:"topicName"`
	Partitions int `json:"partitions,omitempty"`
}


type TopicConf struct {

	MinInsyncReplicas string `json:"min.insync.replicas,omitempty"`
}


type KafkaTopicList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []KafkaTopic `json:"items"`
}