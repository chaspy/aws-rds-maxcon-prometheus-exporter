# aws-rds-maxcon-prometheus-exporter
Prometheus Exporter for AWS RDS Max Connections

![image.png](image.png)

## How to run

### Local

```
$ go run main.go
```

### Binary

Get the binary file from [Releases](https://github.com/chaspy/aws-rds-maxcon-prometheus-exporter/releases) and run it.

### Docker

```
$ docker run chaspy/aws-rds-maxcon-prometheus-exporter:v0.1.0
```

## Metrics

```
$ curl -s localhost:8080/metrics | grep aws_custom_rds_max_connections
aws_custom_rds_max_connections{instance_class="db.r5.4xlarge",instance_identifier="postgres-api-production-a01"} 5000
aws_custom_rds_max_connections{instance_class="db.r5.4xlarge",instance_identifier="postgres-api-production-a02"} 5000
aws_custom_rds_max_connections{instance_class="db.r5.large",instance_identifier="test-postgres-production-a01"} 1800
aws_custom_rds_max_connections{instance_class="db.r5.large",instance_identifier="test-postgres-production-a02"} 1800
```

## IAM Role

The following policy must be attached to the AWS role to be executed.

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "VisualEditor0",
            "Effect": "Allow",
            "Action": [
                "rds:DescribeDBInstances",
                "rds:DescribeDBParameters",
            ],
            "Resource": "*"
        }
    ]
}
```

## Datadog Autodiscovery

If you use Datadog, you can use [Kubernetes Integration Autodiscovery](https://docs.datadoghq.com/agent/kubernetes/integrations/?tab=kubernetes) feature.
