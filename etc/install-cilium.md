How to install Cilium
=====================

```console
@boot-0
$ ckecli etcd user-add cilium cilium
$ ckecli etcd issue cilium --output=file
$ kubectl create secret generic -n kube-system cilium-etcd-secrets \
       --from-file=etcd-client-ca.crt=etcd-ca.crt \
       --from-file=etcd-client.key=etcd-cilium.key \
       --from-file=etcd-client.crt=etcd-cilium.crt
$ kubectl apply -f cilium-external-etcd.yaml
```
