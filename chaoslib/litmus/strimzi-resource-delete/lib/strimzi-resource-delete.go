package lib

import (
	"github.com/litmuschaos/litmus-go/pkg/clients"
	"github.com/litmuschaos/litmus-go/pkg/events"
	"github.com/litmuschaos/litmus-go/pkg/log"
	"github.com/litmuschaos/litmus-go/pkg/probe"
	"github.com/litmuschaos/litmus-go/pkg/strimzi/environment"
	experimentTypes "github.com/litmuschaos/litmus-go/pkg/strimzi/types"
	strimzi_utils "github.com/litmuschaos/litmus-go/pkg/strimzi/utils/resources"
	"github.com/litmuschaos/litmus-go/pkg/types"
	//"github.com/litmuschaos/litmus-go/pkg/utils/annotation"
	"github.com/litmuschaos/litmus-go/pkg/utils/common"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	//v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//"strconv"
	"strings"
	"time"
)

//PrepareResourceDelete contains the preparation steps before chaos injection
func PrepareResourceDelete(experimentsDetails *experimentTypes.ExperimentDetails, clients clients.ClientSets, resultDetails *types.ResultDetails, eventsDetails *types.EventDetails, chaosDetails *types.ChaosDetails) error {

	//Waiting for the ramp time before chaos injection
	if experimentsDetails.Control.RampTime != 0 {
		log.Infof("[Ramp]: Waiting for the %vs ramp time before injecting chaos", experimentsDetails.Control.RampTime)
		common.WaitForDuration(experimentsDetails.Control.RampTime)
	}

	switch strings.ToLower(experimentsDetails.Control.Sequence) {
	case "serial":
		log.Infof("[Chaos]: Injecting chaos in serial mode")
		if err := injectChaosInSerialMode(experimentsDetails, clients, chaosDetails, eventsDetails, resultDetails); err != nil {
			return err
		}
	case "parallel":
		log.Infof("[Chaos]: Injecting chaos in parallel mode")
		if err := injectChaosInParallelMode(experimentsDetails, clients, chaosDetails, eventsDetails, resultDetails); err != nil {
			return err
		}
	default:
		return errors.Errorf("%v sequence is not supported", experimentsDetails.Control.Sequence)
	}

	//Waiting for the ramp time after chaos injection
	if experimentsDetails.Control.RampTime != 0 {
		log.Infof("[Ramp]: Waiting for the %vs ramp time after injecting chaos", experimentsDetails.Control.RampTime)
		common.WaitForDuration(experimentsDetails.Control.RampTime)
	}
	return nil
}

// injectChaosInSerialMode delete the kafka broker pods in serial mode(one by one)
func injectChaosInSerialMode(experimentsDetails *experimentTypes.ExperimentDetails, clients clients.ClientSets, chaosDetails *types.ChaosDetails, eventsDetails *types.EventDetails, resultDetails *types.ResultDetails) error {
	// run the probes during chaos
	if len(resultDetails.ProbeDetails) != 0 {
		if err := probe.RunProbes(chaosDetails, clients, resultDetails, "DuringChaos", eventsDetails); err != nil {
			return err
		}
	}

	//ChaosStartTimeStamp contains the start timestamp, when the chaos injection begin
	ChaosStartTimeStamp := time.Now()
	duration := int(time.Since(ChaosStartTimeStamp).Seconds())

	for duration < experimentsDetails.Control.ChaosDuration {

		if len(experimentsDetails.Resources.Resources) == 0{
			return errors.Errorf("please provide at least one resource (i.e. name of secret, configuration map or service)")
		}

		if experimentsDetails.Control.EngineName != "" {
			msg := "Injecting " + experimentsDetails.Control.ExperimentName + " chaos on application resources"
			types.SetEngineEventAttributes(eventsDetails, types.ChaosInject, msg, "Normal", chaosDetails)
			events.GenerateEvents(eventsDetails, clients, chaosDetails, "ChaosEngine")
		}

		log.InfoWithValues("[Info]: following resources marked for deletion", logrus.Fields{
			//"Resource type": resource.Type,
			"Resource": experimentsDetails.Resources.Resources})

		for _, elem := range experimentsDetails.Resources.Resources {
			log.InfoWithValues("[Chaos]: If exist, killing the following resource", logrus.Fields{
				"Resource": elem})

			if err := strimzi_utils.DeleteResource(experimentsDetails.App.Namespace, elem, experimentsDetails.Control.Force, clients); err != nil {
				return err
			}
			// obtain duration (either fixed, or one in specified by randomness interval)
			chaosIntervalDuration, err := environment.GetChaosIntervalDuration(*experimentsDetails, chaosDetails.Randomness)
			if err != nil {
				return err
			}
			log.InfoWithValues("[Wait]: Waiting for single chaos interval", logrus.Fields{
				"Chaos Interval Duration": chaosIntervalDuration})

			// wait for specified duration
			strimzi_utils.WaitForChaosIntervalDurationResources(*experimentsDetails,clients, chaosIntervalDuration)

			//Verify the existence of given resource after specified interval
			log.Info("[Status]: Verification for the recreation of application resources")
			if err = strimzi_utils.HealthCheckResources(experimentsDetails.App.Namespace,[]experimentTypes.KubernetesResource{elem},0,1,clients); err != nil {
				return err
			}
		}




		duration = int(time.Since(ChaosStartTimeStamp).Seconds())
	}

	log.Infof("[Completion]: %v chaos is done", experimentsDetails.Control.ExperimentName)

	return nil
}

// injectChaosInParallelMode delete the kafka broker pods in parallel mode (all at once)
func injectChaosInParallelMode(experimentsDetails *experimentTypes.ExperimentDetails, clients clients.ClientSets, chaosDetails *types.ChaosDetails, eventsDetails *types.EventDetails, resultDetails *types.ResultDetails) error {

	// run the probes during chaos
	if len(resultDetails.ProbeDetails) != 0 {
		if err := probe.RunProbes(chaosDetails, clients, resultDetails, "DuringChaos", eventsDetails); err != nil {
			return err
		}
	}

	//ChaosStartTimeStamp contains the start timestamp, when the chaos injection begin
	ChaosStartTimeStamp := time.Now()
	duration := int(time.Since(ChaosStartTimeStamp).Seconds())

	for duration < experimentsDetails.Control.ChaosDuration {

		if experimentsDetails.Control.EngineName != "" {
			msg := "Injecting " + experimentsDetails.Control.ExperimentName + " chaos on application resources"
			types.SetEngineEventAttributes(eventsDetails, types.ChaosInject, msg, "Normal", chaosDetails)
			events.GenerateEvents(eventsDetails, clients, chaosDetails, "ChaosEngine")
		}

		log.InfoWithValues("[Chaos]: If exist, killing the following resource", logrus.Fields{
			"Resource": experimentsDetails.Resources.Resources})

		if err := strimzi_utils.DeleteListOfResources(experimentsDetails.App.Namespace, experimentsDetails.Resources.Resources, experimentsDetails.Control.Force, clients); err != nil {
			return err
		}

		chaosIntervalDuration, err := environment.GetChaosIntervalDuration(*experimentsDetails, chaosDetails.Randomness)
		if err != nil {
			return err
		}
		log.InfoWithValues("[Wait]: Waiting for single chaos interval", logrus.Fields{
			//"Resource type": resource.Type,
			"Chaos Interval Duration": chaosIntervalDuration})

		// wait for specified duration
		strimzi_utils.WaitForChaosIntervalDurationResources(*experimentsDetails,clients, chaosIntervalDuration)

		//Verify the existence of resources after
		log.Info("[Status]: Verification for the recreation of application resources")
		if err = strimzi_utils.HealthCheckResources(experimentsDetails.App.Namespace,experimentsDetails.Resources.Resources,0,1,clients); err != nil {
			return err
		}

		duration = int(time.Since(ChaosStartTimeStamp).Seconds())
	}

	log.Infof("[Completion]: %v chaos is done", experimentsDetails.Control.ExperimentName)

	return nil
}
