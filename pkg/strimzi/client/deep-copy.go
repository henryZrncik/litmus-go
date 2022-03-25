package client

import (
	"github.com/litmuschaos/litmus-go/pkg/log"
	"k8s.io/apimachinery/pkg/runtime"
)

func (in *Kafka) DeepCopyInto(out *Kafka) {
	out = in
}

// DeepCopyObject returns a generically typed copy of an object
func (in *Kafka) DeepCopyObject() runtime.Object {
	out := Kafka{}
	in.DeepCopyInto(&out)
	return &out
}

// DeepCopyObject returns a generically typed copy of an object
func (in *KafkaList) DeepCopyObject() runtime.Object {
	out := KafkaList{}
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta

	if in.Items != nil {
		out.Items = make([]Kafka, len(in.Items))
		for i := range in.Items {
			in.Items[i].DeepCopyInto(&out.Items[i])
		}
	}

	return &out
}

//Kafka Topic

func (in *KafkaTopic) DeepCopyInto(out *KafkaTopic) {
	out = in
}

// DeepCopyObject returns a generically typed copy of an object
func (in *KafkaTopic) DeepCopyObject() runtime.Object {
	log.Info("!!!!!!!!!!!!!!!!!!!!!!!!!!")
	out := KafkaTopic{}
	in.DeepCopyInto(&out)
	return &out
}

// DeepCopyObject returns a generically typed copy of an object
func (in *KafkaTopicList) DeepCopyObject() runtime.Object {
	log.Info("$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$")
	out := KafkaTopicList{}
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta

	if in.Items != nil {
		out.Items = make([]KafkaTopic, len(in.Items))
		for i := range in.Items {
			in.Items[i].DeepCopyInto(&out.Items[i])
		}
	}

	return &out
}
