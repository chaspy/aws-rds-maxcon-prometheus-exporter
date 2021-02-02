package postgresql

import (
	"fmt"
	"regexp"
	"strconv"
)

// Parse rawMaxConnections and calculate with instance class.
//
// Example of raw values:
// Aurora PostgreSQL: "LEAST({DBInstanceClassMemory/9531392},5000)"
// Aurora MySQL: "GREATEST({log(DBInstanceClassMemory/805306368)*45},{log(DBInstanceClassMemory/8187281408)*1000})"
// RDS Postgres: Same with Aurora PostgreSQL
// RDS MySQL: {DBInstanceClassMemory/12582880}
func GetPostgresMaxConnections(rawMaxConnections string, instanceClass *string) (int, error) {
	defaultRep := regexp.MustCompile(`(LEAST)\({(DBInstanceClassMemory)/(\d+)},(\d+)\)`)
	setRep := regexp.MustCompile(`(\d+)`)

	if defaultRep.MatchString(rawMaxConnections) {
		ret, err := GetDefaultPostgresMaxConnections(*instanceClass)
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
func GetDefaultPostgresMaxConnections(instanceClass string) (int, error) {
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
