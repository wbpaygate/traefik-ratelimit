# keeper-client
Для исключения ошибки `persistent cache is empty` рекомендуется использовать опцию предзагрузки кеша при старте вашего приложения `WithPreloadCache`.

В случаях ошибки обращения по сети в кипер, ошибка будет залогирована и будет попытка достать значения из inmemory кеша. 
Для прокидывания логера в клиент используйте опцию `WithLogger()`. 
В пакете `observability/v2/logger` есть адаптер логера `logger.Adapter{}`

В случае недоступности кипера и поднятии вашего приложения, у клиента есть возможность использовать холодный кеш (например redis).
Для этого используйте опцию `WithColdCache()`. Имплементация кеша есть в пакете `wbpay-go/packages/cache/redis`.
 
### Пример использования в скоупе pcidss. Походы в локальный кипер
```go
func main() {
    logger.SetupDefaultLogger("example")
    ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	
    keeper, err := keeperClient.New(
	keeperClient.Config{
            KeeperURL:               "https://keeper-ext.dev.paywb.com",
            ReqTimeout:              time.Second,
            CacheTTL:                time.Second * 10,
        },
        keeperClient.WithLogger(logger.NewAdapter(*logger.Logger(ctx))),
        keeperClient.WithColdCache(redisCache, "gateway", time.Minute),
        keeperClient.WithPreloadCache(),
    )
    if err != nil {
        logger.Logger(ctx).Fatal("cannot init keeper client", zap.Error(err))
    }
    if err = keeper.Start(ctx); err != nil {
        logger.Logger(ctx).Fatal("cannot start keeper client", zap.Error(err))
    }
    defer keeper.Stop()
}		

```
### Пример использования в apa/ipa. Поход в мегакипер
При походах в мегакипер будет отличаться конфиг клиента. Используйте такой
```go
keeperClient.Config{
    KeeperURL:               "https://keeper-common.dev.paywb.com",
    KeeperSettingsPath:      "api/v1/settings",
    KeeperSettingsAllPath:   "api/v1/settings",
    ReqTimeout:              time.Second,
    CacheTTL:                time.Second * 10,
}
```
