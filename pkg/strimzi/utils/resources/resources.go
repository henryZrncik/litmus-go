package resources

import (
	"github.com/litmuschaos/litmus-go/pkg/clients"
	"github.com/litmuschaos/litmus-go/pkg/log"
	experimentTypes "github.com/litmuschaos/litmus-go/pkg/strimzi/types"
	"github.com/litmuschaos/litmus-go/pkg/utils/retry"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	kubeErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
	"time"
)

func ParseResourcesFromEnvs(experimentsDetails experimentTypes.ExperimentDetails) []experimentTypes.KubernetesResource {
	resourcesList := make([]experimentTypes.KubernetesResource, 0)
	targetServicesList := strings.Split(experimentsDetails.Resources.Services, ",")
	targetSecretList := strings.Split(experimentsDetails.Resources.Secrets, ",")
	targetConfigMapList := strings.Split(experimentsDetails.Resources.ConfigMaps, ",")

	for _, v := range targetServicesList {
		resourcesList = append(resourcesList, experimentTypes.KubernetesResource{v, experimentTypes.ServiceResourceType})
	}
	for _, v := range targetSecretList {
		resourcesList = append(resourcesList, experimentTypes.KubernetesResource{v, experimentTypes.SecretResourceType})
	}
	for _, v := range targetConfigMapList {
		resourcesList = append(resourcesList, experimentTypes.KubernetesResource{v, experimentTypes.ConfigMapResourceType})
	}
	return resourcesList

}

// DeleteAll deletes all provided resources, it does not consider problem if any of them already does not exist, for that is purpose of default health check, yet it logs it as a warning.
func DeleteAll(appNs string, resources []experimentTypes.KubernetesResource, force bool, clients clients.ClientSets) error {
	for _, resource := range resources {
		log.InfoWithValues("[Chaos]: deleting following resource", logrus.Fields{
			"Resource type": resource.Type,
			"Resource name:": resource.Name,
		})
		err := DeleteResource(appNs, resource, force, clients)
		if err != nil {
			return err
		}
	}
	return nil
}

func DeleteResource(appNs string, resource experimentTypes.KubernetesResource, force bool, clients clients.ClientSets) error {
	GracePeriod := int64(0)

	var err error
	switch resource.Type {
	case experimentTypes.ConfigMapResourceType:
		if force {
			err = clients.KubeClient.CoreV1().ConfigMaps(appNs).Delete(resource.Name, &metav1.DeleteOptions{GracePeriodSeconds: &GracePeriod})
		} else {
			err = clients.KubeClient.CoreV1().ConfigMaps(appNs).Delete(resource.Name, &metav1.DeleteOptions{})
		}
	case experimentTypes.ServiceResourceType:
		if force {
			err = clients.KubeClient.CoreV1().Services(appNs).Delete(resource.Name, &metav1.DeleteOptions{GracePeriodSeconds: &GracePeriod})
		} else {
			err = clients.KubeClient.CoreV1().Services(appNs).Delete(resource.Name, &metav1.DeleteOptions{})
		}
	case experimentTypes.SecretResourceType:
		if force {
			err = clients.KubeClient.CoreV1().Secrets(appNs).Delete(resource.Name, &metav1.DeleteOptions{GracePeriodSeconds: &GracePeriod})
		} else {
			err = clients.KubeClient.CoreV1().Secrets(appNs).Delete(resource.Name, &metav1.DeleteOptions{})
		}
	default:
		return errors.Errorf("unsupported resource type")
	}

	// deleting already deleted resource should only cause warning, as user only injects chaos way too often
	if kubeErrors.IsNotFound(err) {
		log.Warnf("[Chaos]: %v",err.Error())
	}

	if err != nil {
		log.Error(err.Error())
		return err
	}
	return nil
}

func HealthCheckAll(appNs string, resources []experimentTypes.KubernetesResource, timeout, delay int, clients clients.ClientSets) error {
	// used to log info only each 5th time resources are checked (used instead of time due to differences in lagging)
	retryCount := 0
	var isLogged bool = false

	return retry.
		Times(uint(timeout / delay)).
		Wait(time.Duration(delay) * time.Second).
		Try(func(attempt uint) error {

			var err error
			for _, resource := range resources {
				retryCount++
				// choose API call based on resource type
				switch resource.Type {
				case experimentTypes.ConfigMapResourceType:
					_, err = clients.KubeClient.CoreV1().ConfigMaps(appNs).Get(resource.Name, metav1.GetOptions{})
				case experimentTypes.ServiceResourceType:
					_, err = clients.KubeClient.CoreV1().Services(appNs).Get(resource.Name, metav1.GetOptions{})
				case experimentTypes.SecretResourceType:
					_, err = clients.KubeClient.CoreV1().Secrets(appNs).Get(resource.Name, metav1.GetOptions{})
				default:
					return errors.Errorf("Unsupported resource type provided")
				}

				if err != nil {
					// informing user about current problems but not each time,
					if retryCount % 5 == 0 {
						log.Warn(err.Error())
					}
					return errors.Errorf("Resource of type %v and name %v is not created", resource.Type, resource.Name)
				}
			}
			if !isLogged  {
				isLogged = true
				log.Infof("[Status]: all Strimzi resources are available")
			}
			return nil
		})
}


