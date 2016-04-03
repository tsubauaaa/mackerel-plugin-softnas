mackerel-plugin-softnas
======================

SoftNas(https://www.softnas.com/) custom metrics plugin for mackerel.io agent.

### Synopsis
```
mackerel-plugin-softnas [-cmd=<Path of softnas-cmd>] [-url=<URL of softnas-cmd>] [-user=<User of softnas-cmd>] [-password=<Password of softnas-cmd>]
```

### Example of mackerel-agent.conf
```
[plugin.metrics.softnas]
command = "path/to/mackerel-plugin-softnas"
```
