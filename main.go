package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type RDSInfo struct {
	DBInstanceIdentifier string
	DBInstanceClass      string
	MaxConnections       string
}

var (
	//nolint:gochecknoglobals
	maxcon = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "aws_custom",
		Subsystem: "rds",
		Name:      "max_connections",
		Help:      "Max Connections of RDS",
	},
		[]string{"instance_identifier", "instance_class", "max_connections"},
	)
)

func main() {
	interval, err := getInterval()
	if err != nil {
		log.Fatal(err)
	}

	prometheus.MustRegister(maxcon)

	http.Handle("/metrics", promhttp.Handler())

	go func() {
		ticker := time.NewTicker(time.Duration(interval) * time.Second)

		// register metrics as background
		for range ticker.C {
			err := snapshot()
			if err != nil {
				log.Fatal(err)
			}
		}
	}()
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func snapshot() error {
	maxcon.Reset()

	InstanceInfos, err := getRDSInstances()
	if err != nil {
		return fmt.Errorf("failed to read RDS Instance infos: %w", err)
	}
	log.Printf("%v\n", InstanceInfos)

	for _, InstanceInfo := range InstanceInfos {
		if InstanceInfo.MaxConnections == "0" {
			log.Printf("skip: max connection is 0. instance_identifier: %v, instance_class: %v", InstanceInfo.DBInstanceIdentifier, InstanceInfo.DBInstanceClass)
			break
		}

		labels := prometheus.Labels{
			"instance_identifier": InstanceInfo.DBInstanceIdentifier,
			"instance_class":      InstanceInfo.DBInstanceClass,
			"max_connections":     InstanceInfo.MaxConnections,
		}
		maxcon.With(labels).Set(1)
	}

	return nil
}

func getInterval() (int, error) {
	const defaultGithubAPIIntervalSecond = 300
	githubAPIInterval := os.Getenv("AWS_API_INTERVAL")
	if len(githubAPIInterval) == 0 {
		return defaultGithubAPIIntervalSecond, nil
	}

	integerGithubAPIInterval, err := strconv.Atoi(githubAPIInterval)
	if err != nil {
		return 0, fmt.Errorf("failed to read Datadog Config: %w", err)
	}

	return integerGithubAPIInterval, nil
}

func getRDSInstances() ([]RDSInfo, error) {
	var rawMaxConnections string

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	svc := rds.New(sess)
	input := &rds.DescribeDBInstancesInput{}

	RDSInstances, err := svc.DescribeDBInstances(input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe DB instances: %w", err)
	}

	RDSInfos := make([]RDSInfo, len(RDSInstances.DBInstances))

	for i, RDSInstance := range RDSInstances.DBInstances {
		for _, DBParameterGroup := range RDSInstance.DBParameterGroups {
			rawMaxConnections, err = getRawMaxConnections(DBParameterGroup.DBParameterGroupName)
			if err != nil {
				return nil, fmt.Errorf("failed to get Parameter Group: %w", err)
			}
		}

		maxConnections, err := getMaxConnections(rawMaxConnections, RDSInstance.DBInstanceClass)
		if err != nil {
			log.Printf("skip: failed to get max connections: %v", err)
			// break
		}

		RDSInfos[i] = RDSInfo{
			DBInstanceIdentifier: *RDSInstance.DBInstanceIdentifier,
			DBInstanceClass:      *RDSInstance.DBInstanceClass,
			MaxConnections:       strconv.Itoa(maxConnections),
		}
	}

	return RDSInfos, nil
}

// Parse rawMaxConnections and calculate with instance class.
//
// Example of raw values:
// Aurora PostgreSQL: "LEAST({DBInstanceClassMemory/9531392},5000)"
// Aurora MySQL: "GREATEST({log(DBInstanceClassMemory/805306368)*45},{log(DBInstanceClassMemory/8187281408)*1000})"
func getMaxConnections(rawMaxConnections string, instanceClass *string) (int, error) {
	defaultRep := regexp.MustCompile(`(LEAST)\({(DBInstanceClassMemory)/(\d+)},(\d+)\)`)
	setRep := regexp.MustCompile(`(\d+)`)

	if defaultRep.MatchString(rawMaxConnections) {
		ret, err := getDefaultMaxConnections(*instanceClass)
		if err != nil {
			return 0, fmt.Errorf("failed to get default max connections: %w", err)
		}
		return ret, nil
	} else if setRep.MatchString(rawMaxConnections) {
		v := setRep.FindAllStringSubmatch(rawMaxConnections, -1)
		ret, _ := strconv.Atoi(v[0][0])
		return ret, nil
	}

	return 0, nil
}

// Aurora PostgreSQL: "LEAST({DBInstanceClassMemory/9531392},5000)"
// Default is set to this value for all instance classes.
// Note that the DBInstance Class Memory, which is 5000, is,
// DBInstanceClassMemory = 5000 * 9531392(Byte) = 47656960000(Byte)
//                                              = 47.65696(GB)
// In other words, for instances with a memory size larger than 47.65696 GB,
// max_connection is 5000.
// ref: https://aws.amazon.com/rds/instance-types/
func getDefaultMaxConnections(instanceClass string) (int, error) {
	auroraPostgresMaxcon := map[string]int{
		"db.r4.large":    1600, // Memory  15.25 GB
		"db.r4.xlarge":   3200, // Memory  30.5  GB
		"db.r4.2xlarge":  5000, // Memory  61    GB
		"db.r4.4xlarge":  5000, // Memory 122    GB
		"db.r4.8xlarge":  5000, // Memory 244    GB
		"db.r4.16xlarge": 5000, // Memory 488    GB
		"db.r5.large":    1800, // Memory  16    GB
		"db.r5.xlarge":   3600, // Memory  32    GB
		"db.r5.2xlarge":  5000, // Memory  64    GB
		"db.r5.4xlarge":  5000, // Memory 128    GB
		"db.r5.8xlarge":  5000, // Memory 256    GB
		"db.r5.12xlarge": 5000, // Memory 384    GB
		"db.r5.16xlarge": 5000, // Memory 384    GB
		"db.r5.24xlarge": 5000, // Memory 768    GB
		"db.m4.large":    900,  // Memory   8    GB
		"db.m4.xlarge":   1800, // Memory  16    GB
		"db.m4.2xlarge":  3600, // Memory  32    GB
		"db.m4.4xlarge":  5000, // Memory  64    GB
		"db.m4.10xlarge": 5000, // Memory 160    GB
		"db.m4.16xlarge": 5000, // Memory 256    GB
		"db.m5.large":    900,  // Memory   8    GB
		"db.m5.xlarge":   1800, // Memory  16    GB
		"db.m5.2xlarge":  3600, // Memory  32    GB
		"db.m5.4xlarge":  5000, // Memory  64    GB
		"db.m5.8xlarge":  5000, // Memory 128    GB
		"db.m5.12xlarge": 5000, // Memory 192    GB
		"db.m5.16xlarge": 5000, // Memory 256    GB
		"db.m5.24xlarge": 5000, // Memory 384    GB
		"db.t2.micro":    125,  // Memory   1    GB
		"db.t2.small":    250,  // Memory   2    GB
		"db.t2.medium":   450,  // Memory   4    GB
		"db.t2.large":    900,  // Memory   8    GB
		"db.t2.xlarge":   1800, // Memory  16    GB
		"db.t2.2xlarge":  3600, // Memory  32    GB
		"db.t3.micro":    125,  // Memory   1    GB
		"db.t3.small":    250,  // Memory   2    GB
		"db.t3.medium":   450,  // Memory   4    GB
		"db.t3.large":    900,  // Memory   8    GB
		"db.t3.xlarge":   1800, // Memory  16    GB
		"db.t3.2xlarge":  3600, // Memory  32    GB
	}

	ret := auroraPostgresMaxcon[instanceClass]
	if ret == 0 {
		return 0, fmt.Errorf("instance class %v is not supported", instanceClass)
	}

	return ret, nil
}

func getRawMaxConnections(parameterGroupName *string) (string, error) {
	var ParameterInfos []*rds.DescribeDBParametersOutput
	var rawMaxConenctions string

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	svc := rds.New(sess)
	input := &rds.DescribeDBParametersInput{
		DBParameterGroupName: parameterGroupName,
	}

	for {
		result, err := svc.DescribeDBParameters(input)
		if err != nil {
			return "", fmt.Errorf("failed to describe DB instances: %w", err)
		}

		ParameterInfos = append(ParameterInfos, result)

		// pagination
		if result.Marker == nil {
			break
		}
		input.SetMarker(*result.Marker)
	}

	for _, ParameterInfo := range ParameterInfos {
		for _, Parameter := range ParameterInfo.Parameters {
			if *Parameter.ParameterName == "max_connections" {
				rawMaxConenctions = *Parameter.ParameterValue
			}
		}
	}

	return rawMaxConenctions, nil
}
