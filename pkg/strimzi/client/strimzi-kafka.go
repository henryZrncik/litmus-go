package client

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// Kafka is object for representation of Kafka CR.
type Kafka struct {
	// kind
	metav1.TypeMeta   `json:",inline"`
	// al meta data: i.e., name, namespace ...
	metav1.ObjectMeta `json:"metadata,omitempty"`
	//
	Spec KafkaCrSpecification `json:"spec"`
	// path: /status
	Status Status `json:"status,omitempty"`
}

// KafkaCrSpecification is json path: /spec
type KafkaCrSpecification struct {
	Spec KafkaSpec `json:"kafka"`
}

// KafkaSpec is in json path: /spec/kafka
type KafkaSpec struct {
	Replicas int `json:"replicas"`
	// path: /spec/kafka/listeners
	Listeners []Listener `json:"listeners"`
}

// Listener is in path: /spec/kafka/listeners
type Listener struct {
	Name string `json:"name"`
	Port int `json:"port"`
	TLS  bool `json:"tls"`
	Type string `json:"type"`
}

// Status path: /status
type Status struct {
	// path: /status/conditions
	Conditions []metav1.Status `json:"conditions,omitempty"`
}

type KafkaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Kafka `json:"items"`
}