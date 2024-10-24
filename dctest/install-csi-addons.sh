#!/bin/sh -x

git clone https://github.com/csi-addons/kubernetes-csi-addons.git bin/kubernetes-csi-addons
./dcscp -r bin/kubernetes-csi-addons/ boot-0:/home/cybozu/
./dcssh boot-0 -- KUBECONFIG=.kube/user-config kubectl create -f kubernetes-csi-addons/deploy/controller/crds.yaml
./dcssh boot-0 -- KUBECONFIG=.kube/user-config kubectl create -f kubernetes-csi-addons/deploy/controller/setup-controller.yaml
./dcssh boot-0 -- KUBECONFIG=.kube/user-config kubectl create -f kubernetes-csi-addons/deploy/controller/rbac.yaml

