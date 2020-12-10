## Description
This directory holds all kubernetes YAML that needs to be applied to the running cluster.
But before we can do that, we should mint certificates that will allow gandalf to securely talk to
the API Server.

## Assumptions
The guide assumes that the running k8s is integrated Vault as we use Vault as secret engines.
This can be done by following [this guide](https://learn.hashicorp.com/tutorials/vault/kubernetes-sidecar)

All commands shown here are run from curdir.

## Steps to generate certificates.
We use [cloud flare's PKI](https://github.com/cloudflare/cfssl) to generate the requests.
Please install them using `brew install cfssl` if they are not present.

### Generate a signing key and CSR.
```
# This should generate server.csr and server-key.pem
cfssl genkey csr.json | cfssljson -bare server
```

### Create CertificateSigningRequest
```
cat <<EOF | kubectl apply -f -
apiVersion: certificates.k8s.io/v1
kind: CertificateSigningRequest
metadata:
  name: gandalf.default
spec:
  request: $(cat server.csr | base64 | tr -d '\n')
  signerName: kubernetes.io/kubelet-serving
  usages:
  - digital signature
  - key encipherment
  - server auth
EOF
```
Doing a `kubectl get csr` should say that the CSR is created and it is in pending state.
```
spremkumar [ancalagon.local] @  - [main] $ kubectl get csr
NAME              AGE   SIGNERNAME                      REQUESTOR       CONDITION
gandalf.default   5s    kubernetes.io/kubelet-serving   minikube-user   Approved,Issued
```

### Approve the certificate
`kubectl certificate approve gandalf.default`
```
spremkumar [ancalagon.local] @  - [main] $ kubectl get csr
NAME               AGE   SIGNERNAME                      REQUESTOR       CONDITION
gandalf.default   99s   kubernetes.io/kubelet-serving   minikube-user   Approved,Issued

```

### Download the approved certificate
`kubectl get csr gandalf.default -o jsonpath='{.status.certificate}' | base64 --decode > server.crt`

This should download a freshly minted `server.crt` in the current directory.

Along with the `server-key.pem` these will be used for TLSConfig for gandalf.

### Upload the generated certificate and key to Vault

Set up temporary port-forwarding to the Vault svc. This will enable us to upload the server certificates to remote
Vault from localhost
```
kubectl port-forward svc/vault 8200:8200
export VAULT_ADDR=http://localhost:8200
```

Enable v2 Vault secrets at path gandalf
```
vault secrets enable -path=gandalf kv-v2
vault kv put gandalf/secrets certificate=@server.crt key=@server-key.pem
```

This can be verified by doing a GET on the secrets
```
spremkumar [ancalagon.local] @  - [main] $ vault kv get gandalf/secrets
====== Metadata ======
Key              Value
---              -----
created_time     2020-11-23T04:27:08.685412Z
deletion_time    n/a
destroyed        false
version          2

======= Data =======
Key            Value
---            -----
certificate    <redacted>
key            <redacted>
```
### Authenticate Vault with Kubernetes

Exec into vault-0 container `kubectl exec -it vault-0 -- /bin/sh`

Enable k8s auth `vault auth enable kubernetes`

Configure the SA
```
vault write auth/kubernetes/config \
     token_reviewer_jwt="$(cat /var/run/secrets/kubernetes.io/serviceaccount/token)" \
     kubernetes_host="https://$KUBERNETES_PORT_443_TCP_ADDR:443" \
     kubernetes_ca_cert=@/var/run/secrets/kubernetes.io/serviceaccount/ca.crt
```

Create a Vault policy `admission-controller` that allows gandalf to read the secrets
```
vault policy write admission-controller - <<EOF
path "gandalf/data/secrets"{capabilities = ["read"]}
EOF
```

Create a k8s role** `admission-controller` which creates the SA binding with that name in default ns and bind the above Vault policy to it.
```
vault write auth/kubernetes/role/admission-controller \
      bound_service_account_names=admission-controller \
      bound_service_account_namespaces=default \
      policies=admission-controller \
      ttl=24h
```

Exit the `vault-0` container and kill the `kubectl port-forward` command.

**Now Vault is authenticated with k8s and ready to inject secrets!**

### Deploy gandalf
`kubectl apply -f gandalf.yaml`

### Patch the deployment to inject changes
`kubectl patch deployment gandalf --patch "$(cat vault-certs-inject.yaml)"`

On a successful injection we should see the following output.

Note the READY 2/2, vault injector has populated certs inside of `/vault/sece
rets` directory.
The injector patch uses Consul template to write `tls.crt` and `tls.key`
from the Vault secrets
```
spremkumar [ancalagon.local] @ ~/goDir/src/github.com/supriya-premkumar/gandalf - [main] $ kubectl get po
NAME                                    READY   STATUS    RESTARTS   AGE
gandalf-56d5589db5-dpbr4                2/2     Running   0          3m50s
vault-0                                 1/1     Running   0          6m41s
vault-agent-injector-7dd448d6c4-m94js   1/1     Running   0          6m41s
```

## Verify gandalf is up and running with Vault certs:
```
spremkumar [ancalagon.local] @ ~/goDir/src/github.com/supriya-premkumar/gandalf - [main] $ kubectl logs gandalf-56d5589db5-dpbr4 gandalf-controller
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
```