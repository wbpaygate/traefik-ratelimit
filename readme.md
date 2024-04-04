### Local Mode


The plugins must be placed in `./plugins-local` directory,
which should be in the working directory of the process running the Traefik binary.
The source code of the plugin should be organized as follows:

```
 └── plugins-local
    └── src
        └── gitlab-private.wildberries.ru
            └── wbpay-go
                └── traefik-ratelimit
                    ├── main.go
                    ├── go.mod
                    └── ...
```

настройки kubectl

middleware:

k edit middleware traefik-ratelimit -n traefik-v2

```
apiVersion: traefik.containo.us/v1alpha1
kind: Middleware
metadata:
  name: traefik-ratelimit
  namespace: traefik-v2
spec:
  plugin:
    ratelimit:
      keeperRateLimitKey: ratelimits
      keeperURL: http://keeper-ext.wbpay.svc.k8s.wbpay-dev:8080
      keeperAdminPassword: pas$W0rd
      keeperReqTimeout: 300s
      ratelimitPath: ../cfg/ratelimit.json
      keeperReloadInterval: 30s

```

ingress:

```
      middlewares:
        - name: traefik-ratelimit
        namespace: traefik-v2
```

pvc:

```
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: traefik
  namespace: traefik-v2
spec:
  storageClassName: manual
  accessModes:
    - ReadWriteMany
  resources:
    requests:
      storage: 3Gi
```
pv:

```
apiVersion: v1
kind: PersistentVolume
metadata:
  name: traefik
  namespace: traefik-v2
  labels:
    type: local
spec:
  storageClassName: manual
  capacity:
    storage: 10Gi
  accessModes:
    - ReadWriteMany
  hostPath:
    path: "/path/to/traefik"
```

deployments:
```
    spec:
      containers:
      - args:
        - --global.sendanonymoususage
        - --entrypoints.traefik.address=:9000/tcp
        - --entrypoints.web.address=:8000/tcp
        - --entrypoints.websecure.address=:8443/tcp
        - --api.dashboard=true
        - --ping=true
        - --providers.kubernetescrd
        - --providers.kubernetesingress
        - --providers.kubernetescrd.allowCrossNamespace=true
        - --entrypoints.websecure.http.tls=true
        - --log.level=DEBUG
        - --experimental.localPlugins.ratelimit.moduleName=github.com/kav789/traefik-ratelimit


        volumeMounts:
        - mountPath: /plugins-local
          name: plugins

      volumes:
      - name: plugins
        persistentVolumeClaim:
          claimName: traefik

```
заливка нового плагина :
```
kubectl exec -it -n traefik-v2 $(kubectl get pod -n traefik-v2 | grep Runn | awk '{print $1}') -- mkdir -p /plugins-local/src/github.com/kav789
kubectl cp /home/kav/go/src/github.com/kav789/traefik-ratelimit traefik-v2/$(kubectl get pod -n traefik-v2 | grep Runn | awk '{print $1}'):/plugins-local/src/github.com/kav789/ -c 'traefik'
kubectl delete po $(kubectl get pod -n traefik-v2 | grep Runn | awk '{print $1}') -n traefik-v2
sleep 3
kubectl delete po $(kubectl get pod -n default | grep Runn | awk '{print $1}') -n default
sleep 3
kubectl logs $(kubectl get pod -n traefik-v2 | grep Runn | awk '{print $1}') -n traefik-v2  | grep error
```
