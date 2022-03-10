package experiment

import (
	"github.com/litmuschaos/chaos-operator/pkg/apis/litmuschaos/v1alpha1"
	litmusLIB "github.com/litmuschaos/litmus-go/chaoslib/litmus/strimzi-cluster-update/lib"
	"github.com/litmuschaos/litmus-go/pkg/clients"
	"github.com/litmuschaos/litmus-go/pkg/events"
	"github.com/litmuschaos/litmus-go/pkg/log"
	"github.com/litmuschaos/litmus-go/pkg/probe"
	"github.com/litmuschaos/litmus-go/pkg/result"
	"github.com/litmuschaos/litmus-go/pkg/status"
	"github.com/litmuschaos/litmus-go/pkg/utils/common"
	"github.com/sirupsen/logrus"

	experimentEnv "github.com/litmuschaos/litmus-go/pkg/strimzi/environment"
	experimentTypes "github.com/litmuschaos/litmus-go/pkg/strimzi/types"

	strimzikafkaresource "github.com/litmuschaos/litmus-go/pkg/strimzi/utils/update"
	"github.com/litmuschaos/litmus-go/pkg/types"
	"os"
)





func PodDelete(clients clients.ClientSets) {
	experimentsDetails := experimentTypes.ExperimentDetails{}
	resultDetails := types.ResultDetails{}
	eventsDetails := types.EventDetails{}
	chaosDetails := types.ChaosDetails{}



	log.Infof("[PreReq]: Starting Strimzi Kafka Cluster Update experiment")

	//Fetching all the ENV passed from the runner pod
	log.Infof("[PreReq]: Getting the ENV for the %v experiment", os.Getenv("EXPERIMENT_NAME"))
	experimentEnv.GetENV(&experimentsDetails)


	// Initialize the chaos attributes
	types.InitialiseChaosVariables(&chaosDetails)

	// Initialize Chaos Result Parameters
	types.SetResultAttributes(&resultDetails, chaosDetails)

	if experimentsDetails.Control.EngineName != "" {
		// Initialize the probe details. Bail out upon error, as we haven't entered exp business logic yet
		if err := probe.InitializeProbesInChaosResultDetails(&chaosDetails, clients, &resultDetails); err != nil {
			log.Errorf("Unable to initialize the probes, err: %v", err)
			return
		}
	}

	//Updating the chaos result in the beginning of experiment
	log.Infof("[PreReq]: Updating the chaos result of %v experiment (SOT)", experimentsDetails.Control.ExperimentName)
	if err := result.ChaosResult(&chaosDetails, clients, &resultDetails, "SOT"); err != nil {
		log.Errorf("Unable to Create the Chaos Result, err: %v", err)
		failStep := "[pre-chaos]: Failed to update the chaos result of pod-delete experiment (SOT), err: " + err.Error()
		result.RecordAfterFailure(&chaosDetails, &resultDetails, failStep, clients, &eventsDetails)
		return
	}

	// Set the chaos result uid
	result.SetResultUID(&resultDetails, clients, &chaosDetails)

	// generating the event in chaosresult to marked the verdict as awaited
	msg := "experiment: " + experimentsDetails.Control.ExperimentName + ", Result: Awaited"
	types.SetResultEventAttributes(&eventsDetails, types.AwaitedVerdict, msg, "Normal", &resultDetails)
	events.GenerateEvents(&eventsDetails, clients, &chaosDetails, "ChaosResult")

	log.InfoWithValues("[Info]: The application information is as follows ", logrus.Fields{
		"Application Namespace":      experimentsDetails.App.Namespace,
		"Chaos Duration":             experimentsDetails.Control.ChaosDuration,
	})

	// Calling AbortWatcher go routine, it will continuously watch for the abort signal and generate the required events and result
	go common.AbortWatcher(experimentsDetails.Control.ExperimentName, clients, &resultDetails, &chaosDetails, &eventsDetails)


	// Custom Pre chaos step 1:
	// set up strimzi specific k8 client
	log.Infof("[PreReq]: Set up strimzi k8 client")
	err := strimzikafkaresource.InitStrimziClient(&experimentsDetails, clients)
	if err != nil {
		log.Errorf("Unable to create Strimzi client, err: %v", err)
		failStep := "[pre-chaos]: Failed to create Strimzi client, err: " + err.Error()
		result.RecordAfterFailure(&chaosDetails, &resultDetails, failStep, clients, &eventsDetails)
		return
	}

	// PRE-CHAOS APPLICATION STATUS CHECK
	// by checking health status of all kafka pods.
	if chaosDetails.DefaultAppHealthCheck {
		log.Info("[Status]: Verify that all the Strimzi kafka pods are running")
		if err := status.CheckApplicationStatus(experimentsDetails.App.Namespace, experimentsDetails.Kafka.Label, experimentsDetails.Control.Timeout, experimentsDetails.Control.Delay, clients); err != nil {
			log.Errorf("Cluster health check failed, err: %v", err)
			failStep := "[pre-chaos]: Failed to verify that the Kafka cluster is healthy, err: " + err.Error()
			types.SetEngineEventAttributes(&eventsDetails, types.PreChaosCheck, "AUT: Not Running", "Warning", &chaosDetails)
			events.GenerateEvents(&eventsDetails, clients, &chaosDetails, "ChaosEngine")
			result.RecordAfterFailure(&chaosDetails, &resultDetails, failStep, clients, &eventsDetails)
			return
		}
	}


	if experimentsDetails.Control.EngineName != "" {
		// marking AUT as running, as we already checked the status of application under test
		msg := common.GetStatusMessage(chaosDetails.DefaultAppHealthCheck, "AUT: Running", "")

		// run the probes in the pre-chaos check
		if len(resultDetails.ProbeDetails) != 0 {

			if err := probe.RunProbes(&chaosDetails, clients, &resultDetails, "PreChaos", &eventsDetails); err != nil {
				log.Errorf("Probe Failed, err: %v", err)
				failStep := "[pre-chaos]: Failed while running probes, err: " + err.Error()
				msg = common.GetStatusMessage(chaosDetails.DefaultAppHealthCheck, "AUT: Running", "Unsuccessful")
				types.SetEngineEventAttributes(&eventsDetails, types.PreChaosCheck, msg, "Warning", &chaosDetails)
				events.GenerateEvents(&eventsDetails, clients, &chaosDetails, "ChaosEngine")
				result.RecordAfterFailure(&chaosDetails, &resultDetails, failStep, clients, &eventsDetails)
				return
			}
			common.GetStatusMessage(chaosDetails.DefaultAppHealthCheck, "AUT: Running", "Successful")
		}
		// generating the events for the pre-chaos check
		types.SetEngineEventAttributes(&eventsDetails, types.PreChaosCheck, msg, "Normal", &chaosDetails)
		events.GenerateEvents(&eventsDetails, clients, &chaosDetails, "ChaosEngine")
	}

	// TODO
	// PRE-CHAOS LIVENESS CHECK

	// TODO including dosplay
	//kafka.DisplayKafkaBroker(&experimentsDetails)

	// TODO import lib
	// Including the litmus lib for pod-delete
	switch experimentsDetails.Control.ChaosLib {
	case "litmus":
		if err := litmusLIB.PrepareChaosInjection(&experimentsDetails, clients, &resultDetails, &eventsDetails, &chaosDetails); err != nil {
			log.Errorf("Chaos injection failed, err: %v", err)
			failStep := "[chaos]: Failed inside the chaoslib, err: " + err.Error()
			result.RecordAfterFailure(&chaosDetails, &resultDetails, failStep, clients, &eventsDetails)
			return
		}
	default:
		log.Error("[Invalid]: Please Provide the correct LIB")
		failStep := "[chaos]: no match found for specified lib"
		result.RecordAfterFailure(&chaosDetails, &resultDetails, failStep, clients, &eventsDetails)
		return
	}


	log.Infof("[Confirmation]: %v chaos has been injected successfully", experimentsDetails.Control.ExperimentName)
	resultDetails.Verdict = v1alpha1.ResultVerdictPassed













































	// NEW




	//{ "op": "replace", "path": "/baz", "value": "boo" },
	//{ "op": "add", "path": "/hello", "value": ["world"] },
	//{ "op": "remove", "path": "/foo" }

	//var a []utils.Listener = projects.Spec.Spec.Listeners
	//a = append(a, utils.Listener{
	//	"bobino",
	//	9096,
	//	true,
	//	"internal",
	//
	//})



	//// Patch part
	//payload := []patchObjectValue{{
	//	Op:    "add",
	//	Path:  "/spec/kafka/listeners/0",
	//	Value: utils.Listener{
	//		"bobino",
	//		9999,
	//		true,
	//		"internal",
	//	},
	//}}

	//payload := []patchUInt32Value{{
	//	Op:    "replace",
	//	Path:  "/spec/kafka/replicas",
	//	Value: 4,
	//}}
	//
	//payload := []patchUInt32Value{{
	//	Op:    "replace",
	//	Path:  "/spec/kafka/replicas",
	//	Value: 9999,
	//}}





	// NEW END

	// OLD

	// use client for fun
	//clientSet, err := experimentClientSetE.NewForConfig(clients.KubeConfig)
	//if err != nil {
	//	log.Errorf("[Error]: Unable to create Strimzi Kubernetes client")
	//	panic(err)
	//}
	//
	//projects, err := clientSet.Projects("default").Get("example-project-2", metav1.GetOptions{})
	//if err != nil {
	//	log.Errorf("[Error]: Problem using mu Strimzi client")
	//	panic(err)
	//}
	//fmt.Printf("projects found: %+v\n", projects)
	//
	//// Patch part
	//payload := []patchUInt32Value{{
	//	Op:    "replace",
	//	Path:  "/spec/replicas",
	//	Value: 160,
	//}}
	//
	//payloadBytes, _ := json.Marshal(payload)
	//
	//_,err = clientSet.Projects("default").Patch("example-project-2", fucker.JSONPatchType, payloadBytes,metav1.PatchOptions{
	//	FieldManager: "application/apply-patch",
	//},"" )
	//if err != nil {
	//	log.Errorf("[Error]: Problem using mu Strimzi client")
	//	panic(err)
	//}


	// OLD end













	//projects.Spec.Replicas = 15

	//projects, err = clientSet.Projects("default").Update(projects, metav1.UpdateOptions{})
	//if err != nil {
	//	log.Errorf("[Error]: Problem using mu Strimzi client")
	//	panic(err)
	//}

	//fmt.Printf("projects found: %+v\n", projects)

	//
	//result := experimentClient.ProjectList{}
	//err = client.
	//	Get().
	//	Resource("projects").
	//	Do().
	//	Into(&result)
	//
	//if err != nil {
	//	log.Errorf("[Error]: Problem using mu Strimzi client")
	//	panic(err)
	//}

	//return nil
}