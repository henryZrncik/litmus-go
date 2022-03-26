package clientset

import (
	"github.com/litmuschaos/litmus-go/pkg/strimzi/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

func (c *StrimziV1AlphaClient) KafkaTopic(namespace string) TopicInterface {
	return &TopicClient{
		restClient: c.restClient,
		ns: namespace,
	}
}

type TopicClient struct {
	restClient rest.Interface
	ns         string
}

type TopicInterface interface {
	Get(name string, options metav1.GetOptions) (*client.KafkaTopic, error)
	Patch(name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (*client.KafkaTopic, error)
	Create(topic *client.KafkaTopic, opts metav1.CreateOptions) (result *client.KafkaTopic, err error)
	Delete(name string, opts metav1.DeleteOptions) error
}

// Delete takes name of the pod and deletes it. Returns an error if one occurs.
func (c *TopicClient) Delete(name string, opts metav1.DeleteOptions) error {
	return c.restClient.Delete().
		Namespace(c.ns).
		Resource("kafkatopics").
		Name(name).
		Body(&opts).
		Do().
		Error()
}

func (c *TopicClient) Get(name string, opts metav1.GetOptions) (*client.KafkaTopic, error) {
	result := client.KafkaTopic{}
	err := c.restClient.
		Get().
		Namespace(c.ns).
		Resource("kafkatopics").
		Name(name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(&result)

	return &result, err
}

func (c *TopicClient) Patch(name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (*client.KafkaTopic, error) {
	result := client.KafkaTopic{}
	err := c.restClient.Patch(pt).
		Namespace(c.ns).
		Resource("kafkatopics").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do().
		Into(&result)
	return &result, err
}

// Create takes the representation of a pod and creates it.  Returns the server's representation of the pod, and an error, if there is any.
func (c *TopicClient) Create(topic *client.KafkaTopic, opts metav1.CreateOptions) (result *client.KafkaTopic, err error) {
	result = &client.KafkaTopic{}
	err = c.restClient.Post().
		Namespace(c.ns).
		Resource("kafkatopics").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(topic).
		Do().
		Into(result)
	return
}
