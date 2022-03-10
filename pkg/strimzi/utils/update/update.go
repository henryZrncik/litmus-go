package update

import (
	"encoding/json"
	"github.com/litmuschaos/litmus-go/pkg/clients"
	"github.com/litmuschaos/litmus-go/pkg/log"
	"github.com/litmuschaos/litmus-go/pkg/strimzi/client"
	experimentClientSet "github.com/litmuschaos/litmus-go/pkg/strimzi/client/clientset"
	experimentTypes "github.com/litmuschaos/litmus-go/pkg/strimzi/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8types "k8s.io/apimachinery/pkg/types"
)

//  patchStringValue specifies a patch operation for a uint32.
type patchObjectValueLast struct {
	Op    string `json:"op"`
	Path  string            `json:"path"`
	Value []client.Listener `json:"value"`
}


func InitStrimziClient(experimentsDetails *experimentTypes.ExperimentDetails, clients clients.ClientSets,  ) error{
	clientSet, err := experimentClientSet.NewForConfig(clients.KubeConfig)
	experimentsDetails.Strimzi.Client  = clientSet
	return err
}


func Update(experimentsDetails *experimentTypes.ExperimentDetails, clients clients.ClientSets) ([]k8types.UID, error){
	// Warning about conditions on cluster. If error comes, propagate it.
	err := verifyCurrentStrimziKafkaCRState(experimentsDetails)
	if err != nil {
		return nil, err
	}

	// Get UID of all initialKafkaPodsUIDs marked by kafka pod label and namespace,
	initialKafkaPodsUIDs, err := getListOfCurrentPods(experimentsDetails,clients)
	if err != nil {
		return nil, err
	}

	log.Info("[Info]: Initialization of update on Strimzi Cluster")
	// Get suitable change in list of listeners that can cast UPDATE
	patchListeners, err := proposePatch(experimentsDetails)
	if err != nil {
		return nil, err
	}

	// Apply this Patch
	err = patchStrimziKafka(experimentsDetails, patchListeners)
	if err != nil {
		return nil, err
	}

	return initialKafkaPodsUIDs, nil
}

func verifyCurrentStrimziKafkaCRState(exp *experimentTypes.ExperimentDetails) error{
	var isClusterInUnexpectedState bool = false
	log.Infof("[Info]: Getting information about strimzi kafka cluster %s, in namespace %v", exp.Strimzi.StrimziKafkaClusterName, exp.App.Namespace)
	projects, err := exp.Strimzi.Client.Kafka(exp.App.Namespace).Get(exp.Strimzi.StrimziKafkaClusterName, metav1.GetOptions{})
	if err != nil {
		log.Errorf("[Error]: Problem using custom Strimzi client: obtaining info about strimzi Kafka CR with name: %s", exp.Strimzi.StrimziKafkaClusterName)
		return err
	}

	// Check that there are no conditions (i.e., problems) with Strimzi Kafka Cluster.
	for _, condition := range projects.Status.Conditions {
		if condition.Message != "" || condition.Reason != "" {
			isClusterInUnexpectedState = true
			log.Warnf("Kafka Cluster: %s, reason: %s, message:  %s", exp.Strimzi.StrimziKafkaClusterName, condition.Reason, condition.Message)
		}
	}

	// Depending on existence of some serious condition warn about it
	if isClusterInUnexpectedState {
		log.Warnf("[Error]: Strimzi Kafka Cluster has condition that may need to be solved")
		return nil
	}
	log.Info("[Info]: Strimzi Kafka CR State is valid")
	return nil
}


// just take UIDs of selected pods or return error
func getListOfCurrentPods(exp *experimentTypes.ExperimentDetails, clients clients.ClientSets) ([]k8types.UID, error){
	pods, err := clients.KubeClient.CoreV1().Pods(exp.App.Namespace).List(metav1.ListOptions{LabelSelector: exp.Kafka.Label})
	if err != nil {
			return nil, err
	}
	var sliceOfUIDs []k8types.UID
	for _, pod := range pods.Items {
		sliceOfUIDs = append(sliceOfUIDs, pod.UID)
	}
	return sliceOfUIDs, nil
}

//proposePatch tells whether to create/delete listener and what listener exactly.
func proposePatch(exp *experimentTypes.ExperimentDetails) ([]client.Listener,error)  {
	desiredListenerPort := exp.Strimzi.InternalListenerPortNumber
	desiredListenerName := exp.Strimzi.InternalListenerName

	// Get currently present listeners
	kafka, err := exp.Strimzi.Client.Kafka(exp.App.Namespace).Get(exp.Strimzi.StrimziKafkaClusterName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	listeners := kafka.Spec.Spec.Listeners

	// if listener with given name exist REMOVE it, otherwise find first free port and append it.
	if listenerExistsByName(listeners, desiredListenerName){
		// REMOVE listener
		log.Infof("[Info]: Patch by removal of listener with name %s, and port %d",desiredListenerName, desiredListenerPort)
		listeners = setUpListeners(listeners, desiredListenerName, desiredListenerPort, false)
	}else{
		// listener with name does not exist
		// first try provided/default port
		if !isPortTaken(listeners, desiredListenerPort){
			log.Infof("[Info]: Patch by creation of listener with name %s, and original port %d",desiredListenerName, desiredListenerPort)
			listeners = setUpListeners(listeners, desiredListenerName, desiredListenerPort, true)
		}else{
			// find first free port from range (10 000; 11 000>
			for portNumber := 10_001; portNumber < 11_000; portNumber++ {
				if !isPortTaken(listeners, portNumber){
					log.Infof("[Info]: Patch by creation of listener with name %s, and newly assigned port %d",desiredListenerName, portNumber)
					// CREATE listener
					listeners = setUpListeners(listeners, desiredListenerName, desiredListenerPort, true)
					break
				}
			}
		}
	}
	return listeners, nil
}



func listenerExistsByName(listeners []client.Listener, name string) bool{
	for _, listener := range listeners {
		if listener.Name == name {
			log.Infof("[Info]: listener with name %s exist",name)
			return true
		}
	}
	return false 
}

func isPortTaken(listeners []client.Listener, port int ) bool{
	for _, listener := range listeners {
		if listener.Port == port  {
			log.Infof("[Info]: port %d, already taken",port)
			return true
		}
	}
	return false
}


func setUpListeners(listeners []client.Listener, name string, port int,  appendListener bool  ) []client.Listener{
	var resultListenrs []client.Listener
	// if append option than new listener is appended to those existing
	
	if appendListener {
		var chaosListener client.Listener = client.Listener{
			Name: name,
			Port: port,
			TLS:  true,
			Type: "internal",

		}
		resultListenrs = append(listeners, chaosListener)
	}else{
		// listener is removed. (By not being appended to new slice of listeners)
		for _, listener := range listeners {
			if listener.Name != name {
				resultListenrs = append(resultListenrs, listener)
			}
		}
	}
	return resultListenrs
}



func patchStrimziKafka(exp *experimentTypes.ExperimentDetails, newListeners []client.Listener ) error{
	clientSet := exp.Strimzi.Client

	payload := []patchObjectValueLast{{
		Op:    "replace",
		Path:  "/spec/kafka/listeners",
		Value: newListeners,
	}}

	payloadBytes, _ := json.Marshal(payload)
	_,err := clientSet.Kafka(exp.App.Namespace).Patch( exp.Strimzi.StrimziKafkaClusterName, k8types.JSONPatchType, payloadBytes,metav1.PatchOptions{
		FieldManager: "application/apply-patch",
	},"" )
	if err != nil {
		log.Errorf("[Error]: Problem using mu Strimzi client (Patch)")
		return err
	}
	return err
}
