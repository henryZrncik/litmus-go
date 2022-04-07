package workers

import (
	"github.com/litmuschaos/litmus-go/pkg/clients"
	"github.com/litmuschaos/litmus-go/pkg/log"
	"strings"

	experimentTypes "github.com/litmuschaos/litmus-go/pkg/strimzi/types"
	core_v1 "k8s.io/api/core/v1"


)

type WorkerPod struct {
	IP string
	Name string
	IsTask bool
}


func IsPodTask(exp *experimentTypes.ExperimentDetails, clients clients.ClientSets, pod core_v1.Pod ) (bool, error) {
	// IPs of tasks
	ips, err := GetConnectorTasksIps(exp, clients)
	if err != nil {
		return false, err
	}
	return contains(ips, pod.Status.PodIP), nil
}

func GetTaskedPods(exp *experimentTypes.ExperimentDetails, clients clients.ClientSets, podList core_v1.PodList ) (string, error) {
	// IPs of tasks
	ips, err := GetConnectorTasksIps(exp, clients)
	if err != nil {
		return "", err
	}

	// resulted tasks
	var pods []string
	for _, pod := range podList.Items {
		if contains(ips, pod.Status.PodIP) {
			pods = append(pods, pod.Name)
		}
	}
	log.Infof("[Info]: Number of pods targeted after reduction of aim to tasks only: %d", len(pods))
	podsCommaSeparated := strings.Join(pods, ",")
	return podsCommaSeparated, nil
}



// contains checks if a string is present in a slice
func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}