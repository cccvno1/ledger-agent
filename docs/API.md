# Ledger Agent API 文档

**Base URL**: `http://localhost:8080`

**Content-Type**: `application/json`

## 认证

除 `/health` 和 `/api/v1/wechat/qrcode*` 外，所有请求需在 Header 中携带：

```
Authorization: Bearer <AUTH_TOKEN>
```

`AUTH_TOKEN` 在服务端通过环境变量设置，前端与服务端共享同一个值。

## 错误格式

```json
{ "code": "invalid_input", "message": "customer_id is required" }
```

| HTTP 状态码 | 含义 |
|------------|------|
| `400` | 参数错误（`invalid_input`） |
| `401` | 未认证或 Token 错误 |
| `404` | 资源不存在 |
| `500` | 服务器内部错误 |

---

## 一、对话（AI 记账）

### `POST /api/v1/chat`

自然语言记账，AI 解析消息并返回回复和待确认草稿。

**请求体**

```json
{
  "session_id": "user-session-1",
  "message": "张三买了10斤苹果，单价5块"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `session_id` | string | ❌ | 会话 ID，同一用户保持一致即可；为空时每次独立会话 |
| `message` | string | ✅ | 自然语言消息，不能为空 |

**响应 200**

```json
{
  "session_id": "user-session-1",
  "reply": "已理解，张三今天买了10斤苹果，单价5元，共50元，确认保存吗？",
  "draft": [
    {
      "customer_name": "张三",
      "product_name": "苹果",
      "quantity": 10,
      "unit": "斤",
      "unit_price": 5,
      "amount": 50,
      "entry_date": "2026-04-21"
    }
  ]
}
```

> `draft` 非空时代表 AI 等待确认，发送"确认"/"好的"/"是"等即可保存。`draft` 为空时表示操作已完成或只是查询。

---

## 二、仪表板

### `GET /api/v1/dashboard`

首页概览数据，所有客户的汇总统计。

**响应 200**

```json
{
  "total_pending": 1580.0,
  "total_customers": 12,
  "entries_this_month": 45,
  "amount_this_month": 3200.0
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `total_pending` | float64 | 所有客户欠款总额（总账目 - 总收款，最小为 0） |
| `total_customers` | int | 客户总数 |
| `entries_this_month` | int | 本月账目笔数 |
| `amount_this_month` | float64 | 本月出货总金额 |

---

## 三、客户

### `GET /api/v1/customers`

获取所有客户列表。

**响应 200**

```json
[
  {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "张三",
    "aliases": ["张老三", "老张"],
    "created_at": "2026-04-01T10:00:00Z"
  }
]
```

### `GET /api/v1/customers/{id}`

获取单个客户详情。

- **响应 200** — 同上单条结构
- **响应 404** — 客户不存在

---

## 四、账目（出货记录）

### `GET /api/v1/entries`

查询账目列表，支持多维度过滤。

**Query 参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `customer_id` | string | ❌ | 按客户 ID 过滤 |
| `date_from` | string | ❌ | 开始日期，格式 `YYYY-MM-DD` |
| `date_to` | string | ❌ | 结束日期，格式 `YYYY-MM-DD` |
| `is_settled` | bool | ❌ | `true` 只看已结算，`false` 只看未结算 |

**响应 200**

```json
[
  {
    "id": "550e8400-e29b-41d4-a716-446655440001",
    "customer_id": "550e8400-e29b-41d4-a716-446655440000",
    "customer_name": "张三",
    "product_name": "苹果",
    "unit_price": 5.0,
    "quantity": 10.0,
    "unit": "斤",
    "amount": 50.0,
    "entry_date": "2026-04-21",
    "is_settled": false,
    "settled_at": null,
    "notes": "",
    "created_at": "2026-04-21T10:00:00Z",
    "updated_at": "2026-04-21T10:00:00Z"
  }
]
```

### `POST /api/v1/entries`

手动录入一笔账目（直接入库，不经过 AI）。

**请求体**

```json
{
  "customer_id": "550e8400-e29b-41d4-a716-446655440000",
  "customer_name": "张三",
  "product_name": "苹果",
  "unit_price": 5.0,
  "quantity": 10.0,
  "unit": "斤",
  "entry_date": "2026-04-21",
  "notes": ""
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `customer_id` | string | ✅ | |
| `customer_name` | string | ✅ | |
| `product_name` | string | ✅ | |
| `unit_price` | float64 | ✅ | 必须 > 0 |
| `quantity` | float64 | ✅ | 必须 > 0 |
| `unit` | string | ❌ | 单位（斤、箱等），默认空 |
| `entry_date` | string | ✅ | 格式 `YYYY-MM-DD` |
| `notes` | string | ❌ | 备注 |

**响应 201** — 创建成功的 EntryResponse 结构

### `PUT /api/v1/entries/{id}`

更新账目，所有字段可选，只传需要修改的字段。

**请求体**（全部可选）

```json
{
  "product_name": "红富士苹果",
  "unit_price": 6.0,
  "quantity": 12.0,
  "unit": "斤",
  "entry_date": "2026-04-22",
  "notes": "已确认数量"
}
```

**响应 200** — 更新后的 EntryResponse

### `DELETE /api/v1/entries/{id}`

删除账目。

**响应 204**（无 body）

---

## 五、客户汇总与结算

### `GET /api/v1/customers/{customer_id}/summary`

获取该客户账目的汇总统计。

**响应 200**

```json
[
  {
    "customer_id": "550e8400-e29b-41d4-a716-446655440000",
    "customer_name": "张三",
    "total_amount": 300.0,
    "settled_amount": 100.0,
    "pending_amount": 200.0,
    "entry_count": 5
  }
]
```

| 字段 | 说明 |
|------|------|
| `total_amount` | 该客户所有出货总金额 |
| `settled_amount` | 已结算金额 |
| `pending_amount` | 待结算金额 |
| `entry_count` | 账目笔数 |

### `POST /api/v1/customers/{customer_id}/settle`

将该客户所有未结算账目一键标记为已结算。

**响应 200**

```json
{ "status": "settled" }
```

---

## 六、收款记录

### `GET /api/v1/payments`

查询收款记录。

**Query 参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `customer_id` | string | ❌ | 按客户 ID 过滤 |

**响应 200**

```json
[
  {
    "id": "550e8400-e29b-41d4-a716-446655440002",
    "customer_id": "550e8400-e29b-41d4-a716-446655440000",
    "amount": 200.0,
    "payment_date": "2026-04-20",
    "notes": "微信转账",
    "created_at": "2026-04-20T15:00:00Z"
  }
]
```

### `POST /api/v1/payments`

录入一笔收款记录。

**请求体**

```json
{
  "customer_id": "550e8400-e29b-41d4-a716-446655440000",
  "amount": 200.0,
  "payment_date": "2026-04-20",
  "notes": "微信转账"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `customer_id` | string | ✅ | |
| `amount` | float64 | ✅ | 必须 > 0 |
| `payment_date` | string | ✅ | 格式 `YYYY-MM-DD` |
| `notes` | string | ❌ | 备注 |

**响应 201** — 创建成功的 PaymentResponse 结构

---

## 七、商品

### `GET /api/v1/products`

获取所有商品列表，用于前端输入补全。

**响应 200**

```json
[
  {
    "id": "550e8400-e29b-41d4-a716-446655440003",
    "name": "苹果",
    "aliases": ["红富士", "苹"],
    "default_unit": "斤",
    "reference_price": 5.0,
    "created_at": "2026-04-01T10:00:00Z"
  }
]
```

> `reference_price` 为 0 时字段省略（不出现在响应中）。

---

## 八、微信 QR 绑定（无需认证）

用于前端引导商家扫码绑定微信，实现微信消息自动转发到 AI。

### `POST /api/v1/wechat/qrcode`

获取微信登录二维码。

**响应 200**

```json
{
  "qrcode": "<token字符串>",
  "img_content": "<二维码内容字符串，用qrcode.js渲染>"
}
```

> 前端使用 `img_content` 字段配合 [qrcode.js](https://github.com/davidshimjs/qrcodejs) 渲染二维码图片。

### `GET /api/v1/wechat/qrcode/status?qrcode={token}`

轮询扫码状态，建议每 **3 秒**调用一次。

**响应 200**

```json
{ "status": "wait" }
```

| `status` 值 | 说明 |
|-------------|------|
| `wait` | 等待扫码 |
| `scaned` | 已扫码，等待手机端确认 |
| `confirmed` | 确认成功，微信绑定完成 |
| `expired` | 二维码已过期，需重新调用 `POST /api/v1/wechat/qrcode` 获取新的 |

> 收到 `confirmed` 后，提示用户"微信绑定成功！重启服务后即可通过微信发送消息记账"。

---

## 九、健康检查（无需认证）

### `GET /health`

**响应 200**

```json
{ "status": "ok" }
```

---

## 前端交互流程参考

### 首页加载

```
GET /api/v1/dashboard     → 显示总欠款、本月统计
GET /api/v1/customers     → 渲染客户列表
```

### 查看客户详情

```
GET /api/v1/customers/{id}
GET /api/v1/customers/{id}/summary   → 显示汇总金额
GET /api/v1/entries?customer_id={id}&is_settled=false  → 未结算账目
GET /api/v1/payments?customer_id={id}                  → 收款记录
```

### AI 对话记账

```
POST /api/v1/chat  message="张三买了10斤苹果单价5块"
  → draft 非空时显示确认弹窗
POST /api/v1/chat  message="确认"
  → draft 为空，reply 含成功提示
```

### 一键结算

```
POST /api/v1/customers/{id}/settle
  → 刷新 summary 和 entries 列表
```

### 微信绑定流程

```
POST /api/v1/wechat/qrcode
  → 用 img_content 渲染二维码
每3秒轮询 GET /api/v1/wechat/qrcode/status?qrcode={token}
  → status=confirmed 时提示绑定成功
```
