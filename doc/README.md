# 模链云 ModLinkCloud — 文档索引 Document Index

本目录收录 **模链云（ModLinkCloud）** 对外材料草案与中英对照入口，以及 **产品需求文档（PRD）**。各类「要点 / Outline」仅供产品与法务起草正式文本使用，**不构成法律意见**；上线前须审阅并替换占位信息（主体全称、联系方式、适用法域等）。

This folder holds draft external-facing materials and a bilingual index, plus the **PRD**. All “要点 / Outline” documents are **drafting aids**, not legal advice. Replace placeholders (legal entity, contacts, governing law) before launch.

---

## 仓库与运行 Repo & run

| 文档 | 说明 |
|------|------|
| [../README.md](../README.md) | **工作区根目录**：目录结构、**`docker-compose.prod` 一键**、双后端、前端 |
| [../deploy/README.md](../deploy/README.md) | **生产 Docker Compose**：环境变量、**RDS**、HTTPS、扩副本 |

---

## 产品需求 Product requirements

| 文档 | 说明 |
|------|------|
| [模链云-产品需求文档-PRD.md](./模链云-产品需求文档-PRD.md) | 详细 PRD（**v0.5**）：**`/mlk`** 路径前缀；§17 Base URL |

---

## 技术选型 Technical decisions

| 文档 | 说明 |
|------|------|
| [模链云-前端技术选型说明.md](./模链云-前端技术选型说明.md) | **已定**：Vue 3 + Vite + TypeScript + Element Plus；admin/web 双 Web 工程；与 `travel-together`（uni-app）的边界说明 |

---

## 架构与服务 Architecture & repos

| 文档 | 说明 |
|------|------|
| [模链云-服务划分与仓库说明.md](./模链云-服务划分与仓库说明.md) | **已定**：**四仓库**；后端 **逻辑双服务**（`/mlk/v1` + `/mlk/platform/v1`）；**v1.1** |
| [模链云-技术设计方案.md](./模链云-技术设计方案.md) | **总体技术方案**（**v1.0**）：系统上下文、Gateway/Platform 边界、链路、数据与一致性、安全观测、部署演进 |

---

## API 接口 API specification

| 文档 | 说明 |
|------|------|
| [模链云-API接口文档.md](./模链云-API接口文档.md) | **全量**接口（**v1.5**）：附录 A；**`/mlk`**；**A–H** |
| [模链云-API接口文档-详述.md](./模链云-API接口文档-详述.md) | **详述版**：参数表 + JSON + **HTTP 原始请求** + 响应 data（含短信验证码示例；管理端索引） |
| [openapi/README.md](./openapi/README.md) | **OpenAPI 3.1**（**info 1.4.0**）：网关 `/mlk/v1` + 平台 `/mlk/platform/v1` |

---

## 数据持久化 Persistence（MySQL）

| 文档 | 说明 |
|------|------|
| [模链云-MySQL数据库设计.md](./模链云-MySQL数据库设计.md) | 库名 **`modlink_cloud`**；**23 张表**及完整 **`CREATE TABLE`**（MySQL 8.0+） |

---

## 文档对照表 Bilingual document map

| 中文文档 | English document | 用途摘要 Summary |
|----------|------------------|------------------|
| [模链云-产品说明书.md](./模链云-产品说明书.md) | [ModLinkCloud-Product-Brief.md](./ModLinkCloud-Product-Brief.md) | 产品定位、功能模块、仓库划分、第三方免责摘要 |
| [模链云-用户协议要点.md](./模链云-用户协议要点.md) | [ModLinkCloud-Terms-of-Service-Outline.md](./ModLinkCloud-Terms-of-Service-Outline.md) | 正式《用户协议》条款纲要；含 SaaS+后端推理表述附录 |
| [模链云-隐私政策要点.md](./模链云-隐私政策要点.md) | [ModLinkCloud-Privacy-Policy-Outline.md](./ModLinkCloud-Privacy-Policy-Outline.md) | 正式《隐私政策》纲要；含模型分包商、跨境、日志披露 |

---

## 阅读顺序建议 Suggested reading order

1. **产品说明书 / Product Brief** — 统一产品与架构语境。  
2. **隐私政策要点 / Privacy Outline** — 与数据处理、日志、模型分包商强相关，建议与用户协议同步起草。  
3. **用户协议要点 / Terms Outline** — 与隐私政策交叉引用（知识产权、输入输出处理等）。

---

## 修订记录 Revision log

| 日期 Date | 说明 Note |
|-----------|-----------|
| 2026-05-10 | 初版索引与六份文档入库 |
| 2026-05-10 | 新增 PRD（v0.1）及索引条目 |
| 2026-05-10 | PRD 升至 **v0.2**：客户确认 §17 条款 |
| 2026-05-10 | PRD 升至 **v0.3**：采纳 Q4/Q8 细化与 **Q6 独立 API 子域推荐** |
| 2026-05-10 | PRD 升至 **v0.4**：客户确认 **Q6 采用独立 API 子域** |
| 2026-05-10 | 新增 [前端技术选型说明](./模链云-前端技术选型说明.md) |
| 2026-05-10 | 新增 [API 接口文档](./模链云-API接口文档.md)（全量） |
| 2026-05-10 | API 文档 **v1.2**：A–H **已定** |
| 2026-05-10 | API 文档 **v1.3** + `doc/openapi/` **双 YAML** |
| 2026-05-10 | **路径前缀 `/mlk`**：API **v1.4**、OpenAPI **1.4.0**、PRD **v0.5**、服务划分 **v1.1** |
| 2026-05-10 | API 文档 **v1.5**：**附录 A** JSON 示例 |
| 2026-05-10 | 新增 [服务划分与仓库说明](./模链云-服务划分与仓库说明.md) |
| 2026-05-10 | 新增 [MySQL 数据库设计](./模链云-MySQL数据库设计.md) |
| 2026-05-10 | 新增 [API 接口文档-详述](./模链云-API接口文档-详述.md)（表格+HTTP 版式） |
| 2026-05-10 | 新增 [技术设计方案](./模链云-技术设计方案.md)（总体架构与关键链路） |
| 2026-05-10 | 根目录 [README](../README.md)：modlink-gateway / modlink-cloud-web / modlink-cloud-admin 与 compose |
| 2026-05-10 | [deploy/README.md](../deploy/README.md)：`docker-compose.prod.yml` 一键、Nginx、`Dockerfile` |
| 2026-05-10 | 共机端口 **8100 起**、`host-mlk-proxy.conf`；技术方案 **v1.1** |

---

## 占位项清单 Placeholders to resolve before release

- 运营主体法定名称与注册地址  
- 隐私与支持联系方式（含隐私专用邮箱）  
- 适用法域、争议解决方式、未成年人政策  
- 支付与预付费/额度细则（与订单页一致）  
- 第三方模型与分包商清单（类别或名称）  

---

**产品名称**：模链云 · **ModLinkCloud**
