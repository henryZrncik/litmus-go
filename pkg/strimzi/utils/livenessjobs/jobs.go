package jobs

import (
	"bytes"
	"github.com/litmuschaos/litmus-go/pkg/clients"
	"github.com/litmuschaos/litmus-go/pkg/log"
	"github.com/pkg/errors"
	"io"
	batchV1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetJobLogs returns logs from
func GetJobLogs(jobName, namespace string ,clients clients.ClientSets) (string, error){
	log.Infof("[Info]: Obtaining logs")
	// get workerPod name by Job label (i.e., Job name)
	var x, err = clients.KubeClient.CoreV1().Pods(namespace).List(metav1.ListOptions{LabelSelector: "job-name="+ jobName})
	if err != nil {
		return "", err
	}
	// retrieve logs from workerPod
	workerPod := x.Items[0]
	podLogOpts := corev1.PodLogOptions{Previous: false}

	req := clients.KubeClient.CoreV1().Pods(namespace).GetLogs(workerPod.Name, &podLogOpts)
	podLogs, err := req.Stream()
	if err != nil {
		return "", err
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return "", err
	}
	str := buf.String()

	// returning logs
	return str, nil
}


// ExecKube create new job and execute cmd
func ExecKube(jobName, runId, imageName, namespace, command string, envs []corev1.EnvVar, clients clients.ClientSets) error{
	var cmdArray = []string{}
	// if cmd "" we assume that image itself has default CMD or ENTRYPOINT set up.
	if command != "" {
		cmdArray = append([]string{"sh", "-c"}, command)
	}

	job := &batchV1.Job{
		TypeMeta: metav1.TypeMeta{
			Kind: "Job",
			APIVersion: "batch/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: jobName,
			Labels: map[string]string{
				"run":                       runId,
				"app":                       "kafka-liveness",
				"name":                      jobName,
				"app.kubernetes.io/part-of": "litmus",
			},
		},
		Spec: batchV1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"run":                       runId,
						"app":                       "kafka-liveness",
						"app.kubernetes.io/part-of": "litmus",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            "job-container",
							Image:           imageName,
							Command:         cmdArray,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Env:             envs,
						},
					},

					RestartPolicy: corev1.RestartPolicyNever,

				},
			},
			BackoffLimit: new(int32),
		},
	}

	_, err := clients.KubeClient.BatchV1().Jobs(namespace).Create(job)
	return err
}

// GetJobResult returns result of job (running, failed, succeeded)
func GetJobResult(jobName, namespace string ,clients clients.ClientSets) (string, error){
	resultJob, err := clients.KubeClient.BatchV1().Jobs(namespace).Get(jobName, metav1.GetOptions{})
	if err != nil {
		log.Errorf("error while waiting for kubernetes job: %v", err)
		return "",err
	}
	if resultJob.Status.Failed == 1 {
		return "failed", nil
	}
	if resultJob.Status.Succeeded == 1 {
		return "succeeded", nil
	}
	// job is still running
	return  "running", nil
}

// ParseJobResult
//
//returns: repeat if job is running and should continue so
//
//returns: error if job failed or timeout is reached,
func ParseJobResult(state string, isWithinTime bool) (repeat bool, err error){
	switch state {
	case "running":
		if isWithinTime {
			return true, nil
		}
		// timeout
		return false, errors.Errorf("did not end within time")
	case "succeeded":
		return  false, nil
	default:
		return false, errors.Errorf("failed")
	}
}