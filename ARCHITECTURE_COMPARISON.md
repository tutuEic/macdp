# MACDP 架构对比分析：MetaGPT vs CrewAI vs Eino

## 一、三大框架核心对比

```
┌─────────────┬──────────────────┬──────────────────┬──────────────────┐
│             │ MetaGPT (68K⭐)  │ CrewAI (52K⭐)   │ Eino (11K⭐)     │
│             │ Python           │ Python           │ Go               │
├─────────────┼──────────────────┼──────────────────┼──────────────────┤
│ 核心模型     │ Role+Action+Env  │ Agent+Crew+Task  │ Graph+ADK        │
│ 并发模型     │ asyncio 队列     │ asyncio 单进程   │ goroutine 真并发 │
│ 任务分解     │ LLM 规划+人类SOP │ LLM 分解或预定义 │ 用户手建 DAG     │
│ Agent通信    │ 消息总线(Environment) │ 工具委派    │ Graph 边连接     │
│ 状态管理     │ RoleContext+Memory │ 4层Memory      │ context+Checkpoint│
│ 流式支持     │ 有限             │ 无               │ 一等公民          │
│ 代码执行     │ 沙盒             │ 无内置           │ 工具调用          │
│ 生产就绪     │ ★★★★           │ ★★★             │ ★★★★(字节出品)  │
└─────────────┴──────────────────┴──────────────────┴──────────────────┘
```

---

## 二、MetaGPT 架构深度分析

### 核心模式：消息驱动的多 Agent 环境

```python
# MetaGPT 的核心抽象
Role (角色)
  ├── RoleContext (运行时上下文)
  │   ├── env: Environment        ← 全局消息环境
  │   ├── memory: Memory          ← 消息记忆
  │   ├── msg_buffer: MessageQueue ← 消息缓冲区
  │   ├── watch: set[str]         ← 订阅的消息类型
  │   └── react_mode: REACT | BY_ORDER | PLAN_AND_ACT
  ├── Actions (行为列表)
  │   └── Action → ActionNode (LLM 调用单元)
  └── _think() → _act() 循环

Environment (环境)
  ├── MessageQueue (全局消息队列)
  ├── roles: dict[str, Role]     ← 注册的角色
  └── routing (消息路由)

Team (团队编排)
  ├── env: Environment
  ├── investment: float          ← 预算上限
  └── run(n_rounds)              ← 启动 N 轮对话
```

### 消息路由机制

```python
# MetaGPT 的消息流
Role._observe()     → 从 env 读取订阅的消息
Role._think()       → LLM 决定下一步动作
Role._act()         → 执行动作，产出 Message
Role.publish_message() → 发送到 Environment
Environment         → 根据路由规则分发到订阅者
```

### 三种 React 模式

| 模式 | 描述 | 适用场景 |
|------|------|---------|
| REACT | _think → _act 循环 | 通用任务 |
| BY_ORDER | 按预定义 Action 列表顺序执行 | 固定流程 (如: 需求→设计→编码→测试) |
| PLAN_AND_ACT | 先规划再执行 | 复杂项目分解 |

### 优势
- 消息驱动架构，Agent 间松耦合
- Environment 做消息路由，支持一对多广播
- 有完整的软件开发 SOP（产品经理→架构师→工程师→QA）
- 序列化/反序列化完善，支持断点续传

### 劣势
- asyncio 单线程，CPU 密集型任务会阻塞
- 消息路由复杂，调试困难
- Role/Action 继承层次深，扩展成本高
- 无真正的并行执行（单事件循环）

---

## 三、CrewAI 架构深度分析

### 核心模式：Crew 编排 + Agent 委派

```python
# CrewAI 的核心抽象
Agent (Pydantic)
  ├── role, goal, backstory      ← 人格定义
  ├── llm: BaseLLM               ← LLM 实例
  ├── tools: List[BaseTool]      ← 工具集
  ├── allow_delegation: bool     ← 能否委派给其他 Agent
  └── execute_task(task, context) ← ReAct 循环执行

Task (Pydantic)
  ├── description                ← 任务描述 (支持 {variable} 插值)
  ├── expected_output            ← 输出质量标准
  ├── agent: Agent               ← 指定执行者
  ├── context: List[Task]        ← 依赖任务
  ├── async_execution: bool      ← 异步执行标志
  └── output_json/pydantic       ← 结构化输出

Crew (编排器)
  ├── agents: List[Agent]
  ├── tasks: List[Task]
  ├── process: Sequential | Hierarchical
  └── kickoff(inputs={...})      ← 启动执行

Flow (高级 DAG)
  ├── @start()                   ← 入口节点
  ├── @listen(method)            ← 监听完成事件
  ├── @router(method)            ← 条件路由
  └── and_() / or_()             ← 并行门控
```

### 执行策略对比

| 策略 | 行为 | 并发能力 |
|------|------|---------|
| Sequential | 按顺序执行，output 作为下一个的 context | 无 |
| Hierarchical | Manager Agent 动态分配任务 | 有限 (Manager 决策串行) |
| Flow | DAG 编排，@listen 触发 | asyncio 并发 (单进程) |

### 委派机制

```python
# Agent 在 ReAct 循环中可以调用委派工具
delegate_work_to_coworker(coworker="Senior Engineer", task="优化数据库查询")
ask_question_to_coworker(coworker="Architect", question="应该用 Redis 还是 Memcached?")
# 委派是阻塞的 —— 等待目标 Agent 完成
```

### 优势
- API 极简，上手快
- Flow 抽象优雅（装饰器定义 DAG）
- 委派机制灵活（LLM 自主决定）
- 结构化输出支持好（JSON/Pydantic）

### 劣势
- 无真正并行（asyncio 单进程）
- 错误处理弱（无重试、无 fallback）
- 无流式支持
- Memory 锁定 ChromaDB
- Hierarchical 模式依赖 Manager LLM 决策质量

---

## 四、Eino 架构深度分析

### 核心模式：类型安全的 DAG 编排 + ADK

```go
// Eino 的核心抽象
Graph[I, O] (泛型 DAG)
  ├── AddLambdaNode(name, fn)    ← 任意函数节点
  ├── AddChatModelNode(name, m)  ← LLM 节点
  ├── AddToolsNode(name, t)      ← 工具节点
  ├── AddEdge(from, to)          ← 直连边
  ├── AddBranch(from, branch)    ← 条件分支
  └── Compile() → CompiledGraph  ← 编译验证

CompiledGraph (编译后的可执行图)
  ├── Run(ctx, input) → O       ← 同步执行
  ├── Stream(ctx, input) → StreamReader[O]  ← 流式执行
  └── 内部: 层级并行执行

Agent (ADK 接口)
  ├── Run(ctx, *AgentInput) → (*AgentOutput, error)
  └── ReactAgent (内置实现)
       ├── model: ChatModel
       ├── tools: ToolsNode
       └── maxSteps: int

Runner (编排器)
  ├── agent: Agent
  ├── maxSteps: int
  ├── checkpointer: CheckpointStore
  └── Run / RunWithInterrupt
```

### DAG 执行模型

```
Layer 0: [A]           ← 执行 A
           ↓
Layer 1: [B, C, D]     ← B/C/D 并行执行 (goroutine)
           ↓
Layer 2: [E]           ← 等待 B/C/D 全部完成，执行 E

每层内部: goroutine + WaitGroup 真并发
层间: 拓扑排序保证依赖顺序
```

### 并发模型

```go
// 层级并行 —— Go 原生优势
func (r *runner) executeLayer(ctx context.Context, nodes []string) {
    g, ctx := errgroup.WithContext(ctx)
    for _, node := range nodes {
        g.Go(func() error {
            return r.executeNode(ctx, node) // 每个节点一个 goroutine
        })
    }
    g.Wait() // 等待整层完成
}
```

### 优势
- Go 原生并发（goroutine 真并行，不是 asyncio 假并发）
- 类型安全（泛型 Graph[I,O]）
- 流式一等公民（StreamReader/Writer）
- Checkpoint/Interrupt 系统（人机协作）
- 字节跳动生产验证

### 劣势
- 无内置多 Agent 编排（需自建）
- 泛型复杂度高（动态图场景不友好）
- 社区较小
- 无消息总线（Agent 间通信需手动接线）

---

## 五、性能对比（关键指标）

```
┌──────────────────┬──────────┬──────────┬──────────┐
│ 指标             │ MetaGPT  │ CrewAI   │ Eino     │
├──────────────────┼──────────┼──────────┼──────────┤
│ 真并发           │ ✗ asyncio│ ✗ asyncio│ ✓ goroutine│
│ 5个Agent并行耗时 │ ~5x串行  │ ~5x串行  │ ~1x串行  │
│ 内存占用 (基础)  │ ~200MB   │ ~150MB   │ ~30MB    │
│ 启动速度         │ ~3s      │ ~2s      │ ~50ms    │
│ 流式输出         │ 有限     │ ✗        │ ✓ 一等   │
│ 取消/中断        │ 有限     │ ✗        │ ✓ context│
│ 错误恢复         │ 序列化恢复│ ✗       │ ✓ checkpoint│
│ 子进程管理       │ ✗        │ ✗        │ ✓ exec   │
└──────────────────┴──────────┴──────────┴──────────┘
```

---

## 六、MACDP 最终架构建议

### 方案：Eino DAG 引擎 + MetaGPT 消息模式 + CrewAI 委派模式

```
┌─────────────────────────────────────────────────────────┐
│                    MACDP Platform (Go)                    │
├─────────────────────────────────────────────────────────┤
│  CLI / Web / Bot (用户入口)                              │
├─────────────────────────────────────────────────────────┤
│  Task Planner (LLM 驱动的任务分解)                       │
│  ├── 输入: 用户自然语言需求                              │
│  └── 输出: 结构化 DAG (JSON)                             │
├─────────────────────────────────────────────────────────┤
│  Multi-Agent Orchestrator (MACDP 自建)                   │
│  ├── Role/Agent 定义 (借鉴 MetaGPT)                     │
│  │   └── role + goal + backstory + tools + adapter       │
│  ├── 消息总线 (借鉴 MetaGPT Environment)                │
│  │   └── publish/subscribe + routing                     │
│  ├── 委派机制 (借鉴 CrewAI)                             │
│  │   └── delegate_work_to_coworker (channel-based)       │
│  └── 审查流水线                                          │
│       └── self-review → cross-review → auto-test         │
├─────────────────────────────────────────────────────────┤
│  Eino Compose Layer (DAG 引擎)                          │
│  ├── Graph[TaskInput, TaskResult]                        │
│  ├── 层级并行执行 (goroutine)                            │
│  ├── 条件分支 (Branch)                                   │
│  └── 流式输出 (StreamReader)                             │
├─────────────────────────────────────────────────────────┤
│  Agent Adapters (外部 CLI 代理)                          │
│  ├── Hermes  (hermes chat -q)                           │
│  ├── Codex   (codex exec)                               │
│  ├── Claude  (claude -p)                                │
│  └── OpenCode (opencode run)                            │
├─────────────────────────────────────────────────────────┤
│  Git Manager (Worktree 隔离 + 合并)                     │
├─────────────────────────────────────────────────────────┤
│  State Store (SQLite / Redis)                           │
│  ├── 任务状态持久化                                      │
│  ├── Checkpoint (断点续传)                               │
│  └── 审查记录                                            │
└─────────────────────────────────────────────────────────┘
```

### 为什么不纯用 MetaGPT 或 CrewAI？

| 问题 | MetaGPT | CrewAI |
|------|---------|--------|
| 语言 | Python (慢) | Python (慢) |
| 并发 | asyncio 假并发 | asyncio 假并发 |
| 部署 | 需要 Python 环境 | 需要 Python 环境 |
| 外部 CLI 集成 | 不支持 | 不支持 |
| 适合 | LLM 原生 Agent | LLM 原生 Agent |

### 为什么用 Eino 做底层？

| 优势 | 说明 |
|------|------|
| 真并发 | goroutine 不是 asyncio，5 个 Agent 真并行 |
| 单二进制 | 无运行时依赖，部署简单 |
| DAG 引擎 | 成熟的拓扑排序+层级并行，不用手写 |
| 流式 | 实时推送 Agent 输出到前端 |
| Checkpoint | 内置断点续传，长任务可恢复 |
| 生产验证 | 字节跳动大规模使用 |

### 需要自建的部分

1. **Multi-Agent Orchestrator** — Eino 没有多 Agent 编排，需要自建
2. **消息总线** — 借鉴 MetaGPT 的 Environment 模式
3. **Role/Agent 定义** — 借鉴 MetaGPT 的 Role + CrewAI 的 Agent
4. **委派机制** — 借鉴 CrewAI 但用 channel 替代阻塞调用
5. **Agent Adapter 层** — 包装外部 CLI (Hermes/Codex/Claude)
6. **Task Planner** — LLM 驱动的任务分解
7. **Web API + Dashboard** — 实时可视化

---

## 七、结论

**最终推荐：Go + Eino DAG 引擎 + 自建多 Agent 编排层**

理由：
1. Eino 提供了成熟的 DAG 编排引擎（我们不用手写拓扑排序和并行执行）
2. Go 的 goroutine 是真正的并发，不是 Python asyncio 的协作式调度
3. 单二进制部署，用户无需安装 Python/Node 运行时
4. MetaGPT 和 CrewAI 的设计模式（消息总线、委派、角色）可以借鉴但不直接集成
5. Agent Adapter 层包装外部 CLI，保持灵活性
