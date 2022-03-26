package clientset

import (
	"github.com/litmuschaos/litmus-go/pkg/strimzi/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

func (c *StrimziV1AlphaClient) Kafka(namespace string) KafkaInterface {
	return &KafkaClient{
		restClient: c.restClient,
		ns: namespace,
	}
}

type KafkaClient struct {
	restClient rest.Interface
	ns         string
}

type KafkaInterface interface {
	Get(name string, options metav1.GetOptions) (*client.Kafka, error)
	Patch(name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (*client.Kafka, error)
}

func (c *KafkaClient) Get(name string, opts metav1.GetOptions) (*client.Kafka, error) {
	result := client.Kafka{}
	err := c.restClient.
		Get().
		Namespace(c.ns).
		Resource("kafkas").
		Name(name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(&result)

	return &result, err
}

func (c *KafkaClient) Patch(name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (*client.Kafka, error) {
	result := client.Kafka{}
	err := c.restClient.Patch(pt).
		Namespace(c.ns).
		Resource("kafkas").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do().
		Into(&result)
	return &result, err
}




