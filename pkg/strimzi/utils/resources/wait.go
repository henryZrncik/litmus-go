package resources

import (
	"github.com/litmuschaos/litmus-go/pkg/clients"
	"github.com/litmuschaos/litmus-go/pkg/log"
	"github.com/litmuschaos/litmus-go/pkg/strimzi/types"
	"github.com/litmuschaos/litmus-go/pkg/utils/common"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

// WaitForChaosIntervalDurationResources waits for time of single chaos interval
// while it keeps logging information about availability of resources.
func WaitForChaosIntervalDurationResources(experimentsDetails types.ExperimentDetails, clients clients.ClientSets, chaosIntervalDuration int) {
	ChaosStartTimeStamp := time.Now()
	duration := int(time.Since(ChaosStartTimeStamp).Seconds())
	waitTimeOfSingleChaosInterval := chaosIntervalDuration
	appNs := experimentsDetails.App.Namespace

	retryCount := 0
	// there is no need to log info about resources recreated more than once
	var isLogged bool = false

	for duration < waitTimeOfSingleChaosInterval {
		retryCount++
		var err error

		for _, resource := range experimentsDetails.Resources.Resources {
			switch resource.Type {
			case types.ConfigMapResourceType:
				_, err = clients.KubeClient.CoreV1().ConfigMaps(appNs).Get(resource.Name, v1.GetOptions{})
			case types.ServiceResourceType:
				_, err = clients.KubeClient.CoreV1().Services(appNs).Get(resource.Name, v1.GetOptions{})
			case types.SecretResourceType:
				_, err = clients.KubeClient.CoreV1().Secrets(appNs).Get(resource.Name, v1.GetOptions{})
			}
			// once an error is found, another successful read would override it.
			if err != nil {
				break
			}
		}

		common.WaitForDuration(experimentsDetails.Control.Delay)
		duration = int(time.Since(ChaosStartTimeStamp).Seconds())

		// as soon as all resources are ready, this change is printed
		if err == nil && !isLogged {
			isLogged = true
			log.Infof("[Wait]: Time %v/%v. All resources available", duration, waitTimeOfSingleChaosInterval)
			switch experimentsDetails.App.EndChaosInjectionASAP {
			case "enable", "true", "yes":
				log.Info("[Info]: ending successful chaos injection ASAP")
				return
			}
			continue
		}

		// state (that we wait) is printed every 10th iteration
		if retryCount% 10 == 0 {
			if err != nil {
				log.Infof("[Wait]: Time %v/%v. %v ", duration, waitTimeOfSingleChaosInterval, err.Error())
			} else {
				log.Infof("[Wait]: Time %v/%v. All resources available", duration, waitTimeOfSingleChaosInterval)
			}
		}
	}
	log.Infof("[Wait]: time %v/%v. Waiting over", duration, waitTimeOfSingleChaosInterval)
}
