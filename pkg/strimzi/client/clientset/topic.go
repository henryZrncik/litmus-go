package clientset
//
//import (
//	"github.com/litmuschaos/litmus-go/pkg/strimzi/client"
//	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
//	"k8s.io/apimachinery/pkg/types"
//	"k8s.io/client-go/kubernetes/scheme"
//	"k8s.io/client-go/rest"
//)
//
//func (c *ExampleV1Alpha1Client) KafkaTopic(namespace string) ProjectInterface2 {
//	return &projectClient2{
//		restClient: c.restClient,
//		ns: namespace,
//	}
//}
//
//
//type ProjectInterface2 interface {
//	Get(name string, options metav1.GetOptions) (*client.KafkaTopic, error)
//	Patch(name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (*client.KafkaTopic, error)
//}
//
//type projectClient2 struct {
//	restClient rest.Interface
//	ns         string
//}
//
//
//func (c *projectClient2) Get(name string, opts metav1.GetOptions) (*client.KafkaTopic, error) {
//	result := client.KafkaTopic{}
//	err := c.restClient.
//		Get().
//		Namespace(c.ns).
//		Resource("kafkatopics").
//		Name(name).
//		VersionedParams(&opts, scheme.ParameterCodec).
//		Do().
//		Into(&result)
//
//	return &result, err
//}
//
//
//func (c *projectClient2) Patch(name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (*client.KafkaTopic, error) {
//	result := client.KafkaTopic{}
//	err := c.restClient.Patch(pt).
//		Namespace(c.ns).
//		Resource("kafkatopics").
//		Name(name).
//		SubResource(subresources...).
//		VersionedParams(&opts, scheme.ParameterCodec).
//		Body(data).
//		Do().
//		Into(&result)
//	return &result, err
//}