package clientset

import (
	"github.com/litmuschaos/litmus-go/pkg/clients"
	"github.com/litmuschaos/litmus-go/pkg/strimzi/client"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

type StrimziV1AlphaClient struct {
	restClient rest.Interface
}

func NewForConfig(c *rest.Config) (*StrimziV1AlphaClient, error) {
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

	return &StrimziV1AlphaClient{restClient: client}, nil
}

func InitStrimziClient(clients clients.ClientSets) (*StrimziV1AlphaClient, error) {
	clientSet, err := NewForConfig(clients.KubeConfig)
	return clientSet, err
}


