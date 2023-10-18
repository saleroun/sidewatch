# SideWatch (Go Light Exporter)
This GitHub repository contains an application that monitors various services within a cluster and sends their metrics to Prometheus. The application is designed to check the availability and health of Redis, MongoDB, TDengine, RabbitMQ, and HTTP URLs, and collect metrics for monitoring and alerting purposes.
#### requirments: 

memory-> 60Mi

cpu-> 40m

# Features
Service Monitoring: Continuously monitor the availability and health of services within a cluster, including Redis, MongoDB, TDengine, RabbitMQ, and HTTP URLs.
Metric Collection: Collect metrics such as latency, response time, error rate, and other relevant data from the monitored services.
Prometheus Integration: Send collected metrics to Prometheus.
Configurable Monitoring Targets: Easily configure the services to monitor, their endpoints, and any additional parameters required for monitoring using the provided configuration file.

# Deploy

To deploy this project run

```bash
docker buildx build -t private-repo/sidewatch .
```
### Docker compose


```bash
docker-compose up -d 
```
### Deployment on kubernetes

before  deploy on k8s, create configMap from configmap-env.yaml 

```bash
cd kubernetes-deployment

kubectl apply -f sidewatch-deployment.yaml -n namespace
```

