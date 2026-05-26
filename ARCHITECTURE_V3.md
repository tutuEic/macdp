# MACDP v3 架构设计

> Multi-Agent Collaborative Development Platform
> 中央控制台模式：连接外部 Agent 服务，管理项目，协同开发

---

## 一、核心定位

**MACDP 是一个 Agent 中央控制台**，不是 LLM 客户端。

它连接外部 Agent 服务（Hermes、OpenClaw、Codex、Claude Code），在 UI 上统一管理项目、分配任务、监控进度、Agent 间对话、代码协作。

```
┌─────────────────────────────────────────────────────┐
│                MACDP 中央控制台 (Go)                  │
│                                                      │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐            │
│  │ 项目管理  │ │ Agent    │ │ 任务看板  │            │
│  │          │ │ 对话面板  │ │ (Kanban) │            │
│  └──────────┘ └──────────┘ └──────────┘            │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐            │
│  │ 代码协作  │ │ 进度监控  │ │ 审查合并  │            │
│  │ 中心     │ │ 仪表盘   │ │ 流水线   │            │
│  └──────────┘ └──────────┘ └──────────┘            │
└────────────────────┬────────────────────────────────┘
                     │
        ┌────────────┼────────────┬──────────────┐
        ▼            ▼            ▼              ▼
   ┌─────────┐ ┌──────────┐ ┌─────────┐ ┌──────────┐
   │  Hermes  │ │ OpenClaw │ │  Codex  │ │ Claude   │
   │  Agent   │ │ Gateway  │ │  CLI    │ │ Code CLI │
   │ (外部)   │ │ (外部)   │ │ (外部)  │ │ (外部)   │
   └─────────┘ └──────────┘ └─────────┘ └──────────┘
```

---

## 二、系统架构

```
┌─────────────────────────────────────────────────────────────┐
│                    前端 (React + TypeScript)                 │
│                                                             │
│  ┌───────────┐ ┌───────────┐ ┌───────────┐ ┌───────────┐  │
│  │ Dashboard  │ │ Agent Chat│ │  Project   │ │  Kanban   │  │
│  │ 仪表盘     │ │ 对话面板  │ │  项目管理  │ │  任务看板  │  │
│  │           │ │           │ │           │ │           │  │
│  │ • 项目概览 │ │ • 实时对话 │ │ • 模块划分 │ │ • 任务卡片 │  │
│  │ • Agent   │ │ • 消息历史 │ │ • Agent   │ │ • 拖拽分配 │  │
│  │   状态    │ │ • 代码片段 │ │   分配    │ │ • 进度条  │  │
│  │ • 成本    │ │ • 文件共享 │ │ • Git 分支│ │ • 依赖线  │  │
│  │   统计    │ │ • 指令发送 │ │ • 合并策略│ │ • 筛选过滤 │  │
│  └─────┬─────┘ └─────┬─────┘ └─────┬─────┘ └─────┬─────┘  │
│        │             │             │             │          │
│        └─────────────┴──────┬──────┴─────────────┘          │
│                             │ WebSocket + REST               │
└─────────────────────────────┼───────────────────────────────┘
                              │
┌─────────────────────────────▼───────────────────────────────┐
│                    后端 (Go + Gin)                           │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                 API Gateway                          │   │
│  │  REST: /api/projects, /api/agents, /api/tasks       │   │
│  │  WS:   /ws/events (实时推送), /ws/chat (Agent 对话) │   │
│  └─────────────────────┬───────────────────────────────┘   │
│                         │                                    │
│  ┌──────────────────────▼──────────────────────────────┐   │
│  │              Service Layer (业务逻辑)                 │   │
│  │                                                      │   │
│  │  ┌────────────┐ ┌────────────┐ ┌────────────┐      │   │
│  │  │ Project    │ │ Agent      │ │ Task       │      │   │
│  │  │ Service    │ │ Service    │ │ Service    │      │   │
│  │  │ • 创建项目 │ │ • 连接管理 │ │ • 创建任务 │      │   │
│  │  │ • 模块划分 │ │ • 状态监控 │ │ • 分配 Agent│     │   │
│  │  │ • 分支管理 │ │ • 消息路由 │ │ • 进度追踪 │      │   │
│  │  │ • 合并策略 │ │ • 指令下发 │ │ • 依赖管理 │      │   │
│  │  └────────────┘ └────────────┘ └────────────┘      │   │
│  │                                                      │   │
│  │  ┌────────────┐ ┌────────────┐ ┌────────────┐      │   │
│  │  │ Chat       │ │ Code       │ │ Review     │      │   │
│  │  │ Service    │ │ Collab     │ │ Service    │      │   │
│  │  │ • 会话管理 │ │ • 文件共享 │ │ • 代码审查 │      │   │
│  │  │ • 消息存储 │ │ • 代码同步 │ │ • 交叉审查 │      │   │
│  │  │ • 上下文   │ │ • 冲突检测 │ │ • 合并检查 │      │   │
│  │  │   传递     │ │ • 通知机制 │ │ • 报告生成 │      │   │
│  │  └────────────┘ └────────────┘ └────────────┘      │   │
│  └──────────────────────────────────────────────────────┘   │
│                         │                                    │
│  ┌──────────────────────▼──────────────────────────────┐   │
│  │           Agent Gateway (Agent 网关)                 │   │
│  │                                                      │   │
│  │  ┌──────────────┐  ┌────────────────────────────┐  │   │
│  │  │ Adapter      │  │ Message Router              │  │   │
│  │  │ Registry     │  │ • Agent → Agent 消息路由    │  │   │
│  │  │              │  │ • Agent → UI 事件推送       │  │   │
│  │  │ • Hermes     │  │ • 上下文注入               │  │   │
│  │  │ • OpenClaw   │  │ • 文件变更通知              │  │   │
│  │  │ • Codex      │  └────────────────────────────┘  │   │
│  │  │ • Claude Code│                                   │   │
│  │  │ • 自定义...  │  ┌────────────────────────────┐  │   │
│  │  └──────────────┘  │ Connection Pool              │  │   │
│  │                    │ • HTTP/WS 连接池             │  │   │
│  │                    │ • 健康检查                   │  │   │
│  │                    │ • 自动重连                   │  │   │
│  │                    └────────────────────────────┘  │   │
│  └──────────────────────────────────────────────────────┘   │
│                         │                                    │
│  ┌──────────────────────▼──────────────────────────────┐   │
│  │              Data Layer (数据层)                      │   │
│  │  SQLite: 项目、任务、Agent、消息、审查记录           │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                              │
         ┌────────────────────┼────────────────────┐
         ▼                    ▼                    ▼
   ┌──────────┐        ┌──────────┐        ┌──────────┐
   │  Hermes  │        │ OpenClaw │        │  Codex/  │
   │  Agent   │        │ Gateway  │        │  Claude  │
   │ :5000    │        │ :18789   │        │ :3000    │
   └──────────┘        └──────────┘        └──────────┘
```

---

## 三、核心功能模块

### 3.1 项目管理

```
项目 (Project)
├── 基本信息: 名称、描述、Git 仓库路径
├── 模块划分: 前端、后端、数据库、测试...
├── Agent 分配: 每个模块分配一个或多个 Agent
├── Git 管理: 分支策略、合并规则
└── 成本追踪: 各 Agent 的 token 消耗
```

**一个项目的典型配置：**

```yaml
project:
  name: "todo-app"
  repo: "/mnt/d/projects/todo-app"
  modules:
    - name: "backend"
      path: "src/backend/"
      agents: ["hermes", "claude-code"]
      branch: "feature/backend"
    - name: "frontend"
      path: "src/frontend/"
      agents: ["codex"]
      branch: "feature/frontend"
    - name: "database"
      path: "src/db/"
      agents: ["hermes"]
      branch: "feature/database"
```

### 3.2 Agent 管理

```
Agent 连接
├── 连接方式
│   ├── Hermes:  HTTP API (hermes gateway) 或 CLI (hermes chat -q)
│   ├── OpenClaw: HTTP API (localhost:18789)
│   ├── Codex:   CLI (codex exec)
│   └── Claude:  CLI (claude -p) 或 HTTP API
├── 状态监控
│   ├── 在线/离线
│   ├── 空闲/忙碌
│   ├── 当前任务
│   └── 资源消耗
└── 能力声明
    ├── 支持的工具 (shell, file_io, web...)
    ├── 擅长领域 (backend, frontend, review...)
    └── 并发限制
```

**Agent 连接器接口：**

```go
// AgentConnector 是连接外部 Agent 服务的接口
type AgentConnector interface {
    // 基本信息
    Name() string
    Type() AgentType  // hermes, openclaw, codex, claude-code
    
    // 连接管理
    Connect(ctx context.Context, config AgentConfig) error
    Disconnect() error
    Ping() error
    Status() AgentStatus  // online, offline, busy, idle
    
    // 任务执行
    ExecuteTask(ctx context.Context, task *Task) (<-chan *TaskEvent, error)
    CancelTask(taskID string) error
    
    // 对话交互
    SendMessage(ctx context.Context, msg *ChatMessage) (<-chan *ChatMessage, error)
    
    // 文件操作
    ShareFile(ctx context.Context, file *SharedFile) error
    GetChanges(ctx context.Context) ([]*FileChange, error)
}

// AgentType 枚举
type AgentType string
const (
    AgentHermes    AgentType = "hermes"
    AgentOpenClaw  AgentType = "openclaw"
    AgentCodex     AgentType = "codex"
    AgentClaudeCode AgentType = "claude-code"
)

// AgentStatus 状态
type AgentStatus struct {
    Online     bool      `json:"online"`
    State      string    `json:"state"`      // idle, busy, error
    CurrentTask string   `json:"current_task"`
    Uptime     string    `json:"uptime"`
    LastPing   time.Time `json:"last_ping"`
}
```

### 3.3 任务管理 (Kanban)

```
任务看板
├── 列: 待办 | 进行中 | 审查中 | 已完成 | 已阻塞
├── 卡片: 任务名、分配 Agent、进度条、依赖关系
├── 操作:
│   ├── 拖拽分配任务给 Agent
│   ├── 设置依赖关系
│   ├── 拆分子任务
│   ├── 添加审查者
│   └── 标记阻塞/完成
└── 视图:
    ├── 看板视图 (拖拽)
    ├── 甘特图 (时间线)
    └── DAG 视图 (依赖图)
```

**任务数据模型：**

```go
type Task struct {
    ID          string     `json:"id"`
    ProjectID   string     `json:"project_id"`
    Title       string     `json:"title"`
    Description string     `json:"description"`
    Module      string     `json:"module"`       // 所属模块
    Status      TaskStatus `json:"status"`
    Priority    int        `json:"priority"`
    
    // Agent 分配
    AssignedAgent string   `json:"assigned_agent"` // 执行者
    Reviewer      string   `json:"reviewer"`       // 审查者
    
    // 依赖
    DependsOn   []string   `json:"depends_on"`
    Blocks      []string   `json:"blocks"`
    
    // Git
    Branch      string     `json:"branch"`
    Worktree    string     `json:"worktree"`
    
    // 进度
    Progress    int        `json:"progress"`  // 0-100
    StartedAt   *time.Time `json:"started_at"`
    CompletedAt *time.Time `json:"completed_at"`
    
    // 产出
    FilesChanged []string  `json:"files_changed"`
    CostUSD      float64   `json:"cost_usd"`
}
```

### 3.4 Agent 对话面板

```
对话面板 (类似 IDE 终端)
├── 左侧: Agent 列表 (在线状态)
├── 中间: 对话区域
│   ├── 文本消息
│   ├── 代码片段 (语法高亮)
│   ├── 文件变更 (diff 视图)
│   ├── 工具调用 (折叠显示)
│   └── 指令输入框
├── 右侧: 上下文面板
│   ├── 当前任务信息
│   ├── 相关文件列表
│   └── 共享上下文
└── 操作:
    ├── 发送消息/指令给 Agent
    ├── 转发消息给另一个 Agent
    ├── 共享文件/代码片段
    ├── 查看 Agent 执行历史
    └── 暂停/恢复/取消任务
```

### 3.5 代码协作中心

```
代码协作
├── 文件变更监控
│   ├── Agent A 修改了 auth.py → 通知 Agent B
│   ├── Agent B 需要 auth 接口 → 自动获取最新代码
│   └── 冲突检测: 两个 Agent 改了同一文件 → 告警
├── 代码共享
│   ├── Agent A: "这是 API schema，前端按这个对接"
│   ├── → 自动将文件/代码片段发送给 Agent B
│   └── → Agent B 的上下文中注入这些信息
├── Git 同步
│   ├── 定期 pull 各 worktree 的最新代码
│   ├── 检测跨模块依赖变更
│   └── 自动 rebase/merge 提示
└── 合并流水线
    ├── 所有任务完成 → 自动运行测试
    ├── 测试通过 → 创建合并 PR
    └── 人工确认 → merge to main
```

---

## 四、Agent 间协同机制

### 4.1 消息路由

```go
// MessageRouter 负责 Agent 间的消息路由
type MessageRouter struct {
    agents   map[string]AgentConnector
    bus      *EventBus
    store    *Store
}

// 路由规则
// 1. Agent A 的输出 → 存入 Store → 通知相关 Agent
// 2. 文件变更事件 → 通知订阅该文件的 Agent
// 3. 任务状态变更 → 通知相关 Agent + UI
// 4. 用户指令 → 路由到目标 Agent
```

### 4.2 上下文传递

```go
// ContextBridge 负责在 Agent 间传递上下文
type ContextBridge struct {
    store *Store
}

// 当 Agent B 需要 Agent A 的产出时：
func (cb *ContextBridge) BuildContext(task *Task) string {
    var ctx strings.Builder
    
    // 1. 注入依赖任务的产出
    for _, depID := range task.DependsOn {
        dep := cb.store.GetTask(depID)
        ctx.WriteString(fmt.Sprintf("## %s 的产出\n%s\n\n", dep.Title, dep.Result))
    }
    
    // 2. 注入共享文件
    for _, file := range cb.store.GetSharedFiles(task.ProjectID, task.Module) {
        ctx.WriteString(fmt.Sprintf("## 共享文件: %s\n```\n%s\n```\n\n", file.Path, file.Content))
    }
    
    // 3. 注入项目约定
    ctx.WriteString(fmt.Sprintf("## 项目约定\n%s\n", cb.store.GetProjectConventions(task.ProjectID)))
    
    return ctx.String()
}
```

### 4.3 典型协同流程

```
场景: Agent A (后端) 完成 API，Agent B (前端) 需要对接

1. Agent A 完成 task "实现用户注册 API"
   → MACDP 检测到文件变更: src/api/auth.py
   → 自动提取 API schema (路由、参数、返回值)
   → 存入共享上下文

2. Agent B 开始 task "实现登录页面"
   → MACDP 自动注入 Agent A 的 API schema 到 Agent B 的上下文
   → Agent B 说: "我需要 auth API 的接口文档"
   → MACDP 自动将 API schema 发送给 Agent B

3. Agent B 完成后
   → MACDP 检测到前端调用了 /api/auth/register
   → 自动验证接口是否匹配
   → 不匹配 → 告警并通知两个 Agent
```

---

## 五、前端 UI 页面

### 5.1 Dashboard (仪表盘)

```
┌─────────────────────────────────────────────────────────────┐
│  MACDP Dashboard                    [项目: todo-app] [设置] │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐         │
│  │  Agent   │ │  任务    │ │  完成    │ │  成本    │         │
│  │  在线    │ │  进行中  │ │  今日    │ │  今日    │         │
│  │  3/4     │ │  5      │ │  12     │ │  $2.35  │         │
│  └─────────┘ └─────────┘ └─────────┘ └─────────┘         │
│                                                             │
│  ┌─── Agent 状态 ─────────────────────────────────────┐   │
│  │ 🟢 Hermes    空闲          最后活跃: 2分钟前       │   │
│  │ 🟢 Claude    执行中 T-007  进度: ████████░░ 80%    │   │
│  │ 🟢 Codex     空闲          最后活跃: 5分钟前       │   │
│  │ 🔴 OpenClaw  离线          已断开 10分钟           │   │
│  └────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌─── 最近活动 ───────────────────────────────────────┐   │
│  │ 14:32  [Claude] 完成 T-007: 实现登录API            │   │
│  │ 14:28  [Hermes] 审查通过 T-005: 数据库schema       │   │
│  │ 14:25  [Codex]  开始 T-009: 前端路由配置           │   │
│  │ 14:20  [系统]   Agent A 共享 API schema 给 Agent B │   │
│  └────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

### 5.2 Project Manager (项目管理)

```
┌─────────────────────────────────────────────────────────────┐
│  项目管理: todo-app                          [+ 新建模块]   │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─── 模块 ────────┐ ┌─── Agent 分配 ───┐ ┌─── 分支 ───┐ │
│  │                 │ │                  │ │            │ │
│  │ 📦 backend      │ │ 🤖 Hermes        │ │ feat/back  │ │
│  │    src/api/     │ │ 🤖 Claude Code   │ │            │ │
│  │                 │ │                  │ │            │ │
│  │ 📦 frontend     │ │ 🤖 Codex         │ │ feat/front │ │
│  │    src/web/     │ │                  │ │            │ │
│  │                 │ │                  │ │            │ │
│  │ 📦 database     │ │ 🤖 Hermes        │ │ feat/db    │ │
│  │    src/db/      │ │                  │ │            │ │
│  │                 │ │                  │ │            │ │
│  │ 📦 testing      │ │ 🤖 Hermes        │ │ feat/test  │ │
│  │    tests/       │ │                  │ │            │ │
│  └─────────────────┘ └──────────────────┘ └────────────┘ │
│                                                             │
│  ┌─── 项目设置 ───────────────────────────────────────┐   │
│  │ Git 仓库: /mnt/d/projects/todo-app                 │   │
│  │ 合并策略: squash    分支前缀: feature/             │   │
│  │ 自动审查: ✓         自动合并: ✗                    │   │
│  └────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

### 5.3 Agent Chat (Agent 对话)

```
┌─────────────────────────────────────────────────────────────┐
│  Agent 对话                                    [Claude Code]│
├────────┬────────────────────────────────────────────────────┤
│        │                                                    │
│ Agents │  ┌──────────────────────────────────────────────┐ │
│        │  │ [14:30] 你: 实现用户登录API，支持JWT          │ │
│ 🟢Hermes│  │                                              │ │
│        │  │ [14:31] Claude: 我来实现登录API...            │ │
│ 🟢Claude│ │                                              │ │
│ ● Codex │  │ [14:31] Claude: 📄 创建 src/auth/login.py   │ │
│        │  │ ```python                                    │ │
│ 🔴Open  │  │ @router.post("/auth/login")                 │ │
│  Claw  │  │ async def login(req: LoginRequest):          │ │
│        │  │     user = await authenticate(req)           │ │
│ 上下文  │  │     token = create_jwt(user.id)             │ │
│        │  │     return {"access_token": token}           │ │
│ T-007  │  │ ```                                          │ │
│ 登录API │  │                                              │ │
│        │  │ [14:32] Claude: ✅ 完成，已自动提交到分支     │ │
│ 相关文件│  │                                              │ │
│ auth.py │  │ [14:32] Claude: 需要我继续实现刷新token吗？  │ │
│ jwt.py  │  └──────────────────────────────────────────────┘ │
│        │                                                    │
│        │  ┌──────────────────────────────────────────────┐ │
│ 共享上下│  │ [输入框]                          [发送] [📎]│ │
│ 文     │  └──────────────────────────────────────────────┘ │
│        │                                                    │
│ API    │  ┌─── 快捷操作 ───────────────────────────────┐  │
│ schema │  │ [暂停任务] [取消任务] [转发给Hermes] [审查]  │  │
│ from   │  └────────────────────────────────────────────┘  │
│ T-005  │                                                    │
└────────┴────────────────────────────────────────────────────┘
```

### 5.4 Kanban (任务看板)

```
┌─────────────────────────────────────────────────────────────┐
│  任务看板: todo-app                     [+ 新建任务] [筛选] │
├─────────────┬─────────────┬─────────────┬──────────────────┤
│   待办 (3)  │  进行中 (2) │  审查中 (1) │   已完成 (6)     │
├─────────────┼─────────────┼─────────────┼──────────────────┤
│             │             │             │                  │
│ ┌─────────┐ │ ┌─────────┐ │ ┌─────────┐ │ ┌─────────┐    │
│ │T-010    │ │ │T-007    │ │ │T-005    │ │ │T-001    │    │
│ │前端路由  │ │ │登录API  │ │ │数据库   │ │ │项目初始化│    │
│ │         │ │ │         │ │ │schema   │ │ │         │    │
│ │👤 Codex │ │ │👤Claude │ │ │👤Hermes │ │ │👤Hermes │    │
│ │         │ │ │         │ │ │         │ │ │         │    │
│ │⏳ --    │ │ │⏳ 80%  │ │ │⏳ 审查中│ │ │✅ 完成  │    │
│ └─────────┘ │ └─────────┘ │ └─────────┘ │ └─────────┘    │
│             │             │             │                  │
│ ┌─────────┐ │ ┌─────────┐ │             │ ┌─────────┐    │
│ │T-011    │ │ │T-008    │ │             │ │T-002    │    │
│ │注册API  │ │ │JWT中间件 │ │             │ │FastAPI  │    │
│ │         │ │ │         │ │             │ │配置     │    │
│ │👤Claude │ │ │👤Claude │ │             │ │         │    │
│ │         │ │ │         │ │             │ │👤Hermes │    │
│ │⏳ --    │ │ │⏳ 45%  │ │             │ │✅ 完成  │    │
│ └─────────┘ │ └─────────┘ │             │ └─────────┘    │
│             │             │             │                  │
│ ┌─────────┐ │             │             │ ...             │
│ │T-012    │ │             │             │                  │
│ │集成测试  │ │             │             │                  │
│ │         │ │             │             │                  │
│ │👤Hermes │ │             │             │                  │
│ │🔒 阻塞  │ │             │             │                  │
│ │(等T-007)│ │             │             │                  │
│ └─────────┘ │             │             │                  │
└─────────────┴─────────────┴─────────────┴──────────────────┘
```

---

## 六、后端 API 设计

### REST API

```
# 项目
POST   /api/projects                    创建项目
GET    /api/projects                    列表
GET    /api/projects/:id                详情
PUT    /api/projects/:id                更新
DELETE /api/projects/:id                删除

# Agent
GET    /api/agents                      列表
GET    /api/agents/:name                详情/状态
POST   /api/agents/:name/connect        连接
POST   /api/agents/:name/disconnect     断开
POST   /api/agents/:name/message        发送消息

# 任务
POST   /api/projects/:id/tasks          创建任务
GET    /api/projects/:id/tasks          列表
GET    /api/tasks/:id                   详情
PUT    /api/tasks/:id                   更新
POST   /api/tasks/:id/assign/:agent     分配 Agent
POST   /api/tasks/:id/start             开始执行
POST   /api/tasks/:id/cancel            取消
POST   /api/tasks/:id/review            提交审查

# 对话
GET    /api/projects/:id/chat           获取聊天历史
POST   /api/projects/:id/chat           发送消息
POST   /api/chat/forward                转发消息给其他 Agent

# 文件
GET    /api/projects/:id/files/changes  获取文件变更
POST   /api/projects/:id/files/share    共享文件给 Agent
GET    /api/projects/:id/files/diff     获取 diff

# 合并
POST   /api/projects/:id/merge          触发合并
GET    /api/projects/:id/merge/status   合并状态
```

### WebSocket 事件

```
# 连接
WS /ws/projects/:id

# 推送事件
agent.status_changed    Agent 状态变更
agent.message           Agent 发来消息
task.started            任务开始
task.progress           任务进度更新
task.completed          任务完成
task.failed             任务失败
file.changed            文件变更
review.result           审查结果
merge.status            合并状态
```

---

## 七、项目结构

```
macdp/
├── cmd/macdp/main.go                # CLI 入口
├── internal/
│   ├── api/
│   │   ├── server.go                # Gin HTTP 服务
│   │   ├── handlers.go              # REST handlers
│   │   ├── websocket.go             # WebSocket 管理
│   │   └── middleware.go            # 中间件
│   ├── service/
│   │   ├── project.go               # 项目管理
│   │   ├── agent.go                 # Agent 管理
│   │   ├── task.go                  # 任务管理
│   │   ├── chat.go                  # 对话管理
│   │   ├── code_collab.go           # 代码协作
│   │   └── review.go                # 审查流水线
│   ├── agent/
│   │   ├── connector.go             # AgentConnector 接口
│   │   ├── registry.go              # Agent 注册表
│   │   ├── hermes.go                # Hermes 连接器
│   │   ├── openclaw.go              # OpenClaw 连接器
│   │   ├── codex.go                 # Codex 连接器
│   │   └── claude_code.go           # Claude Code 连接器
│   ├── router/
│   │   └── message.go               # 消息路由器
│   ├── store/
│   │   ├── db.go                    # SQLite 初始化
│   │   ├── project.go               # 项目 CRUD
│   │   ├── task.go                  # 任务 CRUD
│   │   ├── agent.go                 # Agent CRUD
│   │   └── chat.go                  # 聊天记录
│   └── config/
│       └── config.go                # 配置
├── web/                             # React 前端
│   ├── src/
│   │   ├── pages/
│   │   │   ├── Dashboard.tsx
│   │   │   ├── ProjectManager.tsx
│   │   │   ├── AgentChat.tsx
│   │   │   ├── Kanban.tsx
│   │   │   └── CodeReview.tsx
│   │   ├── components/
│   │   │   ├── AgentStatus.tsx
│   │   │   ├── TaskCard.tsx
│   │   │   ├── ChatBubble.tsx
│   │   │   ├── DiffViewer.tsx
│   │   │   └── DagGraph.tsx
│   │   └── hooks/
│   │       └── useWebSocket.ts
│   └── package.json
├── configs/
│   └── macdp.yaml
├── go.mod
└── go.sum
```

---

## 八、实现优先级

### Phase 1: 核心骨架 (本次)
1. Gin HTTP 服务 + WebSocket
2. AgentConnector 接口 + Hermes/OpenClaw 连接器
3. 项目/任务 CRUD
4. SQLite 存储层
5. 消息路由器

### Phase 2: 交互功能
6. Agent 对话面板 (WebSocket 实时)
7. 任务看板 (拖拽分配)
8. 代码协作 (文件变更通知)
9. 审查流水线

### Phase 3: 前端 UI
10. React Dashboard
11. Agent Chat 面板
12. Kanban 看板
13. 科幻风 UI 主题

### Phase 4: 高级功能
14. 甘特图/DAG 可视化
15. 成本追踪
16. 批量操作
17. 插件系统
