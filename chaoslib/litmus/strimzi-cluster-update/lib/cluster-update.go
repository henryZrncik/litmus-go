package lib

import (
	"github.com/litmuschaos/litmus-go/pkg/clients"
	"github.com/litmuschaos/litmus-go/pkg/events"
	"github.com/litmuschaos/litmus-go/pkg/log"
	"github.com/litmuschaos/litmus-go/pkg/probe"
	"github.com/litmuschaos/litmus-go/pkg/strimzi/environment"
	experimentTypes "github.com/litmuschaos/litmus-go/pkg/strimzi/types"
	strimzi_utils "github.com/litmuschaos/litmus-go/pkg/strimzi/utils/resources"
	strimzi_kafka_resource "github.com/litmuschaos/litmus-go/pkg/strimzi/utils/update"
	"github.com/litmuschaos/litmus-go/pkg/types"
	"github.com/litmuschaos/litmus-go/pkg/utils/common"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"time"

	//v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//"strconv"
	"strings"
)

//PrepareChaosInjection contains the prepration steps before chaos injection
func PrepareChaosInjection(experimentsDetails *experimentTypes.ExperimentDetails, clients clients.ClientSets, resultDetails *types.ResultDetails, eventsDetails *types.EventDetails, chaosDetails *types.ChaosDetails) error {

	//Waiting for the ramp time before chaos injection
	if experimentsDetails.Control.RampTime != 0 {
		log.Infof("[Ramp]: Waiting for the %vs ramp time before injecting chaos", experimentsDetails.Control.RampTime)
		common.WaitForDuration(experimentsDetails.Control.RampTime)
	}

	switch strings.ToLower(experimentsDetails.Control.Sequence) {
	case "serial", "parallel":
		log.Infof("[Chaos]: Injecting chaos in serial mode")
		if err := injectChaos(experimentsDetails, clients, chaosDetails, eventsDetails, resultDetails); err != nil {
			return err
		}
	default:
		return errors.Errorf("%v sequence is not supported", experimentsDetails.Control.Sequence)
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
		var err error
		if experimentsDetails.Control.EngineName != "" {
			msg := "Injecting " + experimentsDetails.Control.ExperimentName + " chaos on application resources"
			types.SetEngineEventAttributes(eventsDetails, types.ChaosInject, msg, "Normal", chaosDetails)
			events.GenerateEvents(eventsDetails, clients, chaosDetails, "ChaosEngine")
		}

		//log.InfoWithValues("[Chaos]: If exist, following resources will be sequentially deleted", logrus.Fields{
		//	//"Resource type": resource.Type,
		//	"Resource": experimentsDetails.Resources.Resources})


		log.Info("[Chaos]: Chaos injection (iteration) begins")
		// injection
		// TODO improve waiting podsUIDs
		_, err = strimzi_kafka_resource.Update(experimentsDetails, clients)
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
		strimzi_utils.WaitForChaosIntervalDurationResources(*experimentsDetails,clients, chaosIntervalDuration)

		//TODO
		//Verify the health check of all kafka pods after update
		//log.Info("[Status]: Verification for the recreation of application resources")
		//if err = strimzi_utils.HealthCheckAll(experimentsDetails.App.Namespace,experimentsDetails.Resources.Resources,0,1,clients); err != nil {
		//	return err
		//}

		duration = int(time.Since(ChaosStartTimeStamp).Seconds())
	}

	log.Infof("[Completion]: %v chaos is done", experimentsDetails.Control.ExperimentName)

	return nil
}
