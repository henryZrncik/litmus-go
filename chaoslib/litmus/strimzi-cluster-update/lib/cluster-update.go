package lib

import (
	"github.com/litmuschaos/litmus-go/pkg/clients"
	"github.com/litmuschaos/litmus-go/pkg/log"
	experimentTypes "github.com/litmuschaos/litmus-go/pkg/strimzi/types"
	"github.com/litmuschaos/litmus-go/pkg/types"
	"github.com/pkg/errors"
	//v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//"strconv"
	"strings"
)

//PrepareChaosInjection contains the prepration steps before chaos injection
func PrepareChaosInjection(experimentsDetails *experimentTypes.ExperimentDetails, clients clients.ClientSets, resultDetails *types.ResultDetails, eventsDetails *types.EventDetails, chaosDetails *types.ChaosDetails) error {

	//Waiting for the ramp time before chaos injection
	//if experimentsDetails.Control.RampTime != 0 {
	//	log.Infof("[Ramp]: Waiting for the %vs ramp time before injecting chaos", experimentsDetails.Control.RampTime)
	//	common.WaitForDuration(experimentsDetails.Control.RampTime)
	//}

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
	//
	////Waiting for the ramp time after chaos injection
	//if experimentsDetails.Control.RampTime != 0 {
	//	log.Infof("[Ramp]: Waiting for the %vs ramp time after injecting chaos", experimentsDetails.Control.RampTime)
	//	common.WaitForDuration(experimentsDetails.Control.RampTime)
	//}
	return nil
}

// injectChaosInSerialMode delete the kafka broker pods in serial mode(one by one)
func injectChaosInSerialMode(experimentsDetails *experimentTypes.ExperimentDetails, clients clients.ClientSets, chaosDetails *types.ChaosDetails, eventsDetails *types.EventDetails, resultDetails *types.ResultDetails) error {


	return nil
}

// injectChaosInParallelMode delete the kafka broker pods in parallel mode (all at once)
func injectChaosInParallelMode(experimentsDetails *experimentTypes.ExperimentDetails, clients clients.ClientSets, chaosDetails *types.ChaosDetails, eventsDetails *types.EventDetails, resultDetails *types.ResultDetails) error {
	return nil

}
