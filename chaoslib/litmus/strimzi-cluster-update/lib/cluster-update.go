package lib

import (
	"github.com/litmuschaos/litmus-go/pkg/clients"
	"github.com/litmuschaos/litmus-go/pkg/events"
	"github.com/litmuschaos/litmus-go/pkg/log"
	"github.com/litmuschaos/litmus-go/pkg/probe"
	"github.com/litmuschaos/litmus-go/pkg/status"
	"github.com/litmuschaos/litmus-go/pkg/strimzi/environment"
	experimentTypes "github.com/litmuschaos/litmus-go/pkg/strimzi/types"
	strimzi_kafka_resource "github.com/litmuschaos/litmus-go/pkg/strimzi/utils/update"
	"github.com/litmuschaos/litmus-go/pkg/types"
	"github.com/litmuschaos/litmus-go/pkg/utils/common"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"time"
)

//PrepareChaosInjection contains the prepration steps before chaos injection
func PrepareChaosInjection(experimentsDetails *experimentTypes.ExperimentDetails, clients clients.ClientSets, resultDetails *types.ResultDetails, eventsDetails *types.EventDetails, chaosDetails *types.ChaosDetails) error {

	//Waiting for the ramp time before chaos injection
	if experimentsDetails.Control.RampTime != 0 {
		log.Infof("[Ramp]: Waiting for the %vs ramp time before injecting chaos", experimentsDetails.Control.RampTime)
		common.WaitForDuration(experimentsDetails.Control.RampTime)
	}

	// rolling update has predefined way of execution out of scope of serial/parallel
	if err := injectChaos(experimentsDetails, clients, chaosDetails, eventsDetails, resultDetails); err != nil {
			return err
		}

	////Waiting for the ramp time after chaos injection
	if experimentsDetails.Control.RampTime != 0 {
		log.Infof("[Ramp]: Waiting for the %vs ramp time after injecting chaos", experimentsDetails.Control.RampTime)
		common.WaitForDuration(experimentsDetails.Control.RampTime)
	}
	return nil
}


func injectChaos(experimentsDetails *experimentTypes.ExperimentDetails, clients clients.ClientSets, chaosDetails *types.ChaosDetails, eventsDetails *types.EventDetails, resultDetails *types.ResultDetails) error {

	if len(resultDetails.ProbeDetails) != 0 {
		if err := probe.RunProbes(chaosDetails, clients, resultDetails, "DuringChaos", eventsDetails); err != nil {
			return err
		}
	}

	//ChaosStartTimeStamp contains the start timestamp, when the chaos injection begin
	ChaosStartTimeStamp := time.Now()
	duration := int(time.Since(ChaosStartTimeStamp).Seconds())

	for duration < experimentsDetails.Control.ChaosDuration {

		if experimentsDetails.Strimzi.StrimziKafkaClusterName == ""  {
			return errors.Errorf("Please provide name of kafka cluster which is to be updated")
		}

		var err error
		if experimentsDetails.Control.EngineName != "" {
			msg := "Injecting " + experimentsDetails.Control.ExperimentName + " chaos on application pods"
			types.SetEngineEventAttributes(eventsDetails, types.ChaosInject, msg, "Normal", chaosDetails)
			events.GenerateEvents(eventsDetails, clients, chaosDetails, "ChaosEngine")
		}

		log.Info("[Chaos]: Chaos injection (Rolling Update) (iteration) begins")
		// injection
		podsUIDs, err := strimzi_kafka_resource.Update(experimentsDetails, clients)
		if err != nil {
			return err
		}

		// obtain single iteration duration
		chaosIntervalDuration, err := environment.GetChaosIntervalDuration(*experimentsDetails, chaosDetails.Randomness)
		if err != nil {
			return err
		}
		log.InfoWithValues("[Wait]: Waiting for single chaos interval", logrus.Fields{
			//"Resource type": resource.Type,
			"Chaos Interval Duration": chaosIntervalDuration})

		// actually wait and log info about waiting duration
		numberOfUnUpdatedPods, err := strimzi_kafka_resource.WaitForChaosIntervalDurationResources(*experimentsDetails,clients, podsUIDs, chaosIntervalDuration)
		if err != nil {
			return err
		}
		if numberOfUnUpdatedPods !=0  {
			return errors.Errorf("Not all pods were updated. %d were not updated.",numberOfUnUpdatedPods)
		}

		//Verify the status of pod after the chaos injection
		log.Info("[Status]: Verification for the recreation of application pod")
		if err = status.CheckApplicationStatus(experimentsDetails.App.Namespace, experimentsDetails.Control.AppLabel, experimentsDetails.Control.Timeout, experimentsDetails.Control.Delay, clients); err != nil {
			return err
		}

		duration = int(time.Since(ChaosStartTimeStamp).Seconds())
	}

	log.Infof("[Completion]: %v chaos is done", experimentsDetails.Control.ExperimentName)

	return nil
}
