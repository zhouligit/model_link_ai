# ModLinkCloud OpenAPI 规范

| 文件 | 说明 |
|------|------|
| [modlink-gateway.openapi.yaml](./modlink-gateway.openapi.yaml) | **推理网关**：`/mlk/v1/*`，Bearer `mk_live_*` |
| [modlink-platform.openapi.yaml](./modlink-platform.openapi.yaml) | **平台 API**：`/mlk/platform/v1/*`、支付回调、`/mlk/health` 等 |

**与人读文档对应**：[模链云-API接口文档.md](../模链云-API接口文档.md)（**v1.4**，含 **`/mlk`** 前缀与 A–H）。

**使用**：导入 [Swagger Editor](https://editor.swagger.io/)、Stoplight、或 `openapi-generator` 生成客户端。**占位**：将 server URL 中的 `{domain}` 换为备案域名。

**校验**（需本机安装工具）：

```bash
npx @redocly/cli lint modlink-gateway.openapi.yaml
npx @redocly/cli lint modlink-platform.openapi.yaml
```
