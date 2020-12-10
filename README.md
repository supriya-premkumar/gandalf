# gandalf
gandalf is a configurable k8s admission controller maintaining an allow-list of labels.
For everyone else he will say "You shall not pass"

# Building gandalf
`make compile` Runs all checks, linting, formatting and builds the static binary.
`make container` Builds a docker image

# Bootstrapping gandalf.
Detailed instructions on how to bootstrap k8s with vault for gandalf is documented [here](https://github.com/supriya-premkumar/gandalf/blob/main/k8s/README.md)

# Enabling gandalf
gandalf uses `kind:ValidatingWebhookConfiguration` to enforce Admission decisions.

* Get the CA Bundle that needs to be a part of the webhook ca
```
export CA_BUNDLE=$(kubectl config view --raw --minify --flatten -o jsonpath='{.clusters[].cluster.certificate-authority-data}')

cat <<EOF | kubectl apply -f -
apiVersion: admissionregistration.k8s.io/v1beta1
kind: ValidatingWebhookConfiguration
metadata:
  name: gandalf
webhooks:
  - name: gandalf.review.match-labels
    sideEffects: None
    rules:
      - apiGroups:      ["*"]
        apiVersions:    ["*"]
        operations:     ["CREATE", "UPDATE"]
        resources:      ["pods", "deployments", "replicasets", "statefulset", "services", "daemonset", "job", "cronjob",
                         "services", "ingresses"]
    failurePolicy: Fail
    clientConfig:
      service:
        name: gandalf
        namespace: default
        path: "/v1/api/admission/review"
      caBundle: $CA_BUNDLE
EOF
```

# Steps to iteratively run.
* Clean up the ValidatingWebhookConfiguration
`kubectl delete ValidatingWebhookConfiguration gandalf`

* Delete gandalf resources
`kubectl delete -f gandalf.yaml`

* Create gandalf resources
`kubectl apply -f gandalf.yaml`

* Patch gandalf to inject secrets from Vault.
`kubectl patch deployment gandalf --patch "$(cat vault-certs-inject.yaml)"`

* Apply the ValidatingWebhookConfiguration
```
export CA_BUNDLE=$(kubectl config view --raw --minify --flatten -o jsonpath='{.clusters[].cluster.certificate-authority-data}')

cat <<EOF | kubectl apply -f -
apiVersion: admissionregistration.k8s.io/v1beta1
kind: ValidatingWebhookConfiguration
metadata:
  name: gandalf
webhooks:
  - name: gandalf.review.match-labels
    sideEffects: None
    rules:
      - apiGroups:      ["*"]
        apiVersions:    ["*"]
        operations:     ["CREATE", "UPDATE"]
        resources:      ["pods", "deployments", "replicasets", "statefulset", "services", "daemonset", "job", "cronjob",
                         "services", "ingresses"]
    failurePolicy: Fail
    clientConfig:
      service:
        name: gandalf
        namespace: default
        path: "/v1/api/admission/review"
      caBundle: $CA_BUNDLE
EOF
```

# What to do when gandalf goes rogue.
This is not thoroughly tested and Validating Webhooks can have a significant load on the API server, while care is
taken to ensure that gandalf requests only a subset of the resources.
Please follow the follwing guidelines when running this.
1. Run in an isolate cluster. minikube will be ideal.
2. The YAMLs are deployed in the default namespace to make the deployment easier, but we should consider moving gandalf to a different ns and
restrict SA to request resources only from the said namespace.

If something goes wrong, deleting the `ValidatingWebhookConfiguration` should limit the blast radius.

# gandalf in action
```
kubectl logs -f gandalf-5b6ccf9fdd-47q7b gandalf-controller
Dec 10 04:30:33.961 [INFO] [        main] Starting gandalf.
version: 1a8f993
config:
{
  "match-labels": {
    "fellowship": "yes"
  }
} [main.go:68][main()]
Dec 10 04:30:33.961 [INFO] [         api] Listening for admission reviews on 0.0.0.0:8443 [api.go:54][Start()]
Dec 10 04:30:33.962 [INFO] [        main] gandalf is ready to protect admission requests [main.go:86][main()]
Dec 10 04:33:01.165 [INFO] [  controller] Reviewing admission for kind:Deployment | name:nginx-deployment [admission.go:33][Review()]
Dec 10 04:33:01.185 [INFO] [  controller] Got Labels: map[name:balrog] [admission.go:99][Review()]
Dec 10 04:33:01.185 [INFO] [  controller] Not OK to admit kind:apps/v1, Kind=Deployment | name:nginx-deployment [admission.go:114][Review()]
Dec 10 04:33:01.185 [INFO] [         api] gandalf rejected apps/v1, Kind=Deployment [api.go:122][admissionHandler()]
Dec 10 04:34:44.670 [INFO] [  controller] Reviewing admission for kind:Service | name:nazgul-service [admission.go:33][Review()]
Dec 10 04:34:44.670 [INFO] [  controller] Checking for service [admission.go:75][Review()]
Dec 10 04:34:44.673 [INFO] [  controller] Got Labels: map[fellowship:no] [admission.go:99][Review()]
Dec 10 04:34:44.673 [INFO] [  controller] Not OK to admit kind:/v1, Kind=Service | name:nazgul-service [admission.go:114][Review()]
Dec 10 04:34:44.673 [INFO] [         api] gandalf rejected /v1, Kind=Service [api.go:122][admissionHandler()]
Dec 10 04:36:05.591 [INFO] [  controller] Reviewing admission for kind:Service | name:frodo [admission.go:33][Review()]
Dec 10 04:36:05.592 [INFO] [  controller] Checking for service [admission.go:75][Review()]
Dec 10 04:36:05.594 [INFO] [  controller] Got Labels: map[fellowship:yes] [admission.go:99][Review()]
Dec 10 04:36:05.594 [INFO] [  controller] OK to admit kind:Service | name:frodo [admission.go:105][Review()]
Dec 10 04:36:05.594 [INFO] [         api] gandalf admitted /v1, Kind=Service [api.go:119][admissionHandler()]
Dec 10 04:39:07.564 [INFO] [         api] Successfully stopped REST Server [api.go:139][Stop()]
```

# How to test
Admission Deny missing label deployment
```
kubectl apply -f denied-deployment.yaml
Error from server: error when creating "denied-deployment.yaml": admission webhook "gandalf.review.match-labels" denied the request: admission rejected by policy
```


Admission Deny incorrect label service
```
kubectl apply -f denied-service.yaml
Error from server: error when creating "denied-service.yaml": admission webhook "gandalf.review.match-labels" denied the request: admission rejected by policy
```

Admission Success correct labels
```
kubectl apply -f allowed-service.yaml
service/frodo created
```