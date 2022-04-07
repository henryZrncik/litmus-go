package update

import (
	"github.com/litmuschaos/litmus-go/pkg/clients"
	"github.com/litmuschaos/litmus-go/pkg/log"
	"github.com/litmuschaos/litmus-go/pkg/strimzi/types"
	"github.com/litmuschaos/litmus-go/pkg/utils/common"
	v1pod "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8types "k8s.io/apimachinery/pkg/types"
	"time"
)

// WaitForChaosIntervalDurationResources waits for time of single chaos interval
// while it keeps logging information about availability of resources (updated pods).
func WaitForChaosIntervalDurationResources(exp types.ExperimentDetails, clients clients.ClientSets, podsUIDs []k8types.UID,  chaosIntervalDuration int) (int, error) {
	ChaosStartTimeStamp := time.Now()
	duration := int(time.Since(ChaosStartTimeStamp).Seconds())
	waitTimeOfSingleChaosInterval := chaosIntervalDuration

	retryCount := 0
	// there is no need to log info about pods recreated more than once
	var isLogged bool = false

	terminatedOldPodsCount := 0
	podTotal := 0

	for duration < waitTimeOfSingleChaosInterval {
		retryCount++
		var err error

		// get present pods with kafka label
		pods, err := clients.KubeClient.CoreV1().Pods(exp.App.Namespace).List(v1.ListOptions{LabelSelector: exp.Control.AppLabel})
		if err != nil {
			return 0, err
		}

		// how many pods we need to change
		podTotal = len(pods.Items)
		// for each UID (of original Pods) increase counter if it is not present
		terminatedOldPodsCount = 0

		// check how many pods are changed
		for _, uid := range podsUIDs{
			// pod is not present, increase terminated count
			if !isUIDAmongstPods(uid, pods.Items){
				terminatedOldPodsCount++
			}
		}

		common.WaitForDuration(exp.Control.Delay)
		duration = int(time.Since(ChaosStartTimeStamp).Seconds())

		// all old pods were terminated,it can still take some time for new pods to  become ready
		if terminatedOldPodsCount == podTotal && !isLogged{
			isLogged = true
			log.Infof("[Wait]: Time %v/%v. Updated strimzi kafka pods:%d/%d", duration, waitTimeOfSingleChaosInterval, terminatedOldPodsCount, podTotal)

			switch exp.App.EndChaosInjectionASAP {
			case "enable", "true", "yes":
				log.Info("[Info]: ending successful chaos injection ASAP")
				return 0, nil
			}
			continue
		}

		//to make waiting more informative, state (while we wait) is printed every 10th iteration
		if retryCount % 10 == 0 {
			if terminatedOldPodsCount != podTotal {
				log.Infof("[Wait]: Time %v/%v. Updated strimzi kafka pods:%d/%d", duration, waitTimeOfSingleChaosInterval, terminatedOldPodsCount, podTotal)
			} else {
				log.Infof("[Wait]: Time %v/%v. All strimzi kafka pods are updated (%d/%d).", duration, waitTimeOfSingleChaosInterval,terminatedOldPodsCount, podTotal)
			}
		}
	}

	log.Infof("[Wait]: Time %v/%v. Waiting over", duration, waitTimeOfSingleChaosInterval)
	return podTotal - terminatedOldPodsCount, nil
}


func isUIDAmongstPods(wantedUID k8types.UID, listOfPods []v1pod.Pod) bool {
	for _, pod  := range listOfPods {
		if pod.UID == wantedUID {
			return true
		}
	}
	return false
}
