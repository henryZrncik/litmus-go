package jobs

import (
	"fmt"
	"github.com/litmuschaos/litmus-go/pkg/clients"
	"github.com/litmuschaos/litmus-go/pkg/log"
	"github.com/litmuschaos/litmus-go/pkg/utils/common"
	"github.com/litmuschaos/litmus-go/pkg/utils/retry"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
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








// JobExecutionTimeout wait for completion of job, if it does not end before provided time it is considered failed.
func JobExecutionTimeout(jobName, namespace string ,clients clients.ClientSets, delay int)  error{
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


// WaitForExecPod waits for specified time till status of
func WaitForExecPod(jobName, namespace string, timeoutDuration, delayDuration int, clients clients.ClientSets, possibleErrorMessage string) error  {
	return retry.
		Times(uint(timeoutDuration / delayDuration)).
		Wait(time.Duration(delayDuration) * time.Second).
		Try(func(attempt uint) error {
			resultJob, err := clients.KubeClient.BatchV1().Jobs(namespace).Get(jobName, metav1.GetOptions{})

			if err != nil {
				log.Errorf("error while waiting for kubernetes job: %v", err)
				return err
			}
			if resultJob.Status.Failed == 1 {
				return errors.Errorf("Job regarding %s failed", possibleErrorMessage)
			}

			if resultJob.Status.Succeeded == 1 {
				return nil
			}
			return errors.Errorf("timeout while waiting for: %s", possibleErrorMessage)
		})
}