package client

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type KafkaConnector struct {
	// kind
	metav1.TypeMeta   `json:",inline"`
	// al meta data: i.e., name, namespace ...
	metav1.ObjectMeta `json:"metadata,omitempty"`
	//
	ConnectorStatus ConnectorStatus `json:"status,omitempty"`
}

// ConnectorStatus path: /status
type ConnectorStatus struct {
	// path: /status/conditions
	ConnectorStatusInner `json:"connectorStatus,omitempty"`
}

// ConnectorStatusInner path: /status/connectorStatus
type ConnectorStatusInner struct {
	Tasks []Tasks `json:"tasks"`
}


// Tasks is in path: /status/connectorStatus/tasks
type Tasks struct {
	Id int `json:"id"`
	State string `json:"state"`
	// format <ip>:<port>
	WorkerId  string `json:"worker_id"`
}




type KafkaConnectorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []KafkaConnector `json:"items"`
}