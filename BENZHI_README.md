基于 Go 实现的化工过程 DCS 联锁控制系统项目，一款化工过程控制服务，完成反应釜温度/压力控制、进料联锁、紧急停车与工艺参数管理。

# ChemicalProcessDCS

ChemicalProcessDCS 是化工反应釜的集散联锁控制服务。系统按工艺曲线控制反应釜温度与压力，进料前做计量与联锁校验，超温超压时紧急停车并联动告警，反应放热时自动启动冷却，全程记录运行数据。

## 构建

```bash
go build -mod=vendor ./...
```

## 运行

```bash
go run -mod=vendor ./cmd/dcs -addr 127.0.0.1:8090 -dir ./data
```

启动后访问 http://127.0.0.1:8090/ 打开控制台页面。

## HTTP 接口

- `GET /healthz` 健康检查
- `GET /api/v1/reactors` 反应釜数量
- `GET /api/v1/reactors/{id}` 查询反应釜状态
- `POST /api/v1/reactors/{id}` 登记反应釜、下发读数或执行控制步
- `GET /api/v1/params` 查询工艺参数
- `POST /api/v1/params` 更新工艺参数
- `POST /api/v1/esd/trip` 紧急停车
- `POST /api/v1/alarms/restore` 停车恢复

## 状态机

- 反应：idle -> heating -> reacting -> cooling -> idle
- 联锁：free -> interlocked -> tripped -> reset
- 进料：idle -> metering -> feeding -> stopped
