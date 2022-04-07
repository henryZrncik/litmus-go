package jobs

import (
	"fmt"
	"github.com/litmuschaos/litmus-go/pkg/clients"
	"github.com/litmuschaos/litmus-go/pkg/utils/common"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

const (
	stateFailed string = "failed"
	stateSucceeded string = "succeeded"
	stateRunning string = "running"
	// includes pending and all future possible states we don't care about
	stateUnknown string = "unknown"
)

// WaitForRunningJob waits until job's pod is running state (being pulled and started)
func WaitForRunningJob(jobName, namespace string ,clients clients.ClientSets, delay int)  error{
	ChaosStartTimeStamp := time.Now()
	duration := int(time.Since(ChaosStartTimeStamp).Seconds())

	for duration < 30 {
		var jobPod, err = clients.KubeClient.CoreV1().Pods(namespace).List(metav1.ListOptions{LabelSelector: "job-name="+ jobName})

		// if error is encountered
		if err != nil{
			return err
		}
		// pod for given job was not yet created
		if len(jobPod.Items) < 1  {
			common.WaitForDuration(delay)
			duration = int(time.Since(ChaosStartTimeStamp).Seconds())
			continue
		}
		if jobPod.Items[0].Status.Phase == corev1.PodFailed {
			return fmt.Errorf("job %s failed prematurely", jobPod.Items[0].ObjectMeta.Name)
		}
		return nil
	}
	return  nil
}

func WaitForJobEnd(jobName, namespace string, timeout, retryBackoff int, clients clients.ClientSets) error{
	jobResultedsState, err := waitForJobState(jobName, namespace, timeout, retryBackoff, []string{stateFailed, stateSucceeded}, clients )
	// timeout or other error
	if err != nil {
		return err
	}
	switch jobResultedsState {
	case stateSucceeded:
		return nil
	default:
		return errors.New("Failed")
	}
}

func WaitForJobStart(jobName, namespace string, timeout, retryBackoff int, clients clients.ClientSets) error{
	jobResultedsState, err := waitForJobState(jobName, namespace, timeout, retryBackoff, []string{stateRunning, stateSucceeded, stateFailed}, clients )
	// Timeout
	if err != nil {
		return err
	}
	switch jobResultedsState {
	case stateRunning, stateSucceeded:
		return nil
	default:
		return errors.New("Failed")
	}
}

func waitForJobState(jobName, namespace string, timeout, retryBackoff int, desiredStates[]string, clients clients.ClientSets) (string, error){
	ChaosStartTimeStamp := time.Now()
	duration := int(time.Since(ChaosStartTimeStamp).Seconds())
	// waiting is executed at least once for case when check is called after timeout (timeout = 0)
	for ok := true; ok; ok = duration < timeout {
		result, err := isInOneOfStates(jobName, namespace, desiredStates , clients)
		// job obtained error while it was executed
		if err != nil {
			return "", err
		}
		// job is in desired state and that state is returned
		if result != "" {
			return result, nil
		}
		// getting of result is retried every back off time, and duraiton is updated.
		common.WaitForDuration(retryBackoff)
		duration = int(time.Since(ChaosStartTimeStamp).Seconds())
	}
	// timeout while waiting for job
	return "", errors.Errorf("Timeout.")
}

func isInOneOfStates(jobName, namespace string, desiredStates[]string, clients clients.ClientSets) (string, error) {
	res, err := getJobResult(jobName,namespace, clients)
	if err != nil {
		return "", err
	}
	return contains(desiredStates, res), nil
}

// getJobResult returns result of job (running, failed, succeeded)
func getJobResult(jobName, namespace string ,clients clients.ClientSets) (string, error){
	resultJob, err := clients.KubeClient.BatchV1().Jobs(namespace).Get(jobName, metav1.GetOptions{})
	if err != nil {
		return "",err
	}
	if resultJob.Status.Failed == 1 {
		return stateFailed, nil
	}
	if resultJob.Status.Succeeded == 1 {
		return stateSucceeded, nil
	}
	if resultJob.Status.Active == 1 {
		return  stateRunning, nil
	}
	// job is still running
	return stateUnknown, nil
}

func contains(s []string, e string) string {
	for _, a := range s {
		if a == e {
			return e
		}
	}
	return ""
}
