# Overview

This is an example appliation and self-paced tutorial for implementing readiness checks properly in Kubernets and Go.  This Go application **does not drop connections when terminating** due to proper shutdown signal trapping and readiness checks.  **If you do not use readiness checks properly today, your service probably drops connections** for users when your pods are re-deployed or removed.

This follows the [best practices for Kubernetes](https://learnk8s.io/production-best-practices) for web services by implementing a [readiness probe](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/#define-readiness-probes) that removes the pod from the Kubernetes [endpoints](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#endpoints-v1-core) list once the check shows as 'failed'.  When an endpoint is removed, the Kubernetes cluster will reconfigure to remove it from all load balancing.  Only after that process completes can your pod be removed gracefully.

This should be combined with a pod disruption budget to restrict how many pods can be unavailable at one time as well as a pod anti-affinity policy to stripe your pods across nodes for best production resiliency.

## Order of operations

This is how graceful removal of pods from load balancers should look:

- The Kubernetes API is sent a delete command for a pod and changes the pod to the `Terminating` state
- The [kubelet](https://kubernetes.io/docs/reference/command-line-tools-reference/kubelet/) responsible for this pod instructs the [CRI](https://kubernetes.io/docs/concepts/architecture/cri/) to stop the containers in this pod
- The CRI sends a shutdown signal to the containerized processes
- The containerized process catches this signal gracefully
- The containerized process begins failing its **readiness** checks for enough time to have the pod removed from the endpoints list (default [30s](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/#configure-probes))
	- The containerized application *continues serving requests that find their way to it**
- The containerized process waits for the time it takes for readiness probes to fail, plus the time it takes for your Kubernetes cluster to reconfigure (should be less than 10 seconds)
	- This formula is: `readinessProbe.periodSeconds * readinessProbe.failureThreshold + 10s`
- Notice that `kubectl get pods` shows the `READY` colum as `0/1` indicating readiness probes are down
- The pod exits gracefully


## Example Spec

This spec is in this repo as `kubernetes.yaml`.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: graceful-shutdown-app
spec:
  replicas: 1
  selector:
    matchLabels:
      app: graceful-shutdown-app
  template:
    metadata:
      labels:
        app: graceful-shutdown-app
    spec:
      terminationGracePeriodSeconds: 60
      containers:
      - name: graceful-shutdown-app
        image: integrii/go-k8s-graceful-termination:latest
        livenessProbe:
          httpGet:
            path: /alive
            port: 8080
        readinessProbe:
          periodSeconds: 2
          failureThreshold: 3
          httpGet:
            path: /ready
            port: 8080
        ports:
        - containerPort: 8080
        resources:
          requests:
            memory: 128Mi
            cpu: 500m
          limits:
            cpu: 1
            memory: 1Gi
```



## Try it yourself

You can test this graceful shutdown yourself.  Clone this repo and try the following:

```
kubectl create ns graceful-termination
kubectl -n graceful-termination apply -f kubernetes.yaml
<wait for service to come online>
kubectl -n graceful-termination port-forward service/graceful-shutdown-app 8080 (in another terminal)
kubectl -n graceful-termination logs -f -l app=graceful-shutdown-app (in another terminal)
for i in `seq 1 100000`; do 
   curl -v http://localhost:8080 
done (in another terminal)
kubectl -n graceful-termination set env deployment/graceful-shutdown-app TEST=`date +%s` (this will cause a rolling update to the deployment)
watch kubectl -n graceful-termination get pods
<observe terminal doing curl tests>
```

You should see _no_ dropped connections during the rolling update, even though there is only one pod!


## Some Closing Notes

It is to common that I have seen applications not take care when being removed from the flow of traffic, resulting in connection failures.  Hopefully this clears things up.  This process has always existed, even with traditional load balncers, and in those situations it still is regular procedure to remove backends from the load balancer before bringing down those applications.

You also could alternatively do a graceful shutdown integration using [preStop](https://kubernetes.io/docs/concepts/containers/container-lifecycle-hooks/) hooks, which can be configured to send a web request to your application before it is sent a termination signal - but that approach wasn't covered here.