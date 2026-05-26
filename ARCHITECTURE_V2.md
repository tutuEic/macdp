# MACDP v2 架构设计

> 基于 Eino DAG 引擎 + DeepSeek V4 Pro + 多 Agent 协同

---

## 一、整体架构

```
┌─────────────────────────────────────────────────────────────────┐
│                        用户入口层                                │
│   CLI (cobra)  ───  Web API (gin)  ───  Bot (Telegram/Discord) │
└──────────────────────────┬──────────────────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────────────┐
│                    Task Planner (任务规划器)                      │
│   LLM: DeepSeek V4 Pro                                          │
│   输入: 用户自然语言需求                                          │
│   输出: 结构化 Task DAG (JSON)                                    │
│   能力: 需求分析 → 子任务拆分 → 依赖关系 → Agent 匹配             │
└──────────────────────────┬──────────────────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────────────┐
│              Multi-Agent Orchestrator (多 Agent 编排器)           │
│                                                                  │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │ Message Bus  │  │  Scheduler  │  │   Reviewer  │             │
│  │ (消息总线)    │  │ (调度器)     │  │ (审查器)     │             │
│  │ pub/sub      │  │ Eino DAG    │  │ cross-review │             │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘             │
│         │                │                │                      │
│  ┌──────▼────────────────▼────────────────▼──────┐              │
│  │           Eino Compose DAG Engine              │              │
│  │  Graph[TaskInput, TaskResult]                  │              │
│  │  ├── AddLambdaNode (每个任务一个节点)           │              │
│  │  ├── AddEdge (依赖关系)                        │              │
│  │  ├── AddBranch (条件路由)                      │              │
│  │  └── Compile → Run (层级并行 goroutine)        │              │
│  └───────────────────────────────────────────────┘              │
└──────────────────────────┬──────────────────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────────────┐
│                    Agent Layer (Agent 层)                        │
│                                                                  │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐            │
│  │ AgentAdapter  │ │ AgentAdapter  │ │ AgentAdapter  │           │
│  │  Hermes CLI   │ │  Codex CLI    │ │ Claude Code   │          │
│  └──────┬───────┘ └──────┬───────┘ └──────┬───────┘           │
│         │                │                │                      │
│  ┌──────▼────────────────▼────────────────▼──────┐              │
│  │         Agent Toolkit (共享工具层)              │              │
│  │  Git Worktree │ Shell │ File IO │ LLM Call    │              │
│  └───────────────────────────────────────────────┘              │
└──────────────────────────┬──────────────────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────────────┐
│                    State Layer (状态层)                          │
│  SQLite (任务状态) ─── Checkpoint (断点续传) ─── Session Log     │
└─────────────────────────────────────────────────────────────────┘
```

---

## 二、核心模块设计

### 2.1 Task Planner — DeepSeek V4 Pro 驱动

```go
// planner.go — LLM 驱动的任务分解器

type TaskPlanner struct {
    llm    model.ChatModel  // DeepSeek V4 Pro
    prompt string            // 系统提示词
}

type PlanRequest struct {
    Description string            `json:"description"`  // 用户需求
    RepoPath    string            `json:"repo_path"`    // 项目路径
    Context     map[string]string `json:"context"`      // 额外上下文
}

type PlanResult struct {
    Tasks  []PlannedTask `json:"tasks"`
    Summary string       `json:"summary"`
}

type PlannedTask struct {
    ID          string   `json:"id"`
    Title       string   `json:"title"`
    Description string   `json:"description"`
    DependsOn   []string `json:"depends_on"`
    Agent       string   `json:"agent"`        // 推荐的 agent
    Priority    int      `json:"priority"`
    MaxTurns    int      `json:"max_turns"`
}

// Plan 调用 DeepSeek 分解任务
func (p *TaskPlanner) Plan(ctx context.Context, req *PlanRequest) (*PlanResult, error) {
    messages := []*schema.Message{
        schema.SystemMessage(taskPlannerPrompt),
        schema.UserMessage(formatPlanRequest(req)),
    }
    
    resp, err := p.llm.Generate(ctx, messages,
        model.WithJSONSchemaOutput(planResultSchema),
    )
    if err != nil {
        return nil, err
    }
    
    return parsePlanResult(resp.Content)
}
```

**DeepSeek 系统提示词：**

```
你是一个软件项目任务分解专家。用户会描述一个开发需求，你需要：

1. 分析需求，拆分为可独立执行的子任务
2. 确定任务之间的依赖关系（哪些必须先完成）
3. 为每个任务推荐最合适的 Agent：
   - hermes: 调试、测试、通用任务、项目初始化
   - claude-code: 复杂编码、代码审查、重构
   - codex: 快速原型、简单脚本、前端组件
4. 估算每个任务的复杂度（max_turns）

输出严格的 JSON 格式。
```

### 2.2 Eino DAG Engine — 任务执行引擎

```go
// engine.go — 基于 Eino compose 的 DAG 执行引擎

import (
    "github.com/cloudwego/eino/compose"
    "github.com/cloudwego/eino/schema"
)

// TaskInput 是 DAG 的输入类型
type TaskInput struct {
    TaskID  string
    Task    *task.Task
    Context map[string]string  // 上游任务的输出
}

// TaskResult 是 DAG 的输出类型
type TaskResult struct {
    TaskID       string
    Success      bool
    Summary      string
    FilesChanged []string
    CostUSD      float64
}

// DAGEngine 使用 Eino 的 compose.Graph 构建和执行任务 DAG
type DAGEngine struct {
    registry *agent.Registry
    gitMgr   *git.WorktreeManager
}

// BuildGraph 将 task.DAG 转换为 Eino compose.Graph
func (e *DAGEngine) BuildGraph(dag *task.DAG) (*compose.Graph[map[string]*TaskInput, map[string]*TaskResult], error) {
    g := compose.NewGraph[map[string]*TaskInput, map[string]*TaskResult]()
    
    // 为每个任务创建一个 Lambda 节点
    for id, t := range dag.Tasks {
        taskID := id
        task := t
        
        // Lambda: 执行单个任务
        fn := func(ctx context.Context, input *TaskInput) (*TaskResult, error) {
            return e.executeTask(ctx, task, input)
        }
        
        g.AddLambdaNode(taskID, fn)
        
        // 添加依赖边
        if len(task.DependsOn) == 0 {
            // 无依赖: 从 START 连入
            g.AddEdge(compose.START, taskID)
        } else {
            // 有依赖: 从依赖任务连入
            for _, depID := range task.DependsOn {
                g.AddEdge(depID, taskID)
            }
        }
        
        // 所有任务连向 END
        g.AddEdge(taskID, compose.END)
    }
    
    return g, nil
}

// Run 编译并执行 DAG
func (e *DAGEngine) Run(ctx context.Context, dag *task.DAG) (map[string]*TaskResult, error) {
    g, err := e.BuildGraph(dag)
    if err != nil {
        return nil, err
    }
    
    // 编译: 验证 DAG、拓扑排序、准备执行
    compiled, err := g.Compile(ctx)
    if err != nil {
        return nil, fmt.Errorf("graph compile failed: %w", err)
    }
    
    // 构建输入
    inputs := make(map[string]*TaskInput)
    for id, t := range dag.Tasks {
        inputs[id] = &TaskInput{TaskID: id, Task: t}
    }
    
    // 执行: Eino 自动处理层级并行
    // 同一层的节点用 goroutine 并发执行
    // 层间等待依赖完成
    results, err := compiled.Run(ctx, inputs)
    if err != nil {
        return nil, fmt.Errorf("graph run failed: %w", err)
    }
    
    return results, nil
}

// executeTask 执行单个任务
func (e *DAGEngine) executeTask(ctx context.Context, t *task.Task, input *TaskInput) (*TaskResult, error) {
    // 1. 创建 worktree
    worktree, err := e.gitMgr.Create(t.ID, "main")
    if err != nil {
        return nil, err
    }
    t.Worktree = worktree
    
    // 2. 选择 agent
    adapter := e.registry.Get(t.Agent)
    if adapter == nil {
        adapter = e.registry.Get("hermes") // fallback
    }
    
    // 3. 构建请求（包含上游任务的输出作为上下文）
    req := &agent.TaskRequest{
        TaskID:      t.ID,
        Title:       t.Title,
        Description: t.Description,
        Workdir:     worktree,
        MaxTurns:    t.MaxTurns,
        Context:     input.Context,
    }
    
    // 4. 执行
    resp, err := adapter.Execute(ctx, req)
    if err != nil {
        return &TaskResult{TaskID: t.ID, Success: false, Error: err.Error()}, nil
    }
    
    // 5. 自动提交
    e.gitMgr.CommitAll(t.ID, fmt.Sprintf("feat: %s", t.Title))
    
    return &TaskResult{
        TaskID:       t.ID,
        Success:      resp.Success,
        Summary:      resp.Summary,
        FilesChanged: resp.FilesChanged,
        CostUSD:      resp.CostUSD,
    }, nil
}
```

### 2.3 Message Bus — 消息总线

```go
// messagebus.go — 借鉴 MetaGPT 的 Environment 模式

type MessageType string

const (
    MsgTaskStart    MessageType = "task_start"
    MsgTaskComplete MessageType = "task_complete"
    MsgTaskFailed   MessageType = "task_failed"
    MsgReviewResult MessageType = "review_result"
    MsgAgentEvent   MessageType = "agent_event"
    MsgUserCommand  MessageType = "user_command"
)

type Message struct {
    ID        string      `json:"id"`
    Type      MessageType `json:"type"`
    From      string      `json:"from"`       // agent name or "user"
    To        string      `json:"to"`         // agent name or "all"
    TaskID    string      `json:"task_id"`
    Payload   any         `json:"payload"`
    Timestamp time.Time   `json:"timestamp"`
}

type Subscriber func(msg *Message)

type MessageBus struct {
    mu          sync.RWMutex
    subscribers map[MessageType][]Subscriber
    history     []*Message
    historyMu   sync.RWMutex
}

func NewMessageBus() *MessageBus {
    return &MessageBus{
        subscribers: make(map[MessageType][]Subscriber),
    }
}

// Publish 发布消息，通知所有订阅者
func (bus *MessageBus) Publish(msg *Message) {
    msg.Timestamp = time.Now()
    
    // 记录历史
    bus.historyMu.Lock()
    bus.history = append(bus.history, msg)
    bus.historyMu.Unlock()
    
    // 通知订阅者
    bus.mu.RLock()
    defer bus.mu.RUnlock()
    
    // 精确匹配
    for _, sub := range bus.subscribers[msg.Type] {
        go sub(msg) // 异步通知，不阻塞
    }
    
    // 通配符 "all"
    for _, sub := range bus.subscribers["all"] {
        go sub(msg)
    }
}

// Subscribe 订阅特定类型的消息
func (bus *MessageBus) Subscribe(msgType MessageType, handler Subscriber) {
    bus.mu.Lock()
    defer bus.mu.Unlock()
    bus.subscribers[msgType] = append(bus.subscribers[msgType], handler)
}

// GetHistory 获取历史消息
func (bus *MessageBus) GetHistory(msgType MessageType, limit int) []*Message {
    bus.historyMu.RLock()
    defer bus.historyMu.RUnlock()
    
    var result []*Message
    for i := len(bus.history) - 1; i >= 0 && len(result) < limit; i-- {
        if msgType == "" || bus.history[i].Type == msgType {
            result = append(result, bus.history[i])
        }
    }
    return result
}
```

### 2.4 DeepSeek LLM Client

```go
// llm/deepseek.go — DeepSeek V4 Pro 客户端

import (
    "github.com/cloudwego/eino/components/model"
    "github.com/cloudwego/eino/schema"
)

type DeepSeekConfig struct {
    APIKey  string `yaml:"api_key"`
    BaseURL string `yaml:"base_url"`
    Model   string `yaml:"model"`
}

// NewDeepSeekModel 创建 DeepSeek ChatModel (兼容 Eino 接口)
func NewDeepSeekModel(cfg DeepSeekConfig) model.ChatModel {
    // 使用 Eino 的 OpenAI 兼容客户端
    // DeepSeek API 兼容 OpenAI 格式
    return openaimodel.NewChatModel(&openaimodel.Config{
        APIKey:  cfg.APIKey,
        BaseURL: cfg.BaseURL, // https://api.deepseek.com/v1
        Model:   cfg.Model,   // deepseek-chat (V4)
    })
}
```

### 2.5 Review Pipeline — 审查流水线

```go
// review/pipeline.go

type ReviewPipeline struct {
    llm      model.ChatModel  // DeepSeek for review analysis
    bus      *MessageBus
    registry *agent.Registry
}

type ReviewResult struct {
    TaskID    string         `json:"task_id"`
    Reviewer  string         `json:"reviewer"`
    Verdict   ReviewVerdict  `json:"verdict"`
    Comments  []ReviewComment `json:"comments"`
    Score     int            `json:"score"` // 0-100
}

type ReviewVerdict string
const (
    VerdictApprove         ReviewVerdict = "approve"
    VerdictRequestChanges  ReviewVerdict = "request_changes"
    VerdictNeedsDiscussion ReviewVerdict = "needs_discussion"
)

// Run 执行审查流水线
func (rp *ReviewPipeline) Run(ctx context.Context, t *task.Task, result *TaskResult) (*ReviewResult, error) {
    // 1. 自审 (agent 自己检查)
    selfReview := rp.selfReview(ctx, t, result)
    
    // 2. 交叉审查 (另一个 agent 审查)
    crossReview := rp.crossReview(ctx, t, result)
    
    // 3. LLM 综合分析
    finalReview := rp.synthesize(ctx, selfReview, crossReview)
    
    // 4. 发布审查结果
    rp.bus.Publish(&Message{
        Type:   MsgReviewResult,
        From:   "reviewer",
        TaskID: t.ID,
        Payload: finalReview,
    })
    
    return finalReview, nil
}
```

---

## 三、数据流

```
用户: "给项目添加用户认证系统"

Step 1: Task Planner (DeepSeek V4 Pro)
  输入: 用户需求 + 项目上下文
  输出: {
    tasks: [
      {id:"T1", title:"数据库schema", depends_on:[], agent:"hermes"},
      {id:"T2", title:"注册API", depends_on:["T1"], agent:"claude-code"},
      {id:"T3", title:"登录API", depends_on:["T1"], agent:"claude-code"},
      {id:"T4", title:"JWT中间件", depends_on:["T1"], agent:"claude-code"},
      {id:"T5", title:"前端登录页", depends_on:["T2","T3"], agent:"codex"},
      {id:"T6", title:"集成测试", depends_on:["T2","T3","T4"], agent:"hermes"}
    ]
  }

Step 2: Eino DAG Engine
  Graph 构建:
    START → T1
    T1 → T2, T1 → T3, T1 → T4 (并行!)
    T2 → T5, T3 → T5
    T2 → T6, T3 → T6, T4 → T6
    T5 → END, T6 → END

  执行:
    Layer 0: [T1]              ← hermes 执行
    Layer 1: [T2, T3, T4]      ← 3个 agent 并行! (goroutine)
    Layer 2: [T5, T6]          ← codex + hermes 并行

Step 3: Review Pipeline
  T2 完成 → Claude 交叉审查 T2
  T3 完成 → Claude 交叉审查 T3
  ...

Step 4: Git Merge
  所有 worktree 分支 → squash merge → main
```

---

## 四、项目结构 (v2)

```
macdp/
├── cmd/macdp/
│   └── main.go                    # CLI 入口
├── internal/
│   ├── planner/
│   │   ├── planner.go             # Task Planner (DeepSeek)
│   │   └── prompts.go             # 系统提示词
│   ├── engine/
│   │   ├── dag.go                 # Eino DAG 构建器
│   │   ├── executor.go            # 任务执行器
│   │   └── merger.go              # 结果合并
│   ├── agent/
│   │   ├── adapter.go             # AgentAdapter 接口
│   │   ├── registry.go            # Agent 注册表
│   │   ├── hermes.go              # Hermes CLI 适配器
│   │   ├── claude_code.go         # Claude Code 适配器
│   │   ├── codex.go               # Codex 适配器
│   │   └── opencode.go            # OpenCode 适配器
│   ├── messagebus/
│   │   └── bus.go                 # 消息总线
│   ├── review/
│   │   ├── pipeline.go            # 审查流水线
│   │   └── prompts.go             # 审查提示词
│   ├── git/
│   │   ├── worktree.go            # Worktree 管理
│   │   └── merge.go               # 合并策略
│   ├── llm/
│   │   └── deepseek.go            # DeepSeek 客户端
│   ├── store/
│   │   ├── sqlite.go              # SQLite 持久化
│   │   └── checkpoint.go          # 断点续传
│   ├── api/
│   │   ├── server.go              # HTTP + SSE
│   │   └── handlers.go            # API handlers
│   └── config/
│       └── config.go              # YAML 配置
├── configs/
│   ├── macdp.yaml                 # 默认配置
│   └── example_tasks.json         # 示例任务
├── go.mod
└── go.sum
```

---

## 五、依赖清单

```go
// go.mod
module github.com/tutuEic/macdp

go 1.24

require (
    github.com/cloudwego/eino v0.9.0          // DAG 引擎 + ADK
    github.com/gin-gonic/gin v1.10.0          // HTTP 框架
    github.com/gorilla/websocket v1.5.3        // WebSocket
    github.com/mattn/go-sqlite3 v1.14.22       // SQLite
    github.com/spf13/cobra v1.8.1              // CLI 框架
    gopkg.in/yaml.v3 v3.0.1                    // YAML 解析
)
```

---

## 六、与 v1 的区别

| 方面 | v1 (手写) | v2 (Eino 集成) |
|------|----------|----------------|
| DAG 引擎 | 自己写的拓扑排序 | Eino compose.Graph |
| 并发执行 | 手动 goroutine + semaphore | Eino 层级并行 (自动) |
| 任务规划 | 无 (手动 JSON) | DeepSeek V4 Pro 自动分解 |
| 消息传递 | 无 | MessageBus pub/sub |
| 流式输出 | 无 | Eino StreamReader |
| 审查 | 无 | ReviewPipeline (自审+交叉) |
| Checkpoint | 无 | Eino CheckPointStore |
| 条件路由 | 无 | Eino AddBranch |

---

## 七、实现优先级

### Phase 1: 核心引擎 (本次实现)
1. DeepSeek LLM 客户端 (llm/deepseek.go)
2. Task Planner (planner/planner.go)
3. Eino DAG Engine (engine/dag.go)
4. Message Bus (messagebus/bus.go)
5. 集成到 main.go

### Phase 2: 完善功能
6. Agent Adapters 更新 (适配新接口)
7. Review Pipeline
8. SQLite 持久化
9. Checkpoint 断点续传

### Phase 3: Web UI
10. React Dashboard
11. WebSocket 实时日志
12. 科幻风 UI
