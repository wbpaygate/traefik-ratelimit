### Local Mode


The plugins must be placed in `./plugins-local` directory,
which should be in the working directory of the process running the Traefik binary.
The source code of the plugin should be organized as follows:

```
 ├── docker-compose.yml
 └── plugins-local
    └── src
        └── github.com
            └── ghnexpress
                └── traefik-ratelimit
                    ├── main.go
                    ├── vendor
                    ├── go.mod
                    └── ...

```

```yaml
# docker-compose.yml
version: "3.6"

services:
  traefik:
    image: traefik:v2.9.6
    container_name: traefik
    command:
      # - --log.level=DEBUG
      - --log.level=INFO
      - --api
      - --api.dashboard
      - --api.insecure=true
      - --providers.docker=true
      - --entrypoints.web.address=:80
      - --experimental.localPlugins.ratelimit.moduleName=gitlab-private.wildberries.ru/wbpay-go/traefik-ratelimit
    ports:
      - "80:80"
      - "8080:8080"
    networks:
      - traefik-network
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ./plugins-local/src/gitlab-private.wildberries.ru/wbpay-go/traefik-ratelimit:/plugins-local/src/gitlab-private.wildberries.ru/wbpay-go/traefik-ratelimit
    labels:
      - traefik.http.middlewares.rate-limit.plugin.ratelimit.keeperRateLimitKey=wbpay-ratelimits
      - traefik.http.middlewares.rate-limit.plugin.ratelimit.keeperURL=http://keeper-ext.wbpay.svc.k8s.wbpay-dev:8080
      - traefik.http.middlewares.rate-limit.plugin.ratelimit.keeperAdminPassword=Pa$sw0rd
      - traefik.http.middlewares.rate-limit.plugin.ratelimit.keeperReqTimeout=300s
  whoami:
    image: traefik/whoami
    container_name: simple-service
    depends_on:
      - traefik
    networks:
      - traefik-network
    labels:
      - traefik.enable=true
      - traefik.http.routers.whoami.rule=Host(`localhost`)
      - traefik.http.routers.whoami.entrypoints=web
      - traefik.http.routers.whoami.middlewares=rate-limit
networks:
  traefik-network:
    driver: bridge
```