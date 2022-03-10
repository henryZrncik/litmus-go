package clientset

import (
	"github.com/litmuschaos/litmus-go/pkg/strimzi/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

type ExampleV1Alpha1Interface interface {
	Kafka(namespace string) ProjectInterface
}

type ExampleV1Alpha1Client struct {
	restClient rest.Interface
}

func NewForConfig(c *rest.Config) (*ExampleV1Alpha1Client, error) {
	config := *c
	client.AddToScheme(scheme.Scheme)
	config.ContentConfig.GroupVersion = &schema.GroupVersion{Group: client.GroupName, Version: client.GroupVersion}
	config.APIPath = "/apis"
	config.NegotiatedSerializer = serializer.NewCodecFactory(scheme.Scheme)
	config.UserAgent = rest.DefaultKubernetesUserAgent()

	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}

	return &ExampleV1Alpha1Client{restClient: client}, nil
}

func (c *ExampleV1Alpha1Client) Kafka(namespace string) ProjectInterface {
	return &projectClient{
		restClient: c.restClient,
		ns: namespace,
	}
}


type ProjectInterface interface {
	Get(name string, options metav1.GetOptions) (*client.Kafka, error)
	Patch(name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (*client.Kafka, error)
}

type projectClient struct {
	restClient rest.Interface
	ns         string
}


func (c *projectClient) Get(name string, opts metav1.GetOptions) (*client.Kafka, error) {
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





//func (c *projectClient) Create(project *clientUtils.Project) (*clientUtils.Project, error) {
//	result := clientUtils.Project{}
//	err := c.restClient.
//		Post().
//		Namespace(c.ns).
//		Resource("projects").
//		Body(project).
//		Do().
//		Into(&result)
//	return &result, err
//}

//func (c *projectClient) Watch(opts metav1.ListOptions) (watch.Interface, error) {
//	opts.Watch = true
//	return c.restClient.
//		Get().
//		Namespace(c.ns).
//		Resource("projects").
//		VersionedParams(&opts, scheme.ParameterCodec).
//		Watch()
//}
//
//func (c *projectClient) Update( project *clientUtils.Project, opts metav1.UpdateOptions) (*clientUtils.Project, error) {
//	result := clientUtils.Project{}
//	err := c.restClient.Put().
//		Namespace(c.ns).
//		Resource("projects").
//		Name(project.Name).
//		//VersionedParams(&opts, scheme.ParameterCodec).
//		Body(project).
//		Do().
//		Into(&result)
//	return  &result, err
//}
//
func (c *projectClient) Patch(name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (*client.Kafka, error) {
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