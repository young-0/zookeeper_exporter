# zookeeper_exporter [![CircleCI](https://circleci.com/gh/carlpett/zookeeper_exporter.svg?style=shield)](https://circleci.com/gh/carlpett/zookeeper_exporter) [![DockerHub](https://img.shields.io/docker/build/carlpett/zookeeper_exporter.svg?style=shield)](https://hub.docker.com/r/carlpett/zookeeper_exporter/)

A Prometheus exporter for Zookeeper 3.4+. It send the `mntr` command to a Zookeeper node and converts the output to Prometheus format. 

## Usage
Download the [latest release](https://github.com/carlpett/zookeeper_exporter/releases), pull [the Docker image](https://hub.docker.com/r/carlpett/zookeeper_exporter/) or follow the instructions below for building the source.

There is a `-help` flag for listing the available flags.

## Building from source
`go get -u github.com/carlpett/zookeeper_exporter` and then `make build`.

## Limitations
Due to the type of data exposed by Zookeeper's `mntr` command, it currently resets Zookeeper's internal statistics every time it is scraped. This makes it unsuitable for having multiple parallel scrapers.

## Modify

THis project from [carlpett/zookeeper_exporter](https://github.com/carlpett/zookeeper_exporter).

Change to prometheus exporter for zookeeper metrics.

### Kubernetes SD configurations

- zookeeper svc

```
apiVersion: v1
kind: Service
metadata:
  name: zookeeper-svc
spec:
  clusterIP: None
  ports:
  - name: tcp-2181
    port: 2181
    protocol: TCP
    targetPort: 2181
  sessionAffinity: None
  type: ClusterIP
status:
```

- prometheus k8s SD configure

```
- job_name: zookeeper-exporter
  honor_timestamps: true
  scrape_interval: 1m
  scrape_timeout: 10s
  metrics_path: /scrape
  scheme: http
  kubernetes_sd_configs:
  - role: endpoints
  relabel_configs:
  - source_labels: [__meta_kubernetes_endpoint_port_name]
    separator: ;
    regex: tcp-2181
    replacement: $1
    action: keep
  - source_labels: [__address__]
    separator: ;
    regex: (.*)
    target_label: __param_target
    replacement: $1
    action: replace
  - source_labels: [__param_target]
    separator: ;
    regex: (.*)
    target_label: instance
    replacement: $1
    action: replace
  - separator: ;
    regex: (.*)
    target_label: __address__
    replacement: zookeeper-exporter:8080
    action: replace
  - separator: ;
    regex: __meta_kubernetes_pod_label_(.+)
    replacement: $1
    action: labelmap
  - source_labels: [__meta_kubernetes_namespace]
    separator: ;
    regex: (.*)
    target_label: kubernetes_namespace
    replacement: $1
    action: replace
  - source_labels: [__meta_kubernetes_pod_name]
    separator: ;
    regex: (.*)
    target_label: kubernetes_name
    replacement: $1
    action: replace
  - source_labels: [__meta_kubernetes_pod_ip]
    separator: ;
    regex: (.*)
    target_label: kubernetes_pod_ip
    replacement: $1
    action: replace
```

then

- prometheus zookeeper-exporter endpoint is：

`http://zookeeper-exporter:8080/scrape?target=IP:PORT`

