package client

import "k8s.io/apimachinery/pkg/runtime"

// DeepCopyInto copies all properties of this object into another object of the
// same type that is provided as a pointer.
func (in *Kafka) DeepCopyInto(out *Kafka) {
	//out.TypeMeta = in.TypeMeta
	//out.ObjectMeta = in.ObjectMeta
	//out.Spec = KafkaSpec{
	//	Replicas: in.Spec.Replicas,
	//	Listeners: in.Spec.Listeners,
	//}
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
