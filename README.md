rate limit config keeper:

```
{
  "limits": [
    {
      "rules": [
        {"endpointpat": "/api/v2/**/methods",      "headerkey": "aa-bb", "headerval": "AsdfG"},
      ],
      "limit": 100
    }
    {
      "rules": [
        {"endpointpat": "/api/v2/methods"},
        {"endpointpat": "/api/v2/*/aa/**/methods"}
      ],
      "limit": 100
    }
  ]
}
```

Данные настройки предназначкены для конфигурирования лимитов скорости обработки запросов в зависимости от пути и содержимого заголовка запроса
где:
  - rules:
    правила выбора запросов для лимитирования скорости. для применеия лимита дочтаточно соответствия хотя бы одному правилу
    в любом правиле должны быть указан "endpointpat" и/или  "headerkey" и "headerval"
    где:
      - endpointpat:
        - патерн для сравнения пути из запроса может содержать символы * это означает эта часть пути запроса может содержать любое значение
        - после нахождения символов ** остальная часть пути будет сравниваться с хвостом пути
          например патерну: ```/api/v2/**/methods``` будут соответствовать пути запроса которые начинаются с /api/v2 и заканчивается на /methods остальные части пути при сравнении будут проигнорированы
        - патерн может содержать в конце символ ```$``` это означает, что путь запроса должен иметь определенную длинну в частях пути
          например патерн: ```/api/v2/*/*/methods$``` будет соответствовать пути запроса начинающегося с /api/v2 далее две следующие части не имеют значения и последняя часть methods т.е. путь должен будет состоять из 5 частей
          а паттерн: ```/api/v2/*/*/methods``` будет соответствовать пути запроса начинающегося с /api/v2 далее две следующие части не имеют значения и следующая часть methods дальныйшие части пути не важны 
      - headerkey и headerval:
        эти части правила сравниваются с ключами из заголовка запроса, то есть заголовок запроса должен содержать указаный ключ и соответствующее значение и ключи и значения сравниваются вне зависимости от регистра символов
        если оба или одно из этих значений не указаны, то они в проверках не участвуют
  - limit:
    лимит скорости в запросах в секунду. на запросы сверх лимита будет отправлен ответ со статусом: 429 Too Many Requests


Тестирование:
  должно заключаться в записи в keeper json-а c описаными выше параметрами и ключем keeper = keeperRateLimitKey из настроек middleware (будут обговорены при создании стенда),
  далее дождаться что конфигурация будет загужена в плагин traefik. загрузка происходит каждые 30 секунд.
  далее направлять в сторону ресурса запросы. запрос должен будет состоять из статической часть например http://nginx.k8s.local (будет обговорено при создании стенда) 
  и изменяемой части пути запроса и содержать ключи и значения заголовка или не содержать их для проверки работы правил в соответствии с описанием настроек см. выше.
  запросы которые соответствуют правилам и выходят за лимиты указанные в настройках должен приходить ответ состатусом 429 Too Many Requests
  далее повторять данные действия изменяя настройки и отправляемые запросы и частоту отправки запросов


Тестовые кейсы:
  - сделать првила и посмотреть что срабатывают так как положено
  - сделать правило и в момент работы изменить по правилу лимит, и посмотреть что лимит по запросам изменился без перебоев, то есть нет запросов не получивших ответа
    по ошибке передачи и.т.п
  - померить какое максиматьное количество запросов вообще возможно обработать без перебоев когда на каждый запрос приходит либо положительный ответ дибо отрицательный
    и нет запросов без ответа или с ошибкой ввода вывода

Проверку проводить возможно следующим образом в течении минуты например отправлять в адрес запросы с какойто частотой возможно даже с максимальной
и по секундам записывать в три переменных значения пропущеных не пропущеных запросов и запросов без ответа 
далее сравнить это с этолоном и выдать таблицу по секундам.







настройки kubectl
чтобы не забыть


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

middleware:

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
```


ingress:

```
      middlewares:
        - name: traefik-ratelimit
        namespace: traefik-v2
```


parameters:

```
  - keeperRateLimitKey=ratelimits
  - keeperURL=http://keeper:8080
  - keeperAdminPassword=Pa$sw0rd
  - keeperReqTimeout=300s
  - ratelimitPath=cfg/ratelimit.json

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
