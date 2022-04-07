package workers

import (
	"fmt"
	"github.com/litmuschaos/litmus-go/pkg/clients"
	"github.com/litmuschaos/litmus-go/pkg/log"
	experimentTypes "github.com/litmuschaos/litmus-go/pkg/strimzi/types"
	"github.com/litmuschaos/litmus-go/pkg/utils/common"
	"time"
)

func WaitForTaskRecreation2(exp *experimentTypes.ExperimentDetails, clients clients.ClientSets, ips []string, chaosIntervalDuration int) error{
	ChaosStartTimeStamp := time.Now()
	duration := int(time.Since(ChaosStartTimeStamp).Seconds())
	waitTimeOfSingleChaosInterval := chaosIntervalDuration

	retryCount := 0

	// there is no need to log info about resources recreated more than once
	var isLogged bool = false
	var result bool = false
	var prevailedTaskIp string

	log.Infof("[Info]: %d of tasks is being deleted", len(ips))

	for duration < waitTimeOfSingleChaosInterval {
		retryCount++
		var err error

		currentConnectorTaskIps, err := GetConnectorTasksIps(exp,clients)
		if err != nil {
			return err
		}

		prevailedTaskIp = isAtLeastOneOldTaskPresent(ips,currentConnectorTaskIps)
		if prevailedTaskIp == "" {
			result = true
		}

		common.WaitForDuration(exp.Control.Delay)
		duration = int(time.Since(ChaosStartTimeStamp).Seconds())

		// as soon as all resources are ready, this change is printed
		if result  && !isLogged {
			isLogged = true
			result = true
			log.Infof("[Wait]: Time %v/%v. All tasks recreated", duration, waitTimeOfSingleChaosInterval)

			switch exp.App.EndChaosInjectionASAP {
			case "enable", "true", "yes":
				log.Info("[Info]: ending successful chaos injection ASAP")
				return nil
			}
			continue
		}

		// state (that we wait) is printed every 10th iteration
		if retryCount% 10 == 0 {
			if !result  {
				log.Infof("[Wait]: Time %v/%v. Old Task with IP %s is still present", duration, waitTimeOfSingleChaosInterval, prevailedTaskIp)

			} else {
				log.Infof("[Wait]: Time %v/%v. Done", duration, waitTimeOfSingleChaosInterval)
			}
		}
	}

	log.Infof("[Wait]: time %v/%v. Waiting over", duration, waitTimeOfSingleChaosInterval)
	if !result {
		return fmt.Errorf("task  (ip: %s)  not replaced witin expected time", prevailedTaskIp)
	}
	return nil
}

func isAtLeastOneOldTaskPresent(oldIps, newIps []string ) string{
	for _, oldIp := range oldIps {
		for _, currentIp := range newIps {
			if oldIp == currentIp {
				return currentIp
			}
		}
	}
	return ""
}



