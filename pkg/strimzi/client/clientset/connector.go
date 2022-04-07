package clientset

import (
	"github.com/litmuschaos/litmus-go/pkg/strimzi/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)



func (c *StrimziV1AlphaClient) KafkaConnector(namespace string) ConnectorInterface {
	return &ConnectorClient{
		restClient: c.restClient,
		ns: namespace,
	}
}

type ConnectorClient struct {
	restClient rest.Interface
	ns         string
}

type ConnectorInterface interface {
	Get(name string, options metav1.GetOptions) (*client.KafkaConnector, error)
}

func (c *ConnectorClient) Get(name string, opts metav1.GetOptions) (*client.KafkaConnector, error) {
	result := client.KafkaConnector{}
	err := c.restClient.
		Get().
		Namespace(c.ns).
		Resource("kafkaconnectors").
		Name(name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(&result)

	return &result, err
}
