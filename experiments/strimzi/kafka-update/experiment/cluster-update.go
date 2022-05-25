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
	"github.com/litmuschaos/litmus-go/pkg/strimzi/client/clientset"
	experimentEnv "github.com/litmuschaos/litmus-go/pkg/strimzi/environment"
	strimziLiveness "github.com/litmuschaos/litmus-go/pkg/strimzi/livenessstream"
	experimentTypes "github.com/litmuschaos/litmus-go/pkg/strimzi/types"
	"github.com/litmuschaos/litmus-go/pkg/types"
	"github.com/litmuschaos/litmus-go/pkg/utils/common"
	"github.com/sirupsen/logrus"
	"os"
	"strings"
)


func Update(clients clients.ClientSets) {
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
		"Application Namespace": experimentsDetails.App.Namespace,
		"Chaos Duration":        experimentsDetails.Control.ChaosDuration,
		"Liveness stream": experimentsDetails.App.LivenessStream,
	})

	// Calling AbortWatcher go routine, it will continuously watch for the abort signal and generate the required events and result
	go common.AbortWatcher(experimentsDetails.Control.ExperimentName, clients, &resultDetails, &chaosDetails, &eventsDetails)


	// Pre chaos step: set up strimzi specific k8 client
	log.Infof("[PreReq]: Set up strimzi k8 client")
	strimziClient, err := clientset.InitStrimziClient(clients)
	if err != nil {
		log.Errorf("Unable to create Strimzi client, err: %v", err)
		failStep := "[pre-chaos]: Failed to create Strimzi client, err: " + err.Error()
		result.RecordAfterFailure(&chaosDetails, &resultDetails, failStep, clients, &eventsDetails)
		return
	}
	experimentsDetails.Strimzi.Client = strimziClient


	// PRE-CHAOS APPLICATION STATUS CHECK
	// by checking health status of all kafka pods.
	if chaosDetails.DefaultAppHealthCheck {
		log.Info("[Status]: Verify that all the Strimzi kafka pods are running")
		if err := status.CheckApplicationStatus(experimentsDetails.App.Namespace, chaosDetails.AppDetail.Label, experimentsDetails.Control.Timeout, experimentsDetails.Control.Delay, clients); err != nil {
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

	// PRE-CHAOS STRIMZI KAFKA APPLICATION LIVENESS CHECK
	switch strings.ToLower(experimentsDetails.App.LivenessStream) {
	case "enable":
		// defer delete liveness jobs
		if strings.ToLower(experimentsDetails.App.LivenessStreamJobsCleanup) == "enable" {
			log.Infof("[Liveness-Cleanup]: Defer clean up of jobs")
			defer strimziLiveness.JobsCleanup(&experimentsDetails,clients)
		}
		// defer Delete liveness topic
		if strings.ToLower(experimentsDetails.App.LivenessStreamTopicCleanup) == "enable" {
			log.Infof("[Liveness-Cleanup]: Defer clean up of topic")
			defer strimziLiveness.TopicCleanup(&experimentsDetails)
		}
		// apply liveness stream
		err = strimziLiveness.LivenessStream(&experimentsDetails, clients)
		if err != nil {
			log.Errorf("Liveness check failed, err: %v", err)
			failStep := "[pre-chaos]: Failed to verify liveness check, err: " + err.Error()
			result.RecordAfterFailure(&chaosDetails, &resultDetails, failStep, clients, &eventsDetails)
			return
		}
		log.Info("[Liveness]: The Liveness stream creation completed")
	}


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

	//POST-CHAOS APPLICATION STATUS CHECK
	if chaosDetails.DefaultAppHealthCheck {
		log.Info("[Status]: Verify that the AUT (Application Under Test) is running (post-chaos)")

		if err := status.CheckApplicationStatus(experimentsDetails.App.Namespace, chaosDetails.AppDetail.Label, experimentsDetails.Control.Timeout, experimentsDetails.Control.Delay, clients); err != nil {
			log.Errorf("Application status check failed, err: %v", err)
			failStep := "[pre-chaos]: Failed to verify that the AUT (Application Under Test, err: " + err.Error()
			types.SetEngineEventAttributes(&eventsDetails, types.PreChaosCheck, "AUT: Not Running", "Warning", &chaosDetails)
			events.GenerateEvents(&eventsDetails, clients, &chaosDetails, "ChaosEngine")
			result.RecordAfterFailure(&chaosDetails, &resultDetails, failStep, clients, &eventsDetails)
			return
		}

	}

	if experimentsDetails.Control.EngineName != "" {
		// marking AUT as running, as we already checked the status of application under test
		msg := common.GetStatusMessage(chaosDetails.DefaultAppHealthCheck, "AUT: Running", "")

		// run the probes in the post-chaos check
		if len(resultDetails.ProbeDetails) != 0 {
			if err := probe.RunProbes(&chaosDetails, clients, &resultDetails, "PostChaos", &eventsDetails); err != nil {
				log.Errorf("Probes Failed, err: %v", err)
				failStep := "[post-chaos]: Failed while running probes, err: " + err.Error()
				msg = common.GetStatusMessage(chaosDetails.DefaultAppHealthCheck, "AUT: Running", "Unsuccessful")
				types.SetEngineEventAttributes(&eventsDetails, types.PostChaosCheck, msg, "Warning", &chaosDetails)
				events.GenerateEvents(&eventsDetails, clients, &chaosDetails, "ChaosEngine")
				result.RecordAfterFailure(&chaosDetails, &resultDetails, failStep, clients, &eventsDetails)
				return
			}
			common.GetStatusMessage(chaosDetails.DefaultAppHealthCheck, "AUT: Running", "Successful")
		}

		// generating post chaos event
		types.SetEngineEventAttributes(&eventsDetails, types.PostChaosCheck, msg, "Normal", &chaosDetails)
		events.GenerateEvents(&eventsDetails, clients, &chaosDetails, "ChaosEngine")
	}

	// Liveness Status Check (post-chaos) and cleanup
	switch strings.ToLower(experimentsDetails.App.LivenessStream) {
	case "enable":
		log.Info("[Status]: Verify that the Kafka liveness jobs finished successfully (post-chaos)")
		if err := strimziLiveness.VerifyLivenessStream(&experimentsDetails, clients); err != nil {
			log.Errorf("Application liveness status check failed, err: %v", err)
			failStep := "[post-chaos]: Failed to verify that the liveness jobs finished successfully, err: " + err.Error()
			result.RecordAfterFailure(&chaosDetails, &resultDetails, failStep, clients, &eventsDetails)
			return
		}
	}


	//Updating the chaosResult in the end of experiment
	log.Infof("[The End]: Updating the chaos result of %v experiment (EOT)", experimentsDetails.Control.ExperimentName)
	if err := result.ChaosResult(&chaosDetails, clients, &resultDetails, "EOT"); err != nil {
		log.Errorf("Unable to Update the Chaos Result, err: %v", err)
		return
	}

	// generating the event in chaosresult to marked the verdict as pass/fail
	msg = "experiment: " + experimentsDetails.Control.ExperimentName + ", Result: " + string(resultDetails.Verdict)
	reason := types.PassVerdict
	eventType := "Normal"
	if resultDetails.Verdict != "Pass" {
		reason = types.FailVerdict
		eventType = "Warning"
	}
	types.SetResultEventAttributes(&eventsDetails, reason, msg, eventType, &resultDetails)
	events.GenerateEvents(&eventsDetails, clients, &chaosDetails, "ChaosResult")

	if experimentsDetails.Control.EngineName != "" {
		msg := experimentsDetails.Control.ExperimentName + " experiment has been " + string(resultDetails.Verdict) + "ed"
		types.SetEngineEventAttributes(&eventsDetails, types.Summary, msg, "Normal", &chaosDetails)
		events.GenerateEvents(&eventsDetails, clients, &chaosDetails, "ChaosEngine")
	}

}