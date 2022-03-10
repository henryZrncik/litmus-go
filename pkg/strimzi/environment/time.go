package environment

import (
	"github.com/litmuschaos/litmus-go/pkg/strimzi/types"
	"github.com/pkg/errors"
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

