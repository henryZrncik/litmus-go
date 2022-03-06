package time

import (
	"github.com/litmuschaos/litmus-go/pkg/clients"
	"github.com/litmuschaos/litmus-go/pkg/log"
	"github.com/litmuschaos/litmus-go/pkg/strimzi/types"
	"github.com/litmuschaos/litmus-go/pkg/utils/common"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

func GetChaosIntervalDuration(experimentsDetails types.ExperimentDetails, randomness bool) (int, error) {

	switch randomness {
	// if randomness pick time between lower and upper bound
	case true:
		intervals := strings.Split(experimentsDetails.Control.ChaosInterval, "-")
		var lowerBound, upperBound int
		switch len(intervals) {
		case 1:
			lowerBound = 0
			upperBound, _ = strconv.Atoi(intervals[0])
		case 2:
			lowerBound, _ = strconv.Atoi(intervals[0])
			upperBound, _ = strconv.Atoi(intervals[1])
		default:
			return 0, errors.Errorf("unable to parse CHAOS_INTERVAL, provide in valid format")
		}

		rand.Seed(time.Now().UnixNano())
		waitTime := lowerBound + rand.Intn(upperBound-lowerBound)
		return waitTime, nil
	default:
		intervalTime, err := strconv.Atoi(experimentsDetails.Control.ChaosInterval)
		if err != nil {
			return 0, errors.Errorf("unable to parse CHAOS_INTERVAL, provide in valid format")
		}
		return intervalTime, err
	}

}

// WaitForChaosIntervalDurationResources waits for time of single chaos interval
// while it keeps logging information about availability of resources.
func WaitForChaosIntervalDurationResources(experimentsDetails types.ExperimentDetails, clients clients.ClientSets, chaosIntervalDuration int) {
	ChaosStartTimeStamp := time.Now()
	duration := int(time.Since(ChaosStartTimeStamp).Seconds())
	waitTimeOfSingleChaosInterval := chaosIntervalDuration
	appNs := experimentsDetails.Control.AppNS

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
		}

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


