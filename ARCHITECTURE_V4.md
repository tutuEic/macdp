# MACDP v4 架构设计

> 自然语言驱动的多 Agent 协同开发平台
> 用户说需求，MACDP 自动调度 Agent 团队完成开发

---

## 一、核心理念

**用户只需要说一句话，剩下的交给 MACDP。**

```
用户: "用 FastAPI + React 做一个带 JWT 认证的 Todo 应用"

MACDP 自动完成:
  1. 理解需求，分解为 8 个子任务
  2. 分析依赖关系，生成任务 DAG
  3. 为每个任务匹配最合适的 Agent
  4. 并行调度 Agent 执行
  5. Agent 间自动共享代码和上下文
  6. 完成后自动交叉审查
  7. 审查通过自动合并
  8. 用户在 Dashboard 看到全过程，随时可以介入
```

---

## 二、系统架构

```
┌─────────────────────────────────────────────────────────────────┐
│                         用户层                                   │
│                                                                  │
│  自然语言输入: "做一个带认证的Todo应用"                           │
│         │                                                        │
│         ▼                                                        │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                    UI 层 (React)                         │    │
│  │                                                          │    │
│  │  ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐           │    │
│  │  │Dashboard│ │ Agent  │ │Kanban  │ │ Code   │           │    │
│  │  │ 仪表盘  │ │ Chat   │ │ 看板   │ │ Review │           │    │
│  │  └────────┘ └────────┘ └────────┘ └────────┘           │    │
│  │  ┌────────┐ ┌────────┐ ┌────────┐                      │    │
│  │  │Project │ │ DAG    │ │ Cost   │                      │    │
│  │  │Manager │ │ Graph  │ │ Report │                      │    │
│  │  └────────┘ └────────┘ └────────┘                      │    │
│  └──────────────────────────┬──────────────────────────────┘    │
│                             │ WebSocket + REST                   │
└─────────────────────────────┼────────────────────────────────────┘
                              │
┌─────────────────────────────▼────────────────────────────────────┐
│                     Go 后端 (Gin + Eino)                         │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                   Task Planner                            │   │
│  │  LLM 驱动的需求分解                                       │   │
│  │  输入: 用户自然语言                                        │   │
│  │  输出: 结构化任务 DAG + Agent 推荐                         │   │
│  │  支持: DeepSeek / GPT-4o / Claude / 任意 OpenAI 兼容模型  │   │
│  └──────────────────────────┬───────────────────────────────┘   │
│                              │                                    │
│  ┌──────────────────────────▼───────────────────────────────┐   │
│  │                  Orchestrator (编排器)                     │   │
│  │                                                           │   │
│  │  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐        │   │
│  │  │ DAG Engine  │ │  Scheduler  │ │   Merger    │        │   │
│  │  │ Eino Graph  │ │  并行调度    │ │  结果合并    │        │   │
│  │  │ 拓扑排序    │ │  Agent 匹配  │ │  Git 合并   │        │   │
│  │  └─────────────┘ └─────────────┘ └─────────────┘        │   │
│  │                                                           │   │
│  │  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐        │   │
│  │  │  Context    │ │   Review    │ │   EventBus  │        │   │
│  │  │  Bridge     │ │  Pipeline   │ │  事件总线    │        │   │
│  │  │  上下文传递  │ │  交叉审查    │ │  实时推送    │        │   │
│  │  └─────────────┘ └─────────────┘ └─────────────┘        │   │
│  └───────────────────────────────────────────────────────────┘   │
│                              │                                    │
│  ┌──────────────────────────▼───────────────────────────────┐   │
│  │               Agent Gateway (Agent 网关)                  │   │
│  │                                                           │   │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐   │   │
│  │  │  Hermes  │ │ OpenClaw │ │  Codex   │ │  Claude  │   │   │
│  │  │Connector │ │Connector │ │Connector │ │Connector │   │   │
│  │  └────┬─────┘ └────┬─────┘ └────┬─────┘ └────┬─────┘   │   │
│  └───────┼────────────┼────────────┼────────────┼───────────┘   │
│          │            │            │            │                 │
│  ┌───────▼────────────▼────────────▼────────────▼───────────┐   │
│  │                    Data Layer                             │   │
│  │  SQLite: 项目 / 任务 / Agent / 消息 / 审查 / 文件变更    │   │
│  └───────────────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────────────┘
          │            │            │            │
          ▼            ▼            ▼            ▼
    ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐
    │  Hermes  │ │ OpenClaw │ │  Codex   │ │  Claude  │
    │  Agent   │ │ Gateway  │ │  CLI     │ │ Code CLI │
    │ (外部)   │ │ (外部)   │ │ (外部)   │ │ (外部)   │
    └──────────┘ └──────────┘ └──────────┘ └──────────┘
```

---

## 三、核心流程

### 3.1 自动模式（默认）

```
用户输入: "用 FastAPI + React 做一个带 JWT 认证的 Todo 应用"

┌─────────────────────────────────────────────────────────────┐
│ Step 1: Task Planner (LLM 分解)                             │
│                                                              │
│  输入: 用户需求 + 项目上下文                                  │
│  LLM: DeepSeek V4 Pro (或配置的任意模型)                     │
│  输出:                                                       │
│  ┌────────────────────────────────────────────────────────┐ │
│  │ {                                                      │ │
│  │   "tasks": [                                           │ │
│  │     {"id":"T1", "title":"项目初始化",                   │ │
│  │      "agent":"hermes", "depends_on":[]},               │ │
│  │     {"id":"T2", "title":"数据库Schema",                 │ │
│  │      "agent":"hermes", "depends_on":["T1"]},           │ │
│  │     {"id":"T3", "title":"用户注册API",                  │ │
│  │      "agent":"claude-code", "depends_on":["T2"]},      │ │
│  │     {"id":"T4", "title":"JWT中间件",                    │ │
│  │      "agent":"claude-code", "depends_on":["T2"]},      │ │
│  │     {"id":"T5", "title":"React初始化",                  │ │
│  │      "agent":"codex", "depends_on":["T1"]},            │ │
│  │     {"id":"T6", "title":"登录页面",                     │ │
│  │      "agent":"codex", "depends_on":["T3","T5"]},       │ │
│  │     {"id":"T7", "title":"集成测试",                     │ │
│  │      "agent":"hermes", "depends_on":["T3","T4","T6"]}, │ │
│  │     {"id":"T8", "title":"API文档",                      │ │
│  │      "agent":"hermes", "depends_on":["T3","T4"]}       │ │
│  │   ]                                                    │ │
│  │ }                                                      │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│ Step 2: Scheduler (自动调度)                                │
│                                                              │
│  构建 DAG:                                                   │
│  ┌───┐                                                       │
│  │T1│──┬──→ T2 ──┬──→ T3 ──┬──→ T6 ──→ T7                 │
│  └───┘  │        │        │                                 │
│         └──→ T5 ─┘   T4 ──┤──→ T7                          │
│                            └──→ T8                          │
│                                                              │
│  执行计划:                                                   │
│  Layer 0: [T1]                hermes                         │
│  Layer 1: [T2, T5]            hermes + codex (并行)          │
│  Layer 2: [T3, T4]            claude + claude (并行)         │
│  Layer 3: [T6, T8]            codex + hermes (并行)          │
│  Layer 4: [T7]                hermes                         │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│ Step 3: Agent 执行 (并行)                                   │
│                                                              │
│  时间线:                                                     │
│  0min  5min  10min  15min  20min  25min  30min              │
│  ├──────┤                                                      │
│  │ T1   │ Hermes: 初始化项目结构                              │
│  ├──────┼────────┤                                            │
│  │      │ T2     │ Hermes: 创建数据库表                       │
│  │      ├────────┤                                            │
│  │      │ T5     │ Codex: 初始化 React                       │
│  │      ├────────┼────────┤                                   │
│  │      │        │ T3     │ Claude: 实现注册 API              │
│  │      │        ├────────┤                                   │
│  │      │        │ T4     │ Claude: 实现 JWT 中间件           │
│  │      │        ├────────┼────────┤                          │
│  │      │        │        │ T6     │ Codex: 实现登录页面      │
│  │      │        │        ├────────┤                          │
│  │      │        │        │ T8     │ Hermes: 生成 API 文档    │
│  │      │        │        ├────────┼────────┤                 │
│  │      │        │        │        │ T7     │ Hermes: 集成测试 │
│  │      │        │        │        ├────────┤                 │
│                                                              │
│  Agent 间协作:                                               │
│  T3 完成 → 自动将 API schema 共享给 T6 (前端需要)            │
│  T4 完成 → 自动将 JWT 配置共享给 T3, T6                      │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│ Step 4: Review + Merge                                       │
│                                                              │
│  每个任务完成后:                                              │
│  1. 自审 (执行 Agent 自己检查)                               │
│  2. 交叉审查 (另一个 Agent 审查)                             │
│  3. 自动测试 (lint + type-check + test)                      │
│                                                              │
│  全部完成后:                                                  │
│  4. 合并所有分支 → main                                      │
│  5. 运行集成测试                                             │
│  6. 生成项目报告                                             │
│  7. 通知用户                                                 │
└─────────────────────────────────────────────────────────────┘
```

### 3.2 用户介入模式

用户可以随时介入自动流程：

```
场景 1: 用户在看板上重新分配任务
  "T3 不给 Claude，改给 Hermes 做"
  → MACDP 暂停 T3，重新分配给 Hermes

场景 2: 用户在对话中直接指导 Agent
  用户 → Claude: "注册接口要用 Pydantic v2，不是 v1"
  → Claude 按照用户指示修改

场景 3: 用户添加新任务
  用户: "再加一个管理员后台"
  → Task Planner 生成新任务，插入 DAG

场景 4: 用户暂停/恢复
  用户: "暂停前端任务，先专注后端"
  → MACDP 暂停 T5, T6，重新调度

场景 5: 用户审查决策
  审查发现问题 → 用户选择:
    [自动修复] [手动修复] [忽略] [重新分配]
```

---

## 四、核心模块设计

### 4.1 Task Planner (任务规划器)

```go
type TaskPlanner struct {
    llm     model.ChatModel  // 可配置任意 LLM
    store   *Store
    project *Project
}

// Plan 将用户需求分解为任务 DAG
func (tp *TaskPlanner) Plan(ctx context.Context, requirement string) (*PlanResult, error) {
    // 1. 收集项目上下文
    projectCtx := tp.buildProjectContext()
    
    // 2. 调用 LLM 分解任务
    prompt := fmt.Sprintf(taskPlanPrompt, projectCtx, requirement)
    resp, err := tp.llm.Generate(ctx, []*schema.Message{
        schema.SystemMessage(prompt),
        schema.UserMessage(requirement),
    }, model.WithJSONOutput())
    
    // 3. 解析为结构化任务
    plan := parsePlanResult(resp.Content)
    
    // 4. 验证依赖关系（无环）
    if err := validateDAG(plan.Tasks); err != nil {
        return nil, err
    }
    
    // 5. 为每个任务创建 Git worktree
    for _, t := range plan.Tasks {
        worktree, _ := tp.gitMgr.Create(t.ID, "main")
        t.Worktree = worktree
    }
    
    return plan, nil
}
```

### 4.2 Scheduler (调度器)

```go
type Scheduler struct {
    dag       *DAG
    agents    *AgentRegistry
    planner   *TaskPlanner
    bridge    *ContextBridge
    bus       *EventBus
    mu        sync.RWMutex
}

// Run 自动调度执行所有任务
func (s *Scheduler) Run(ctx context.Context) error {
    for !s.dag.IsComplete() {
        // 1. 获取可执行任务（依赖已完成）
        ready := s.dag.GetReadyTasks()
        
        // 2. 并行执行
        var wg sync.WaitGroup
        for _, task := range ready {
            wg.Add(1)
            go func(t *Task) {
                defer wg.Done()
                s.executeTask(ctx, t)
            }(task)
        }
        wg.Wait()
    }
    return nil
}

func (s *Scheduler) executeTask(ctx context.Context, task *Task) {
    // 1. 选择 Agent（优先用推荐的，fallback 到其他）
    agent := s.selectAgent(task)
    
    // 2. 构建上下文（注入依赖任务的产出）
    context := s.bridge.BuildContext(task)
    
    // 3. 更新状态
    task.Status = StatusRunning
    task.AssignedAgent = agent.Name()
    s.bus.Emit("task.started", task)
    
    // 4. 执行
    events, _ := agent.ExecuteTask(ctx, task, context)
    for event := range events {
        switch event.Type {
        case "progress":
            task.Progress = event.Progress
            s.bus.Emit("task.progress", task)
        case "output":
            task.Output += event.Content
            s.bus.Emit("agent.message", event)
        case "file_change":
            s.handleFileChange(task, event)
        case "complete":
            task.Status = StatusDone
            s.bus.Emit("task.completed", task)
        case "error":
            task.Status = StatusFailed
            s.bus.Emit("task.failed", task)
        }
    }
    
    // 5. 触发审查
    s.reviewPipeline.Run(ctx, task)
    
    // 6. 通知依赖此任务的其他任务
    s.dag.NotifyDependents(task.ID)
}
```

### 4.3 Context Bridge (上下文桥)

```go
// ContextBridge 负责在 Agent 间传递上下文
type ContextBridge struct {
    store *Store
}

// BuildContext 为任务构建完整上下文
func (cb *BuildContext) BuildContext(task *Task) string {
    var ctx strings.Builder
    
    // 1. 项目约定
    ctx.WriteString("## 项目约定\n")
    ctx.WriteString(cb.store.GetProjectConventions(task.ProjectID))
    ctx.WriteString("\n\n")
    
    // 2. 依赖任务的产出
    for _, depID := range task.DependsOn {
        dep := cb.store.GetTask(depID)
        ctx.WriteString(fmt.Sprintf("## %s 的产出\n", dep.Title))
        ctx.WriteString(dep.Output)
        ctx.WriteString("\n\n")
        
        // 注入变更的文件列表
        if len(dep.FilesChanged) > 0 {
            ctx.WriteString("变更文件:\n")
            for _, f := range dep.FilesChanged {
                ctx.WriteString(fmt.Sprintf("- %s\n", f))
            }
        }
    }
    
    // 3. 共享文件（其他 Agent 主动共享的）
    shared := cb.store.GetSharedFiles(task.ProjectID)
    for _, f := range shared {
        ctx.WriteString(fmt.Sprintf("## 共享文件: %s\n```\n%s\n```\n\n", f.Path, f.Content))
    }
    
    // 4. 当前模块的文件结构
    ctx.WriteString(fmt.Sprintf("## 当前模块: %s\n", task.Module))
    ctx.WriteString(cb.getModuleFileTree(task))
    
    return ctx.String()
}
```

### 4.4 Agent Connector (Agent 连接器)

```go
// AgentConnector 连接外部 Agent 服务
type AgentConnector interface {
    Name() string
    Type() AgentType
    Status() AgentStatus
    
    // 连接管理
    Connect(ctx context.Context, config AgentConfig) error
    Disconnect() error
    Ping() error
    
    // 任务执行
    ExecuteTask(ctx context.Context, task *Task, context string) (<-chan *TaskEvent, error)
    CancelTask(taskID string) error
    
    // 对话交互
    SendMessage(ctx context.Context, msg string) (<-chan *ChatMessage, error)
}

// TaskEvent 是任务执行过程中的事件
type TaskEvent struct {
    Type     string `json:"type"`     // progress, output, file_change, complete, error
    TaskID   string `json:"task_id"`
    Agent    string `json:"agent"`
    Content  string `json:"content"`
    Progress int    `json:"progress"`
    File     string `json:"file,omitempty"`
}

// HermesConnector 通过 HTTP API 连接 Hermes Agent
type HermesConnector struct {
    baseURL string
    client  *http.Client
    status  AgentStatus
}

func (h *HermesConnector) ExecuteTask(ctx context.Context, task *Task, context string) (<-chan *TaskEvent, error) {
    // 方式 1: HTTP API (如果 Hermes Gateway 运行中)
    // POST /api/agent/execute {prompt, workdir, ...}
    
    // 方式 2: CLI 调用
    // hermes chat -q "任务描述" --workdir /path/to/worktree
    
    // 返回事件流 (SSE 或 WebSocket)
    events := make(chan *TaskEvent, 100)
    go h.streamEvents(ctx, task, context, events)
    return events, nil
}

// OpenClawConnector 通过 HTTP API 连接 OpenClaw Gateway
type OpenClawConnector struct {
    baseURL string
    client  *http.Client
}

func (oc *OpenClawConnector) ExecuteTask(ctx context.Context, task *Task, context string) (<-chan *TaskEvent, error) {
    // POST http://localhost:18789/api/agent
    // OpenClaw 的 agent 执行接口
}

// CodexConnector 通过 CLI 连接 Codex
type CodexConnector struct {
    binary string // codex
}

func (c *CodexConnector) ExecuteTask(ctx context.Context, task *Task, context string) (<-chan *TaskEvent, error) {
    // codex exec "任务描述" --full-auto
    // 通过 stdout 流获取事件
}

// ClaudeCodeConnector 通过 CLI 连接 Claude Code
type ClaudeCodeConnector struct {
    binary string // claude
}

func (cc *ClaudeCodeConnector) ExecuteTask(ctx context.Context, task *Task, context string) (<-chan *TaskEvent, error) {
    // claude -p "任务描述" --output-format stream-json
    // 通过 stream-json 获取实时事件
}
```

---

## 五、EventBus (事件总线)

```go
// EventBus 是系统的核心事件总线
// 所有模块通过事件通信，松耦合
type EventBus struct {
    mu          sync.RWMutex
    subscribers map[string][]EventHandler
    wsHub       *WebSocketHub  // 同时推送给前端
}

type EventHandler func(event *Event)

type Event struct {
    Type      string    `json:"type"`
    Source    string    `json:"source"`    // planner, scheduler, agent, user
    Target    string    `json:"target"`    // agent name, "ui", "all"
    Payload   any       `json:"payload"`
    Timestamp time.Time `json:"timestamp"`
}

// 事件类型
const (
    // 任务生命周期
    EventTaskCreated   = "task.created"
    EventTaskAssigned  = "task.assigned"
    EventTaskStarted   = "task.started"
    EventTaskProgress  = "task.progress"
    EventTaskCompleted = "task.completed"
    EventTaskFailed    = "task.failed"
    
    // Agent 状态
    EventAgentConnected    = "agent.connected"
    EventAgentDisconnected = "agent.disconnected"
    EventAgentStatusChanged = "agent.status_changed"
    
    // Agent 消息
    EventAgentMessage = "agent.message"
    EventAgentOutput  = "agent.output"
    
    // 文件协作
    EventFileChanged  = "file.changed"
    EventFileShared   = "file.shared"
    EventConflictDetected = "conflict.detected"
    
    // 审查
    EventReviewStarted = "review.started"
    EventReviewResult  = "review.result"
    
    // 用户操作
    EventUserCommand = "user.command"
    EventUserChat    = "user.chat"
    
    // 系统
    EventPlanCreated = "plan.created"
    EventMergeStatus = "merge.status"
)

// Emit 发布事件，同时推送给 WebSocket 客户端
func (eb *EventBus) Emit(eventType string, payload any) {
    event := &Event{
        Type:      eventType,
        Payload:   payload,
        Timestamp: time.Now(),
    }
    
    // 通知订阅者
    eb.mu.RLock()
    for _, handler := range eb.subscribers[eventType] {
        go handler(event)
    }
    eb.mu.RUnlock()
    
    // 推送给前端 WebSocket
    eb.wsHub.Broadcast(event)
}
```

---

## 六、前端 UI 设计

### 6.1 主界面布局

```
┌─────────────────────────────────────────────────────────────────┐
│  MACDP                           [项目: todo-app ▼] [+ 新建]   │
├──────┬──────────────────────────────────────────────────────────┤
│      │                                                          │
│ 侧边 │  ┌─ 需求输入 ─────────────────────────────────────────┐ │
│ 栏   │  │ [输入框: 请输入开发需求...]              [开始执行] │ │
│      │  └────────────────────────────────────────────────────┘ │
│ 🏠   │                                                          │
│ 仪表 │  ┌─ DAG 可视化 ──────────┐ ┌─ Agent 状态 ────────────┐ │
│ 盘   │  │                       │ │                          │ │
│      │  │   ┌───┐               │ │ 🟢 Hermes   空闲        │ │
│ 📋   │  │   │T1 │               │ │ 🟡 Claude   执行 T3     │ │
│ 看板 │  │   └─┬─┘               │ │ 🟢 Codex    空闲        │ │
│      │  │   ┌─┴──┐              │ │ 🔴 OpenClaw 离线        │ │
│ 💬   │  │   │T2  │T5            │ │                          │ │
│ 对话 │  │   └─┬──┘              │ └──────────────────────────┘ │
│      │  │   ┌─┴───┐             │                              │
│ 📁   │  │   │T3  │T4            │ ┌─ 任务进度 ──────────────┐ │
│ 文件 │  │   └──┬──┘             │ │                          │ │
│      │  │   ┌──┴──┐             │ │ T1 ████ 100%  ✓         │ │
│ 🔍   │  │   │T6  │T8            │ │ T2 ████ 100%  ✓         │ │
│ 审查 │  │   └──┬──┘             │ │ T3 ████░░░░  60%  ▶     │ │
│      │  │   ┌──┴──┐             │ │ T4 ░░░░░░░░   0%        │ │
│ 📊   │  │   │  T7 │             │ │ T5 ████ 100%  ✓         │ │
│ 报告 │  │   └─────┘             │ │ T6 ░░░░░░░░   0%  等待  │ │
│      │  │                       │ │ T7 ░░░░░░░░   0%  等待  │ │
│ ⚙️   │  └───────────────────────┘ │ T8 ░░░░░░░░   0%  等待  │ │
│ 设置 │                            └──────────────────────────┘ │
│      │                                                          │
│      │  ┌─ 最近消息 ─────────────────────────────────────────┐ │
│      │  │ [14:32] Claude: 完成用户注册API，已提交到分支      │ │
│      │  │ [14:30] Hermes: T1 项目初始化完成                  │ │
│      │  │ [14:28] 系统: T3 已分配给 Claude                   │ │
│      │  │ [14:25] 系统: 任务分解完成，共 8 个任务            │ │
│      │  └────────────────────────────────────────────────────┘ │
└──────┴──────────────────────────────────────────────────────────┘
```

### 6.2 Agent 对话面板

```
┌─────────────────────────────────────────────────────────────────┐
│  Agent 对话                    [Hermes ▼]  [暂停] [取消] [转给] │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  [14:30] 系统: T3 "用户注册API" 已分配给 Claude                │
│                                                                 │
│  [14:30] Claude: 我来实现用户注册 API。                         │
│           首先创建 Pydantic 模型...                             │
│                                                                 │
│  [14:31] Claude: 📄 创建 src/models/user.py                    │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │ from pydantic import BaseModel, EmailStr                 │  │
│  │                                                          │  │
│  │ class UserCreate(BaseModel):                             │  │
│  │     email: EmailStr                                      │  │
│  │     password: str                                        │  │
│  │     name: str                                            │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                 │
│  [14:31] Claude: 📄 创建 src/api/auth/register.py              │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │ @router.post("/auth/register")                           │  │
│  │ async def register(user: UserCreate):                    │  │
│  │     hashed = hash_password(user.password)                │  │
│  │     db_user = await create_user(user, hashed)            │  │
│  │     return {"id": db_user.id, "email": db_user.email}    │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                 │
│  [14:32] Claude: ✅ 注册 API 完成，已提交到 feature/T3 分支    │
│           变更文件: src/models/user.py, src/api/auth/register.py│
│                                                                 │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │ [输入消息给 Claude...]                        [发送] [📎] │  │
│  └──────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 七、项目结构

```
macdp/
├── cmd/macdp/main.go
├── internal/
│   ├── api/
│   │   ├── server.go              # Gin + WebSocket
│   │   ├── handlers.go            # REST handlers
│   │   └── ws.go                  # WebSocket hub
│   ├── planner/
│   │   ├── planner.go             # Task Planner (LLM)
│   │   └── prompts.go             # 提示词
│   ├── orchestrator/
│   │   ├── scheduler.go           # 并行调度器
│   │   ├── context_bridge.go      # 上下文传递
│   │   └── merger.go              # 结果合并
│   ├── agent/
│   │   ├── connector.go           # AgentConnector 接口
│   │   ├── registry.go            # Agent 注册表
│   │   ├── hermes.go              # Hermes 连接器
│   │   ├── openclaw.go            # OpenClaw 连接器
│   │   ├── codex.go               # Codex 连接器
│   │   └── claude_code.go         # Claude Code 连接器
│   ├── event/
│   │   └── bus.go                 # EventBus
│   ├── review/
│   │   └── pipeline.go            # 审查流水线
│   ├── git/
│   │   ├── worktree.go            # Worktree 管理
│   │   └── merge.go               # 合并策略
│   ├── store/
│   │   ├── db.go                  # SQLite
│   │   └── models.go              # 数据模型
│   ├── llm/
│   │   └── client.go              # LLM 客户端 (OpenAI 兼容)
│   └── config/
│       └── config.go
├── web/                           # React 前端
│   └── src/
│       ├── pages/
│       │   ├── Dashboard.tsx
│       │   ├── Kanban.tsx
│       │   ├── AgentChat.tsx
│       │   ├── ProjectManager.tsx
│       │   └── CodeReview.tsx
│       └── components/
│           ├── DagGraph.tsx
│           ├── TaskCard.tsx
│           └── ChatBubble.tsx
├── configs/macdp.yaml
├── go.mod
└── go.sum
```

---

## 八、实现路线

### Phase 1: 核心骨架
1. Gin HTTP + WebSocket 服务
2. AgentConnector 接口 + Hermes 连接器
3. 任务/项目 CRUD
4. EventBus
5. SQLite 存储

### Phase 2: 智能调度
6. Task Planner (LLM 分解)
7. Scheduler (并行调度)
8. Context Bridge (上下文传递)
9. OpenClaw + Codex + Claude 连接器

### Phase 3: 协作功能
10. Agent 对话 (WebSocket 实时)
11. 代码协作 (文件变更通知)
12. 审查流水线
13. Git 合并

### Phase 4: 前端 UI
14. React Dashboard
15. DAG 可视化
16. Kanban 看板
17. Agent Chat 面板
18. 科幻风 UI
