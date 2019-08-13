package etcdbackup

import (
	"text/template"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
)

var configMapTemplate = template.Must(template.New("").Parse(`
metadata:
  name: ` + op.EtcdBackupAppName + `
  namespace: kube-system
data:
  config.yml: |
    backup-dir: /etcdbackup
    listen: 0.0.0.0:8080
    rotate: {{ .Rotate }}
    etcd:
      endpoints: 
        - https://cke-etcd:2379
      tls-ca-file: /etcd-certs/ca
      tls-cert-file: /etcd-certs/cert
      tls-key-file: /etcd-certs/key
`))

var secretTemplate = template.Must(template.New("").Parse(`
metadata:
  name: ` + op.EtcdBackupAppName + `
  namespace: kube-system
data:
  cert: "{{ .Cert }}"
  key: "{{ .Key }}"
  ca: "{{ .CA }}"
`))

var podTemplate = template.Must(template.New("").Parse(`
metadata:
  name: ` + op.EtcdBackupAppName + `
  namespace: kube-system
  labels:
    app.kubernetes.io/name: ` + op.EtcdBackupAppName + `
spec:
  serviceAccountName: cke-etcdbackup
  containers:
  - name: etcdbackup
    image: ` + cke.ToolsImage.Name() + `
    command:
      - /usr/local/cke-tools/bin/etcdbackup
    args: ['-config', '/config/config.yml']
    ports:
      - containerPort: 8080
    volumeMounts:
      - mountPath: /etcd-certs
        name: etcd-certs
      - mountPath: /etcdbackup
        name: etcdbackup
      - mountPath: /config
        name: config
    securityContext:
      readOnlyRootFilesystem: true
      fsGroup: 10000
  volumes:
  - name: etcd-certs
    secret:
      secretName: ` + op.EtcdBackupAppName + `
      defaultMode: 0444
  - name: etcdbackup
    persistentVolumeClaim:
      claimName: {{ .PVCName }}
  - name: config
    configMap:
      name: ` + op.EtcdBackupAppName + `
      items:
        - key: config.yml
          path: config.yml
      defaultMode: 0644
`))

var cronJobTemplate = template.Must(template.New("").Parse(`
metadata:
  name: ` + op.EtcdBackupAppName + `
  namespace: kube-system
spec:
  jobTemplate:
    spec:
      template:
        spec:
          serviceAccountName: cke-etcdbackup
          securityContext:
            runAsUser: 10000
          containers:
            - name: etcdbackup
              image: ` + cke.ToolsImage.Name() + `
              command:
                - curl
                - -s
                - -XPOST
                - http://etcdbackup:8080/api/v1/backup
          restartPolicy: Never
  schedule:  "{{ .Schedule }}"
`))

var serviceText = `
metadata:
  name: ` + op.EtcdBackupAppName + `
  namespace: kube-system
  labels:
    app.kubernetes.io/name: ` + op.EtcdBackupAppName + `
spec:
  ports:
    - port: 8080
      protocol: TCP
      targetPort: 8080
  selector:
    app.kubernetes.io/name: ` + op.EtcdBackupAppName + `
  type: NodePort`
