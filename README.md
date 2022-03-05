# Overview

This is an example Go appliation and self-paced tutorial for implementing the graceful shutdown of Kubernetes pods.  This Go application **does not drop connections when terminating** due to proper shutdown signal trapping.  **If your app does not capture shutdown signals properly today, your service probably drops connections** for users when your pods are re-deployed or removed for any reason.

![image](https://user-images.githubusercontent.com/98695/156859137-1c2a5b0b-6a97-4400-8eee-65c4cb8a97c4.png)

This follows the [best practices for Kubernetes](https://learnk8s.io/production-best-practices) web services by implementing shutdown signal capturing that keeps the pod alive while connections are still ariving to it. Once a pod goes into the `Terminating` state, the pod is removed from the Kubernetes [endpoints](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#endpoints-v1-core) list.  When an endpoint is removed, the Kubernetes cluster will reconfigure to remove it from all load balancing.  Only after that process completes can your pod be removed gracefully.  You can find [detailed documentation from Kubernetes](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#pod-termination) about this.

Check out the `go` code [here](https://github.com/integrii/go-k8s-graceful-termination/blob/main/cmd/app/main.go)

This should be combined with a pod disruption budget to restrict how many pods can be unavailable at one time as well as a pod anti-affinity policy to stripe your pods across nodes for best production resiliency.

## Order of operations

This is how graceful removal of pods from load balancers should look:

- The Kubernetes API is sent a delete command for a pod and changes the pod to the `Terminating` state
- The [kubelet](https://kubernetes.io/docs/reference/command-line-tools-reference/kubelet/) responsible for this pod instructs the [CRI](https://kubernetes.io/docs/concepts/architecture/cri/) to stop the containers in this pod
- The CRI sends a shutdown signal to the containerized processes
- The containerized process catches this signal gracefully
  - The containerized application **continues serving requests that find their way to it**
  - This should be less than 10 seconds
	- This must be less than the `terminationGracePeriodSeconds` of your pod
- The Kubernetes cluster reconfigures to remove the pod from the flow of service traffic 
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
        ports:
        - containerPort: 8080
        resources:
          requests:
            memory: 128Mi
            cpu: 500m
          limits:
            cpu: 1
            memory: 1Gi
---
apiVersion: v1
kind: Service
metadata:
  name: graceful-shutdown-app
spec:
  ports:
  - name: "8080"
    port: 8080
    protocol: TCP
    targetPort: 8080
  selector:
    app: graceful-shutdown-app
  type: NodePort
```



## Try it yourself

You can test this graceful shutdown yourself.  Clone this repo and try the following:

```
kubectl create ns graceful-termination
kubectl -n graceful-termination apply -f https://raw.githubusercontent.com/integrii/go-k8s-graceful-termination/main/kubernetes.yaml
<wait for service to come online>
kubectl -n graceful-termination port-forward service/graceful-shutdown-app 8080 (in another terminal)
kubectl -n graceful-termination logs -f -l app=graceful-shutdown-app (in another terminal)
for i in `seq 1 100000`; do 
   curl -v http://localhost:8080 
done (in another terminal)
kubectl -n graceful-termination set env deployment/graceful-shutdown-app TEST=`date +%s` (this will cause a rolling update to the deployment)
watch kubectl -n graceful-termination get pods
<observe terminal doing curl tests>
kubectl delete namespace graceful-termination (when you're done with everything)
```

You should _not see_ dropped connections during the rolling update, even though there is only one pod!


## Some Closing Notes

It is too common that I have seen applications not take care when being removed from the flow of traffic, resulting in connection failures.  Hopefully this clears things up.  This process has always existed, even with traditional load balncers, and in those situations it remains a regular procedure to remove backends from the load balancer before bringing down those applications.

You also could alternatively do a graceful shutdown integration using [preStop](https://kubernetes.io/docs/concepts/containers/container-lifecycle-hooks/) hooks, which can be configured to send a web request to your application before it is sent a termination signal - but that approach wasn't covered here.
