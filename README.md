# MACDP — Multi-Agent Collaborative Development Platform

一个用 Go 构建的多 AI Agent 协同开发平台，能统一调度 Hermes、Codex、Claude Code 等编码代理，像一支 AI 开发团队一样协作完成软件项目。

## 解决什么问题？

现在 AI 编码代理越来越多（Hermes、Codex、Claude Code、OpenCode……），但它们各自为战：

- **一次只能用一个** — 你想让 Claude 写后端、Codex 写前端，得手动来回切换
- **无法并行** — 一个 Agent 在干活时，其他的只能等着
- **没有审查闭环** — 写完代码没人 review，质量无法保证
- **没有任务管理** — 大项目怎么拆分、谁做什么、进度如何，全靠人盯

MACDP 就是要解决这些问题：**一个入口，调度多个 Agent，并行开发，自动审查，结果汇总。**

## 核心架构

```
用户 (自然语言描述需求)
  │
  ▼
Orchestrator (编排器)
  ├── Task Planner: 把大任务拆成子任务 DAG
  ├── Scheduler: 根据依赖图并行调度
  ├── Git Manager: 每个任务一个 worktree，互不干扰
  └── Review Pipeline: Agent 交叉审查 + 自动测试
  │
  ▼
Worker Pool (Agent 工人池)
  ├── Hermes   — 调试、测试、通用任务
  ├── Codex    — 快速原型、脚本编写
  ├── Claude   — 复杂编码、代码审查
  └── OpenCode — 可扩展的通用 Worker
```

## 技术选型

| 层 | 技术 | 理由 |
|----|------|------|
| 语言 | Go | goroutine 真并发，单二进制部署 |
| DAG 引擎 | cloudwego/eino | 字节跳动出品，生产级 DAG 编排 |
| Agent 通信 | 消息总线模式 | 借鉴 MetaGPT |
| 任务委派 | channel-based | 借鉴 CrewAI，非阻塞 |
| 持久化 | SQLite | 任务状态 + Checkpoint |
| 前端 | React + WebSocket | 实时日志推送 |

## 对比现有方案

| | MetaGPT | CrewAI | **MACDP** |
|---|---|---|---|
| 语言 | Python | Python | **Go** |
| 并发 | asyncio 假并发 | asyncio 假并发 | **goroutine 真并发** |
| 外部 CLI 集成 | ✗ | ✗ | **✓** |
| 部署 | 需 Python 环境 | 需 Python 环境 | **单二进制** |
| Git Worktree | ✗ | ✗ | **✓ 自动隔离** |
| 交叉审查 | 有限 | ✗ | **✓ 自动化** |

## 项目结构

```
macdp/
├── cmd/macdp/main.go              # CLI 入口
├── internal/
│   ├── agent/
│   │   ├── adapter.go             # AgentAdapter 接口
│   │   ├── hermes.go              # Hermes CLI 适配器
│   │   ├── claude_code.go         # Claude Code 适配器
│   │   └── codex.go               # Codex 适配器
│   ├── task/task.go               # Task + DAG 数据结构
│   ├── orchestrator/scheduler.go  # 并发调度器
│   ├── git/worktree.go            # Git Worktree 管理
│   ├── api/server.go              # HTTP + SSE API
│   └── config/config.go           # YAML 配置
├── configs/
│   ├── macdp.yaml                 # 默认配置
│   └── example_tasks.json         # 示例任务
├── DESIGN.md                      # 详细设计文档
└── ARCHITECTURE_COMPARISON.md     # 架构对比分析
```

## 快速开始

```bash
# 编译
go build -o macdp ./cmd/macdp/

# 用示例任务运行
./macdp --tasks configs/example_tasks.json

# 启动 API 服务
./macdp --serve
```

## License

MIT
