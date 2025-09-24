1. 目前系统读取配置可执行项文件是从默认从 config.yaml 中读取的，帮我新增一个配置文件用来全局管理多个可执行配置文件全路径，并且记录最后一次打开的配置文件，下次启动从最后一次打开的配置文件进行读取
2. 新增 worldPaths 配置在可执行文件中可以使用${xxx} 进行设置，读取配置文件后进行替换
3. 新增原生菜单项，配置文件的下来项是configFile 中配置的名称名称，默认勾选lastOpenedFile 的值，点击切换之后变更勾选，并且重新刷新配置文件，并刷新应用
```yaml
appId: com.runner
windowsName: 运行器
configFile:
    - /Users/ll/Documents/workspace/quick_cmd/ui/xyj.yaml
lastOpenedFile: /Users/ll/Documents/workspace/quick_cmd/ui/xyj.yaml
workPaths:
    ACC-CLOUD: /Users/ll/Documents/workspace/xyj-acc-cloud
    MVM: /Users/ll/Documents/Maven/apache-maven-3.6.1/bin/mvn
    XYJ-ACC: /Users/ll/Documents/workspace/xyjacc
    XYJ-ACC-CRON: /Users/ll/Documents/workspace/xyjacc-cron
```