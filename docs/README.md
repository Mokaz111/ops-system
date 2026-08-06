# 文档索引（现网对齐）

本目录维护「当前架构认知与实现边界」的全套设计文档；阶段性计划已完结归档。

## 现行文档

- [`云原生可观测性监控平台-总体设计文档v4.md`](./%E4%BA%91%E5%8E%9F%E7%94%9F%E5%8F%AF%E8%A7%82%E6%B5%8B%E6%80%A7%E7%9B%91%E6%8E%A7%E5%B9%B3%E5%8F%B0-%E6%80%BB%E4%BD%93%E8%AE%BE%E8%AE%A1%E6%96%87%E6%A1%A3v4.md) — **SaaS 目标架构**（Workspace + 监控/日志/告警域，唯一权威设计）
- `01-前端详细设计.md` — 嵌套侧栏域、WorkspaceSwitcher、告警子页、业务集群采集配置
- `02-后端详细设计.md` — 分层、路由、Workspace 成员 RBAC、残余 `tenant_id` 说明、错误码
- `03-部署架构详细设计.md` — VLogs / 多 Grafana / 多 K8s / Helm、敏感凭据、上线动作
- `04-数据流详细设计.md` — Workspace Provisioning、接入安装、采集配置、告警导入与 Webhook
- `05-告警引擎详细设计.md` — 平台一等告警（规则/事件/渠道/导入/静默路线图）+ VMRule
- `06-用户同步详细设计.md` — bootstrap、平台角色、Workspace 成员、Grafana Org 联动
- `07-运维监控设计.md` — 健康检查、审计表、告警建议、关键日志事件 key

## 术语约定

| 概念 | 说明 |
|---|---|
| Workspace / 工作空间 | 唯一租户边界 |
| 监控实例（Instance） | 共享池上的逻辑观测单元（勿称「指标空间」） |
| 日志实例（LogInstance） | VictoriaLogs 查询入口（勿称「日志空间」） |
| 平台角色 | `admin` / `user` |
| 成员角色 | `admin` / `member` / `viewer` |
| `tenant_id`（残余） | 存储/标签历史字段名，值为 Workspace UUID |

## 使用建议

- 设计与实现冲突时，**以代码为准**，并修正文档
- 各模块的运行/开发说明优先看：
  - `README.md`
  - `backend/README.md`
  - `frontend/README.md`
- 新增能力上线前请同步更新本目录对应文档（不要新增过渡性临时文件）
