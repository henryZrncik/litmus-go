package lib

import (
	"github.com/litmuschaos/litmus-go/pkg/clients"
	"github.com/litmuschaos/litmus-go/pkg/events"
	"github.com/litmuschaos/litmus-go/pkg/log"
	"github.com/litmuschaos/litmus-go/pkg/probe"
	"github.com/litmuschaos/litmus-go/pkg/status"
	experimentTypes "github.com/litmuschaos/litmus-go/pkg/strimzi/types"
	"github.com/litmuschaos/litmus-go/pkg/types"
	"github.com/litmuschaos/litmus-go/pkg/utils/annotation"
	"github.com/litmuschaos/litmus-go/pkg/utils/common"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strconv"
	"strings"
	"time"
)

// PreparePodDelete contains the prepration steps before chaos injection
func PreparePodDelete(experimentsDetails *experimentTypes.ExperimentDetails, clients clients.ClientSets, resultDetails *types.ResultDetails, eventsDetails *types.EventDetails, chaosDetails *types.ChaosDetails) error {

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

	GracePeriod := int64(0)
	//ChaosStartTimeStamp contains the start timestamp, when the chaos injection begin
	ChaosStartTimeStamp := time.Now()
	duration := int(time.Since(ChaosStartTimeStamp).Seconds())

	for duration < experimentsDetails.Control.ChaosDuration {
		// Get the target pod details for the chaos execution
		// if the target pod is not defined it will derive the random target pod list using pod affected percentage
		if chaosDetails.AppDetail.Label == "" && experimentsDetails.Kafka.KafkaInstancesName == "" {
			return errors.Errorf("please provide on of options: kafka label, instances or enable killing of leader as well as liveness stream")
		}
		targetPodList, err := common.GetPodList(experimentsDetails.Kafka.KafkaInstancesName, experimentsDetails.Control.PodsAffectedPerc, clients, chaosDetails)
		if err != nil {
			return err
		}

		// deriving the parent name of the target resources
		if chaosDetails.AppDetail.Kind != "" {
			for _, pod := range targetPodList.Items {
				parentName, err := annotation.GetParentName(clients, pod, chaosDetails)
				if err != nil {
					return err
				}
				common.SetParentName(parentName, chaosDetails)
			}
			for _, target := range chaosDetails.ParentsResources {
				common.SetTargets(target, "targeted", chaosDetails.AppDetail.Kind, chaosDetails)
			}
		}

		if experimentsDetails.Control.EngineName != "" {
			msg := "Injecting " + experimentsDetails.Control.ExperimentName + " chaos on application pod"
			types.SetEngineEventAttributes(eventsDetails, types.ChaosInject, msg, "Normal", chaosDetails)
			events.GenerateEvents(eventsDetails, clients, chaosDetails, "ChaosEngine")
		}

		//Deleting the application pod
		for _, pod := range targetPodList.Items {

			log.InfoWithValues("[Info]: Killing the following pods", logrus.Fields{
				"PodName": pod.Name})

			if experimentsDetails.Control.Force {
				err = clients.KubeClient.CoreV1().Pods(experimentsDetails.App.Namespace).Delete(pod.Name, &v1.DeleteOptions{GracePeriodSeconds: &GracePeriod})
			} else {
				err = clients.KubeClient.CoreV1().Pods(experimentsDetails.App.Namespace).Delete(pod.Name, &v1.DeleteOptions{})
			}
			if err != nil {
				return err
			}

			switch chaosDetails.Randomness {
			case true:
				if err := common.RandomInterval(experimentsDetails.Control.ChaosInterval); err != nil {
					return err
				}
			default:
				//Waiting for the chaos interval after chaos injection
				if experimentsDetails.Control.ChaosInterval != "" {
					log.Infof("[Wait]: Wait for the chaos interval %vs", experimentsDetails.Control.ChaosInterval)
					waitTime, _ := strconv.Atoi(experimentsDetails.Control.ChaosInterval)
					common.WaitForDuration(waitTime)
				}
			}

			//Verify the status of pod after the chaos injection
			log.Info("[Status]: Verification for the recreation of application pod")
			if err = status.CheckApplicationStatus(experimentsDetails.App.Namespace, chaosDetails.AppDetail.Label, experimentsDetails.Control.Timeout, experimentsDetails.Control.Delay, clients); err != nil {
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

	GracePeriod := int64(0)
	//ChaosStartTimeStamp contains the start timestamp, when the chaos injection begin
	ChaosStartTimeStamp := time.Now()
	duration := int(time.Since(ChaosStartTimeStamp).Seconds())


	for duration < experimentsDetails.Control.ChaosDuration {
		// Get the target pod details for the chaos execution
		// if the target pod is not defined it will derive the random target pod list using pod affected percentage
		if experimentsDetails.Kafka.KafkaInstancesName == "" && chaosDetails.AppDetail.Label == "" {
			return errors.Errorf("please provide one of the appLabel or KAFKA_BROKER")
		}
		targetPodList, err := common.GetPodList(experimentsDetails.Kafka.KafkaInstancesName, experimentsDetails.Control.PodsAffectedPerc, clients, chaosDetails)
		if err != nil {
			return err
		}

		// deriving the parent name of the target resources
		if chaosDetails.AppDetail.Kind != "" {
			for _, pod := range targetPodList.Items {
				parentName, err := annotation.GetParentName(clients, pod, chaosDetails)
				if err != nil {
					return err
				}
				common.SetParentName(parentName, chaosDetails)
			}
			for _, target := range chaosDetails.ParentsResources {
				common.SetTargets(target, "targeted", chaosDetails.AppDetail.Kind, chaosDetails)
			}
		}

		if experimentsDetails.Control.EngineName != "" {
			msg := "Injecting " + experimentsDetails.Control.ExperimentName + " chaos on application pod"
			types.SetEngineEventAttributes(eventsDetails, types.ChaosInject, msg, "Normal", chaosDetails)
			events.GenerateEvents(eventsDetails, clients, chaosDetails, "ChaosEngine")
		}

		//Deleting the application pod
		for _, pod := range targetPodList.Items {

			log.InfoWithValues("[Info]: Killing the following pods", logrus.Fields{
				"PodName": pod.Name})

			if experimentsDetails.Control.Force {
				err = clients.KubeClient.CoreV1().Pods(experimentsDetails.App.Namespace).Delete(pod.Name, &v1.DeleteOptions{GracePeriodSeconds: &GracePeriod})
			} else {
				err = clients.KubeClient.CoreV1().Pods(experimentsDetails.App.Namespace).Delete(pod.Name, &v1.DeleteOptions{})
			}
			if err != nil {
				return err
			}
		}

		switch chaosDetails.Randomness {
		case true:
			if err := common.RandomInterval(experimentsDetails.Control.ChaosInterval); err != nil {
				return err
			}
		default:
			//Waiting for the chaos interval after chaos injection
			if experimentsDetails.Control.ChaosInterval != "" {
				log.Infof("[Wait]: Wait for the chaos interval %vs", experimentsDetails.Control.ChaosInterval)
				waitTime, _ := strconv.Atoi(experimentsDetails.Control.ChaosInterval)
				common.WaitForDuration(waitTime)
			}
		}

		//Verify the status of pod after the chaos injection
		log.Info("[Status]: Verification for the recreation of application pod")
		if err = status.CheckApplicationStatus(experimentsDetails.App.Namespace, chaosDetails.AppDetail.Label, experimentsDetails.Control.Timeout, experimentsDetails.Control.Delay, clients); err != nil {
			return err
		}

		duration = int(time.Since(ChaosStartTimeStamp).Seconds())
	}

	log.Infof("[Completion]: %v chaos is done", experimentsDetails.Control.ExperimentName)

	return nil
}

