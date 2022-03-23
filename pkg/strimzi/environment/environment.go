package environment

import (
	experimentTypes "github.com/litmuschaos/litmus-go/pkg/strimzi/types"
	"github.com/litmuschaos/litmus-go/pkg/types"
	clientTypes "k8s.io/apimachinery/pkg/types"
	"strconv"
)

//GetENV fetches all the env variables from the runner pod
func GetENV(experimentDetails *experimentTypes.ExperimentDetails) {
	// initialize variables needed to properly execute litmus based part of experiment
	experimentDetails.Control = new(experimentTypes.Control)
	experimentDetails.Control.ExperimentName = types.Getenv("EXPERIMENT_NAME", "pod-delete")
	experimentDetails.Control.ChaosNamespace = types.Getenv("CHAOS_NAMESPACE", "litmus")
	experimentDetails.Control.EngineName = types.Getenv("CHAOSENGINE", "")
	experimentDetails.Control.ChaosDuration, _ = strconv.Atoi(types.Getenv("TOTAL_CHAOS_DURATION", "30"))
	experimentDetails.Control.ChaosInterval = types.Getenv("CHAOS_INTERVAL", "10")
	experimentDetails.Control.RampTime, _ = strconv.Atoi(types.Getenv("RAMP_TIME", "0"))
	experimentDetails.Control.ChaosLib = types.Getenv("LIB", "litmus")
	experimentDetails.Control.ChaosServiceAccount = types.Getenv("CHAOS_SERVICE_ACCOUNT", "")
	// Deprecated
	experimentDetails.Control.AppLabel = types.Getenv("APP_LABEL", "")
	experimentDetails.Control.AppKind = types.Getenv("APP_KIND", "")
	experimentDetails.Control.ChaosUID = clientTypes.UID(types.Getenv("CHAOS_UID", ""))
	experimentDetails.Control.InstanceID = types.Getenv("INSTANCE_ID", "")
	experimentDetails.Control.ChaosPodName = types.Getenv("POD_NAME", "")
	experimentDetails.Control.Force, _ = strconv.ParseBool(types.Getenv("FORCE", "false"))
	experimentDetails.Control.Delay, _ = strconv.Atoi(types.Getenv("STATUS_CHECK_DELAY", "2"))
	experimentDetails.Control.Timeout, _ = strconv.Atoi(types.Getenv("STATUS_CHECK_TIMEOUT", "180"))
	experimentDetails.Control.TargetPods = types.Getenv("TARGET_PODS", "")
	experimentDetails.Control.PodsAffectedPerc, _ = strconv.Atoi(types.Getenv("PODS_AFFECTED_PERC", "0"))
	experimentDetails.Control.Sequence = types.Getenv("SEQUENCE", "parallel")
	experimentDetails.Control.TargetContainer = types.Getenv("TARGET_CONTAINER", "")

	// other parts of experiment Separted by logical parts

	// Strimzi kafka
	experimentDetails.Kafka = new(experimentTypes.Kafka)
	experimentDetails.Kafka.Port = types.Getenv("KAFKA_PORT","")
	experimentDetails.Kafka.Service = types.Getenv("KAFKA_SERVICE","")


	experimentDetails.Kafka.KafkaInstancesName = types.Getenv("KAFKA_INSTANCE_NAME","")
	experimentDetails.Kafka.KafkaPartitionLeaderKill = types.Getenv("KAFKA_TOPIC_LEADER_KILL", "disable")


	// whole App
	experimentDetails.App = new(experimentTypes.App)
	experimentDetails.App.LivenessStream = types.Getenv("LIVENESS_STREAM","disable")
	experimentDetails.App.LivenessStreamJobsCleanup = types.Getenv("LIVENESS_STREAM_JOBS_CLEANUP","disable")
	experimentDetails.App.LivenessStreamTopicCleanup = types.Getenv("LIVENESS_STREAM_TOPIC_CLEANUP","disable")
	experimentDetails.App.Namespace = types.Getenv("APP_NAMESPACE", "")


	// Strimzi Topic
	experimentDetails.Topic = new(experimentTypes.Topic)
	experimentDetails.Topic.ReplicationFactor = types.Getenv("TOPIC_REPLICATION_FACTOR","1")
	experimentDetails.Topic.MinInSyncReplica = types.Getenv("TOPIC_MIN_IN_SYNC_REPLICAS","1")
	experimentDetails.Topic.Name = types.Getenv("TOPIC_NAME", "")

	// Producer
	experimentDetails.Producer = new(experimentTypes.Producer)
	experimentDetails.Producer.Acks = types.Getenv("PRODUCER_ACKS","all")
	experimentDetails.Producer.MessageCount = types.Getenv("PRODUCER_MESSAGE_COUNT","30")
	experimentDetails.Producer.MessageDelayMs = types.Getenv("PRODUCER_MESSAGE_DELAY_MS", "1000")
	experimentDetails.Producer.RequestTimeoutMs = types.Getenv("PRODUCER_REQUEST_TIMEOUT_MS", "6000")
	experimentDetails.Producer.MessageDeliveryTimeoutMs = types.Getenv("PRODUCER_MESSAGE_DELIVERY_TIMEOUT_MS", "30000")


	// Consumer
	experimentDetails.Consumer = new(experimentTypes.Consumer)
	experimentDetails.Consumer.TimeoutMs = types.Getenv("KAFKA_CONSUMER_TIMEOUT_MS","30000")
	experimentDetails.Consumer.MessageCount, _  = strconv.Atoi(types.Getenv("KAFKA_CONSUMER_MAX_MESSAGE_COUNT","30"))


	// Images for execution of cmds
	experimentDetails.Images = new(experimentTypes.Images)
	experimentDetails.Images.KafkaImage = types.Getenv("KAFKA_LIVENESS_IMAGE","litmuschaos/kafka-client:latest")
	experimentDetails.Images.ProducerImage = types.Getenv("STRIMZI_PRODUCER_IMAGE","quay.io/strimzi-examples/java-kafka-producer:latest")


	// Strimzi resources
	experimentDetails.Resources = new(experimentTypes.Resources)
	experimentDetails.Resources.Secrets = types.Getenv("RESOURCE_SECRETS", "")
	experimentDetails.Resources.Services = types.Getenv("RESOURCE_SERVICES", "")
	experimentDetails.Resources.ConfigMaps = types.Getenv("RESOURCE_CONFIG_MAPS", "")
	// experimentDetails.Resources.Resources provided as part of init

	// Strimzi kafka cluster
	experimentDetails.Strimzi = new(experimentTypes.Strimzi)
	// experimentDetails.Strimzi.Client: provided as part of init
	experimentDetails.Strimzi.StrimziKafkaClusterName = types.Getenv("STRIMZI_KAFKA_CLUSTER_NAME", "")
	experimentDetails.Strimzi.InternalListenerPortNumber, _ = strconv.Atoi(types.Getenv("STRIMZI_KAFKA_CLUSTER_LISTENER_PORT", "10001"))
	experimentDetails.Strimzi.InternalListenerName = types.Getenv("STRIMZI_KAFKA_CLUSTER_LISTENER_NAME", "litmus")

}
