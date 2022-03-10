package update

import (
	"github.com/litmuschaos/litmus-go/pkg/clients"
	"github.com/litmuschaos/litmus-go/pkg/log"
	"github.com/litmuschaos/litmus-go/pkg/strimzi/types"
	"github.com/litmuschaos/litmus-go/pkg/utils/common"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1pod "k8s.io/api/core/v1"
	k8types "k8s.io/apimachinery/pkg/types"
	"time"
)

// WaitForChaosIntervalDurationResources waits for time of single chaos interval
// while it keeps logging information about availability of resources.
func WaitForChaosIntervalDurationResources(exp types.ExperimentDetails, clients clients.ClientSets, podsUIDs []k8types.UID,  chaosIntervalDuration int) {
	ChaosStartTimeStamp := time.Now()
	duration := int(time.Since(ChaosStartTimeStamp).Seconds())
	waitTimeOfSingleChaosInterval := chaosIntervalDuration

	retryCount := 0
	// there is no need to log info about pods recreated more than once
	var isLogged bool = false

	for duration < waitTimeOfSingleChaosInterval {
		retryCount++
		var err error

		// get present pods with kafka label
		pods, err := clients.KubeClient.CoreV1().Pods(exp.App.Namespace).List(v1.ListOptions{LabelSelector: exp.Kafka.Label})
		if err != nil {
			return
		}

		// how many pods are already changed.
		podReadyAndChanged := 0
		// how many pods we need to change
		podTotal := len(pods.Items)

		// check how many pods are changed

		// for each UID (of original Pods) increase counter if it is not present
		terminatedOldPodsCount := 0
		for _, uid := range podsUIDs{
			// pod is not present, increase terminated count
			if !isUIDAmongstPods(uid, pods.Items){
				terminatedOldPodsCount++
			}
		}

		// all old pods were terminated, waiting for
		if terminatedOldPodsCount == podTotal{

		}


		for _, pod := range pods.Items {
			if  {

			}
			pod.UID
		}

		// check how many pods are ready



		common.WaitForDuration(1)
		duration = int(time.Since(ChaosStartTimeStamp).Seconds())

		// as soon as all resources are ready, this change is printed
		if err == nil && !isLogged {
			isLogged = true
			log.Infof("[Wait]: time %v/%v. All resources available", duration, waitTimeOfSingleChaosInterval)
			continue
		}

		// state (that we wait) is printed approximately every 30 seconds
		if retryCount%30 == 0 {
			if err != nil {
				log.Infof("[Wait]: time %v/%v. %v ", duration, waitTimeOfSingleChaosInterval, err.Error())
			} else {
				log.Infof("[Wait]: time %v/%v. All resources available", duration, waitTimeOfSingleChaosInterval)
			}
		}
	}

}


func isUIDAmongstPods(wantedUID k8types.UID, listOfPods []v1pod.Pod) bool {
	for _, pod  := range listOfPods {
		if pod.UID == wantedUID {
			return true
		}
	}
	return false
}
