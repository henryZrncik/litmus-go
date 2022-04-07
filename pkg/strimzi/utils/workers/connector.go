package workers

import (
	"fmt"
	"github.com/litmuschaos/litmus-go/pkg/clients"
	experimentTypes "github.com/litmuschaos/litmus-go/pkg/strimzi/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

func GetConnectorTasksIps(exp *experimentTypes.ExperimentDetails, clients clients.ClientSets) ([]string, error){
	client := exp.Strimzi.Client
	connector, err := client.KafkaConnector(exp.App.Namespace).Get(exp.Connector.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	var tasks []string
	for _, task := range connector.ConnectorStatus.ConnectorStatusInner.Tasks {

		workerPodsIps := strings.Split(task.WorkerId, ":")
		if len(workerPodsIps) != 2 {
			return nil, fmt.Errorf("connector %s does not contain correctly formatted metadata ( worker workerPodsIps: %s)", exp.Connector.Name, task.WorkerId)
		}
		// part of workerId holding its IP is in first elem.
		tasks = append(tasks,  workerPodsIps[0])
	}

	return tasks, nil
}

