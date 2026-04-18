# TCOP 对齐改造实施计划（已完结归档）

> **状态：已完成。** 本文档为历史阶段性工作计划的归档，保留以便追溯当时的设计决策与里程碑划分。
> 现行设计与能力边界请以以下文档为准：
> - [`云原生可观测性监控平台-总体设计文档v3.md`](./%E4%BA%91%E5%8E%9F%E7%94%9F%E5%8F%AF%E8%A7%82%E6%B5%8B%E6%80%A7%E7%9B%91%E6%8E%A7%E5%B9%B3%E5%8F%B0-%E6%80%BB%E4%BD%93%E8%AE%BE%E8%AE%A1%E6%96%87%E6%A1%A3v3.md)
> - `01-前端详细设计.md` / `02-后端详细设计.md` / `03-部署架构详细设计.md` / `04-数据流详细设计.md` / `05-告警引擎详细设计.md` / `06-用户同步详细设计.md` / `07-运维监控设计.md`

## 一、改造目标（回顾）

对齐腾讯云可观测平台（TCOP）形态，将现有控制面扩展为：

- 监控管理（VM 实例为中心，4 Tab 详情）
- 日志管理（VictoriaLogs）
- Grafana 多主机管理 + Dashboard 管理
- 接入中心（采集 + 告警 + 大盘三合一模板市场）
- 指标库（指标字典 + 模板关联）

**核心替换**：全部使用 VictoriaMetrics 栈，不引入 Prometheus 运行时。

## 二、关键决策（已落地）

| 决策项 | 结论 | 落地位置 |
|---|---|---|
| 日志引擎 | VictoriaLogs（Helm/VLogs CR） | `model.LogInstance` + `service.LogInstanceService` |
| 告警下发 | 模板内 `alertTargets` 声明，M2 仅 VMRule，N9E 占位 | `integration.applier.vmRuleApplier` + `n9e` noop |
| 模板存储 | PostgreSQL，版本化 | `ops_integration_templates` + `ops_integration_template_versions` |
| 指标来源 | 自动解析 collector / dashboard，允许手工覆盖 | `metric_parser_service`，`Metric.ManualOverride` |
| 安装粒度 | 按 VM 监控实例，1 模板 × 1 实例 = 1 安装 | `uk_install_instance_tpl_active where deleted_at IS NULL` |
| Grafana 归属 | 平台共享 + 租户自带 | `ops_grafana_hosts` + `grafanaResolver` |
| 实例详情 Tab | 基本信息 / 数据采集 / Dashboard / 告警 | `pages/InstanceDetail/index.tsx` |
| 告警菜单 | 移除独立菜单，入口收敛到 InstanceDetail | `appRoutes.ts` 中 `alerts` 不在 sidebar |

## 三、里程碑完成情况

| 阶段 | 交付物 | 状态 |
|---|---|---|
| **M1 骨架** | 侧边栏重排、四 Tab 壳、新模型 + AutoMigrate、空 CRUD 路由 | ✓ |
| **M2 接入中心核心** | 模板/版本、Plan/Apply、dryRun diff、安装向导、revision 审计 | ✓ |
| **M3 指标库闭环** | 解析器（collector + dashboard）+ 关联展示 | ✓ |
| **M4 日志管理** | VLogs 实例 CRUD + LogsQL 查询接口 | ✓ |
| **M5 Grafana 多主机** | GrafanaHost 注册表 + Dashboard 管理页 | ✓ |
| **M6 加固** | N9E AlertApplier 占位、权限规则收敛、审计查询扩展 | ✓ |

## 四、Review 修复完成情况（阶段 1–6）

在 M1–M6 完成后，进行了分阶段 review，识别并修复了以下 P0/P1 问题（已合入 master）：

| 阶段 | 主题 | 主要修复 |
|---|---|---|
| 阶段 1 | 凭据 / 线程安全 / 401 体验 | bcrypt cost env、idempotency 并发安全、401 toast 体验 |
| 阶段 2 | 数据模型 | 新增缺失索引、补审计表、修补漏注册的模型 |
| 阶段 3 | 安全 | JWT 黑名单（Redis）、租户隔离统一、bootstrap 审计 |
| 阶段 4 | 核心 schema | partial unique index 修正（活跃唯一）、外键 cascade、status 白名单、scale 互斥锁 |
| 阶段 5 | 接入 lifecycle | 模板/版本/安装的 P0/P1：版本最后一个不允许删、版本被引用 409、卸载后 reinstall 复用旧 ID 等 |
| 阶段 6 | 前端 | extractApiError 全站统一、Sidebar 角色过滤、token 收敛到 useAuthStore、AbortController、reinstall / uninstall_failed 文案、确认弹窗修正 |

## 五、归档说明

- 本文档不再作为开发依据；现行设计请看上方"现行文档"链接
- 后续若有大型阶段性工作，新建独立文件（如 `99-2-xxx改造计划.md`），完成后同样合入正式设计文档并归档
- 不要把临时计划长期保留为权威文档
