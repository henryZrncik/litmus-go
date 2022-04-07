package lib

import (
	"github.com/litmuschaos/litmus-go/pkg/clients"
	"github.com/litmuschaos/litmus-go/pkg/events"
	"github.com/litmuschaos/litmus-go/pkg/log"
	"github.com/litmuschaos/litmus-go/pkg/probe"
	"github.com/litmuschaos/litmus-go/pkg/strimzi/environment"
	//"github.com/litmuschaos/litmus-go/pkg/strimzi/environment"
	experimentTypes "github.com/litmuschaos/litmus-go/pkg/strimzi/types"
	strimzi_utils "github.com/litmuschaos/litmus-go/pkg/strimzi/utils/workers"
	"github.com/litmuschaos/litmus-go/pkg/types"
	"github.com/litmuschaos/litmus-go/pkg/utils/common"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"strings"
	"time"
)

// PrepareWorkerPodDelete contains the prepration steps before chaos injection
func PrepareWorkerPodDelete(experimentsDetails *experimentTypes.ExperimentDetails, clients clients.ClientSets, resultDetails *types.ResultDetails, eventsDetails *types.EventDetails, chaosDetails *types.ChaosDetails) error {

	if experimentsDetails.Connector.Name == "" {
		return errors.New("Not provided KAFKA_CONNECTOR_NAME value")
	}

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
	//// run the probes during chaos
	if len(resultDetails.ProbeDetails) != 0 {
		if err := probe.RunProbes(chaosDetails, clients, resultDetails, "DuringChaos", eventsDetails); err != nil {
			return err
		}
	}

	GracePeriod := int64(0)
	////ChaosStartTimeStamp contains the start timestamp, when the chaos injection begin
	ChaosStartTimeStamp := time.Now()
	duration := int(time.Since(ChaosStartTimeStamp).Seconds())
	targetPods := ""

	for duration < experimentsDetails.Control.ChaosDuration {
		switch strings.ToLower(experimentsDetails.Connector.KillTaskedWorkers) {
		case "enable", "true", "yes":
			log.Info("[Info]: obtaining all available pods")
			allPods, err := common.GetPodList(targetPods, experimentsDetails.Control.PodsAffectedPerc, clients, chaosDetails)
			newPods, err := strimzi_utils.GetTaskedPods(experimentsDetails, clients, allPods)
			if err != nil {
				return err
			}
			targetPods = newPods
		}

		// pods are obtained by label in any case, separation can be done only to tasked by param
		if  experimentsDetails.Connector.Name == "" || chaosDetails.AppDetail.Label == "" {
			return errors.Errorf("please provide name kafka connector: env KAFKA_CONNECTOR_NAME")
		}

		// provide litmus with "" none pods if
		targetPodList, err := common.GetPodList(targetPods, experimentsDetails.Control.PodsAffectedPerc, clients, chaosDetails)
		if err != nil {
			return err
		}

		if experimentsDetails.Control.EngineName != "" {
			msg := "Injecting " + experimentsDetails.Control.ExperimentName + " chaos on application pod"
			types.SetEngineEventAttributes(eventsDetails, types.ChaosInject, msg, "Normal", chaosDetails)
			events.GenerateEvents(eventsDetails, clients, chaosDetails, "ChaosEngine")
		}

		var ips []string
		//Deleting the application pod
		for _, pod := range targetPodList.Items {

			// for each pod that is to be deleted verify it is task.
			isTask, err := strimzi_utils.IsPodTask(experimentsDetails, clients, pod)
			if err != nil {
				log.Error("[Fatal]: Error while fetching data about Worker Pod")
				return err
			}
			// we need ips of tasks (those that will be deleted) before their deletion
			if isTask {
				log.Infof("[Info]: Pod %s is task (IP: %s)", pod.Name, pod.Status.PodIP)
				ips = append(ips, pod.Status.PodIP)
			} else {
				log.Infof("[Info] Pod %s is not task", pod.Name)
			}

			// kill the pod
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

			// wait for duration and make sure that ips taken from tasks are recreated
			chaosIntervalDuration, err := environment.GetChaosIntervalDuration(*experimentsDetails, chaosDetails.Randomness)
			if err != nil {
				return err
			}
			log.InfoWithValues("[Wait]: Waiting for single chaos interval", logrus.Fields{
				"Chaos Interval Duration": chaosIntervalDuration})
			err = strimzi_utils.WaitForTaskRecreation2(experimentsDetails,clients, ips , chaosIntervalDuration)
			if err != nil  {
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
	//// run the probes during chaos
	if len(resultDetails.ProbeDetails) != 0 {
		if err := probe.RunProbes(chaosDetails, clients, resultDetails, "DuringChaos", eventsDetails); err != nil {
			return err
		}
	}

	GracePeriod := int64(0)
	////ChaosStartTimeStamp contains the start timestamp, when the chaos injection begin
	ChaosStartTimeStamp := time.Now()
	duration := int(time.Since(ChaosStartTimeStamp).Seconds())
	targetPods := ""

	for duration < experimentsDetails.Control.ChaosDuration {
		switch strings.ToLower(experimentsDetails.Connector.KillTaskedWorkers) {
		case "enable", "true", "yes":
			log.Info("[Info]: obtaining all available pods")
			// we pass here 100% as we want to consider only
			allPods, err := common.GetPodList(targetPods, 100, clients, chaosDetails)
			newPods, err := strimzi_utils.GetTaskedPods(experimentsDetails, clients, allPods)
			if err != nil {
				return err
			}
			targetPods = newPods
		}

		// pods are obtained by label in any case, separation can be done only to tasked by param
		if  experimentsDetails.Connector.Name == "" || chaosDetails.AppDetail.Label == "" {
			return errors.Errorf("please provide name kafka connector: env KAFKA_CONNECTOR_NAME")
		}

		// provide litmus with "" none pods if
		targetPodList, err := common.GetPodList(targetPods, experimentsDetails.Control.PodsAffectedPerc, clients, chaosDetails)
		if err != nil {
			return err
		}

		if experimentsDetails.Control.EngineName != "" {
			msg := "Injecting " + experimentsDetails.Control.ExperimentName + " chaos on application pod"
			types.SetEngineEventAttributes(eventsDetails, types.ChaosInject, msg, "Normal", chaosDetails)
			events.GenerateEvents(eventsDetails, clients, chaosDetails, "ChaosEngine")
		}

		var ips []string
		//Deleting the application pod
		for _, pod := range targetPodList.Items {

			// for each pod that is to be deleted verify it is task.
			isTask, err := strimzi_utils.IsPodTask(experimentsDetails, clients, pod)
			if err != nil {
				log.Error("[Fatal]: Error while fetching data about Worker Pod")
				return err
			}
			// we need ips of tasks (those that will be deleted) before their deletion
			if isTask {
				log.Infof("[Info]: Pod %s is task (IP: %s)", pod.Name, pod.Status.PodIP)
				ips = append(ips, pod.Status.PodIP)
			} else {
				log.Infof("[Info] Pod %s is not task", pod.Name)
			}
		}
		for _, pod := range targetPodList.Items {
			// kill the pod
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

		// wait for duration and make sure that ips taken from tasks are recreated
		chaosIntervalDuration, err := environment.GetChaosIntervalDuration(*experimentsDetails, chaosDetails.Randomness)
		if err != nil {
			return err
		}
		log.InfoWithValues("[Wait]: Waiting for single chaos interval", logrus.Fields{
			"Chaos Interval Duration": chaosIntervalDuration})
		err = strimzi_utils.WaitForTaskRecreation2(experimentsDetails,clients, ips , chaosIntervalDuration)
		if err != nil  {
			return err
		}
		duration = int(time.Since(ChaosStartTimeStamp).Seconds())
	}
	log.Infof("[Completion]: %v chaos is done", experimentsDetails.Control.ExperimentName)

	return nil
}

