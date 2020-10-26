The Prometheus lifecycle API is enabled in dev mode (when Sourcegraph is started by dev/start.sh or similar).
You can edit, add, delete rules in this directory and then send

```shell script
curl -X POST http://localhost:9090/-/reload
```

to your running Prometheus in your dev Sourcegraph instance and it will reload the configs with your changes.
