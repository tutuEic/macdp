# Multi-Agent Collaborative Development Platform (MACDP)

## 一、项目愿景

构建一个类似 Codex 的多 Agent 协同开发平台，能统一调度 Hermes、OpenClaw、Codex、Claude Code、OpenCode 等多个 AI 编码代理，形成一个"AI 开发团队"，支持任务分解、并行执行、代码审查、冲突解决等完整开发流程。

---

## 二、需求分析

### 2.1 核心需求

| 需求 | 描述 | 优先级 |
|------|------|--------|
| 统一调度 | 一个入口调度多个 Agent (Hermes/Codex/Claude Code/OpenCode) | P0 |
| 任务分解 | 自动将大任务拆分为可并行的子任务 | P0 |
| Git Worktree 隔离 | 每个 Agent 独立 worktree，避免文件冲突 | P0 |
| 实时监控 | 查看每个 Agent 的进度、日志、输出 | P0 |
| 代码审查 | Agent 之间交叉审查代码 | P1 |
| 冲突解决 | 多 Agent 修改同一仓库时自动解决 git 冲突 | P1 |
| 审批门控 | 关键操作需要用户确认 (approval gate) | P1 |
| 结果汇总 | 所有 Agent 完成后自动汇总、合并、测试 | P1 |
| 模型自由选择 | 每个任务/Agent 可选不同模型 (GPT-4o/Claude/DeepSeek等) | P2 |
| 成本控制 | 预算上限、token 追踪、成本报告 | P2 |
| 持久化会话 | 支持暂停/恢复整个开发流程 | P2 |
| 多渠道触发 | 从 Telegram/Discord/CLI/Web 触发开发任务 | P2 |

### 2.2 角色模型

```
用户 (Human)
  │
  ▼
┌─────────────────────────────────────────────┐
│           Orchestrator (编排器)              │
│  - 理解用户意图                              │
│  - 分解任务为子任务                          │
│  - 分配 Agent 角色                           │
│  - 监控进度、处理异常                        │
│  - 汇总结果                                  │
└──────────┬──────────┬──────────┬────────────┘
           │          │          │
     ┌─────▼──┐ ┌─────▼──┐ ┌─────▼──┐
     │Worker A│ │Worker B│ │Worker C│  ... (可扩展)
     │(Hermes)│ │(Codex) │ │(Claude)│
     └────────┘ └────────┘ └────────┘
```

### 2.3 Agent 角色分配

| 角色 | 推荐 Agent | 原因 |
|------|-----------|------|
| **Orchestrator** (编排器) | Hermes | 有 delegate_task、kanban、profile 系统 |
| **Backend Dev** (后端开发) | Claude Code | print mode 结构化输出、--max-turns 控制 |
| **Frontend Dev** (前端开发) | Claude Code / Codex | 强代码生成能力 |
| **Reviewer** (代码审查) | Hermes / Claude Code | 可读 diff、结构化分析 |
| **Tester** (测试) | Hermes | 可执行 shell、管理进程 |
| **Doc Writer** (文档) | OpenCode / Hermes | 通用能力即可 |

---

## 三、系统架构

### 3.1 整体架构图

```
                         ┌──────────────────────────────────┐
                         │        User Interface Layer       │
                         │  ┌─────┐ ┌──────┐ ┌──────┐       │
                         │  │ CLI │ │ Web  │ │ Bot  │       │
                         │  │(TUI)│ │(React)│ │(TG/DC)│     │
                         │  └──┬──┘ └──┬───┘ └──┬───┘       │
                         └─────┼───────┼────────┼────────────┘
                               │       │        │
                         ┌─────▼───────▼────────▼────────────┐
                         │        Gateway Layer               │
                         │  ┌──────────────────────┐          │
                         │  │ OpenClaw Gateway      │          │
                         │  │ (multi-channel)       │          │
                         │  └──────────┬───────────┘          │
                         │             │                      │
                         │  ┌──────────▼───────────┐          │
                         │  │ Hermes Gateway        │          │
                         │  │ (profiles + skills)   │          │
                         │  └──────────┬───────────┘          │
                         └─────────────┼──────────────────────┘
                                       │
                         ┌─────────────▼──────────────────────┐
                         │     Orchestrator Core (编排核心)     │
                         │                                     │
                         │  ┌───────────┐  ┌──────────────┐   │
                         │  │ Task      │  │ Agent         │   │
                         │  │ Planner   │  │ Registry      │   │
                         │  │ (LLM)     │  │ (capabilities)│   │
                         │  └─────┬─────┘  └──────┬───────┘   │
                         │        │               │           │
                         │  ┌─────▼───────────────▼───────┐   │
                         │  │   Task Scheduler / Router     │   │
                         │  │   (dependency graph + queue)  │   │
                         │  └─────┬───────────────────────┘   │
                         │        │                           │
                         │  ┌─────▼───────────────────────┐   │
                         │  │   Git Manager               │   │
                         │  │   (worktree + merge + conflict)│ │
                         │  └─────┬───────────────────────┘   │
                         │        │                           │
                         │  ┌─────▼───────────────────────┐   │
                         │  │   Review Pipeline            │   │
                         │  │   (cross-review + lint + test)│  │
                         │  └──────────────────────────────┘  │
                         └─────────────┬──────────────────────┘
                                       │
                    ┌──────────────────┼──────────────────┐
                    │                  │                   │
          ┌─────────▼────┐  ┌─────────▼────┐  ┌──────────▼───┐
          │  Worker Pool  │  │  Worker Pool  │  │  Worker Pool │
          │               │  │               │  │              │
          │ ┌───────────┐ │  │ ┌───────────┐ │  │ ┌──────────┐│
          │ │ Hermes    │ │  │ │ Codex CLI │ │  │ │ Claude   ││
          │ │ Subagent  │ │  │ │           │ │  │ │ Code CLI ││
          │ └───────────┘ │  │ └───────────┘ │  │ └──────────┘│
          │ ┌───────────┐ │  │               │  │ ┌──────────┐│
          │ │ OpenCode  │ │  │               │  │ │ Custom   ││
          │ │ CLI       │ │  │               │  │ │ Agent    ││
          │ └───────────┘ │  │               │  │ └──────────┘│
          └───────────────┘  └───────────────┘  └─────────────┘
```

### 3.2 核心组件

#### 3.2.1 Task Planner (任务规划器)

```
输入: 用户的高层需求 (自然语言)
输出: 结构化的任务依赖图 (DAG)

示例:
  用户: "给项目添加用户认证系统，包括注册、登录、JWT、中间件"

  Task Planner 输出:
  ┌─ T1: 设计数据库 schema (users table)          ─┐
  ├─ T2: 实现注册 API (POST /auth/register)        ─┤── 可并行
  ├─ T3: 实现登录 API (POST /auth/login)           ─┤
  ├─ T4: JWT token 生成和验证工具                   ─┤
  ├─ T5: 认证中间件                                ─┘ (依赖 T4)
  ├─ T6: 前端登录/注册页面                          (依赖 T2, T3)
  ├─ T7: 集成测试                                  (依赖 T2, T3, T5)
  └─ T8: API 文档更新                               (依赖 T2, T3)

  依赖图:
  T1 ──┬──> T2 ──┬──> T6
       │         │
       ├──> T3 ──┤──> T7
       │         │
       └──> T4 ──┴──> T5 ──> T7
                    T8 (独立)
```

#### 3.2.2 Agent Registry (Agent 注册表)

```yaml
agents:
  hermes:
    type: orchestrator | worker
    capabilities: [shell, file_io, web, delegation, kanban]
    strengths: [orchestration, testing, debugging, general_tasks]
    max_concurrent: 3
    entrypoint: "hermes chat -q"
    cost_per_1k_tokens: varies  # 取决于配置的模型

  codex:
    type: worker
    capabilities: [code_gen, shell, file_io]
    strengths: [fast_prototyping, single_file_tasks]
    max_concurrent: 5
    entrypoint: "codex exec"
    flags: ["--full-auto"]

  claude-code:
    type: worker
    capabilities: [code_gen, shell, file_io, review, subagents]
    strengths: [complex_refactoring, code_review, multi_step]
    max_concurrent: 3
    entrypoint: "claude -p"
    flags: ["--max-turns 15", "--output-format json"]

  opencode:
    type: worker
    capabilities: [code_gen, shell, file_io]
    strengths: [provider_agnostic, session_resume]
    max_concurrent: 3
    entrypoint: "opencode run"
```

#### 3.2.3 Task Scheduler (任务调度器)

```
核心职责:
  1. 维护任务 DAG (有向无环图)
  2. 计算可执行任务 (入度为 0 的节点)
  3. 根据 Agent 能力匹配任务
  4. 管理 worktree 分配
  5. 处理失败重试

调度算法:
  while (有未完成任务):
    ready = get_tasks_with_all_deps_met()
    for task in ready:
      agent = best_agent_for(task)  # 匹配能力、负载、成本
      if agent.available:
        dispatch(task, agent, worktree=allocate_worktree(task))
      else:
        wait_for_agent()
```

#### 3.2.4 Git Manager (Git 管理器)

```
职责:
  - 创建/管理 git worktrees (每个任务一个隔离目录)
  - 任务完成后: commit → push branch → 创建 PR
  - 冲突检测与自动合并
  - 最终集成: merge all branches → test → merge to main

Worktree 策略:
  project/
  ├── .git/
  ├── main/              (主分支, 用户工作目录)
  ├── .macdp/
  │   ├── worktrees/
  │   │   ├── T1-schema/        (Task 1 的隔离目录)
  │   │   ├── T2-register-api/  (Task 2)
  │   │   ├── T3-login-api/     (Task 3)
  │   │   └── ...
  │   ├── sessions/             (会话记录)
  │   └── reports/              (执行报告)
```

#### 3.2.5 Review Pipeline (审查流水线)

```
每个任务完成后:
  1. Agent 自审 (self-review)
  2. 交叉审查 (cross-review by another agent)
  3. 自动化检查 (lint, type-check, test)
  4. 用户审批 (可选, approval gate)

审查失败 → 回退到原 Agent 修复 → 重新审查 (最多 3 轮)
```

### 3.3 通信协议

```
Orchestrator 与 Worker 之间通过 JSON 消息通信:

// 任务分发
{
  "type": "task_assign",
  "task_id": "T2",
  "title": "Implement registration API",
  "description": "Create POST /auth/register endpoint...",
  "context": {
    "schema": "...(T1的输出)...",
    "conventions": "FastAPI + SQLAlchemy + Pydantic",
    "worktree": "/project/.macdp/worktrees/T2-register-api"
  },
  "constraints": {
    "max_turns": 15,
    "timeout_seconds": 300,
    "allowed_tools": ["Read", "Edit", "Write", "Bash"]
  }
}

// 任务完成
{
  "type": "task_complete",
  "task_id": "T2",
  "status": "success",
  "summary": "Created register endpoint with validation...",
  "files_changed": ["src/auth/register.py", "tests/test_register.py"],
  "git_branch": "feature/T2-register-api",
  "cost_usd": 0.15,
  "duration_seconds": 45
}

// 任务失败
{
  "type": "task_failed",
  "task_id": "T2",
  "error": "Max turns exceeded",
  "partial_output": "...",
  "retry_suggested": true
}
```

---

## 四、技术选型

### 4.1 核心技术栈

| 层 | 技术 | 理由 |
|----|------|------|
| **Orchestrator** | Python + Hermes Agent | 已有 delegate_task/kanban/profile 系统 |
| **Worker 调度** | Python asyncio | 并发管理多个 Agent 进程 |
| **通信** | JSON over stdin/stdout + Unix socket | 兼容所有 CLI Agent |
| **任务持久化** | SQLite (复用 Hermes state.db) | 轻量、已有基础设施 |
| **前端** | React + WebSocket | 实时进度推送 |
| **Git 操作** | GitPython + subprocess | worktree/merge/rebase |
| **配置** | YAML (复用 Hermes config.yaml 格式) | 一致性 |

### 4.2 Agent 集成方式

```python
# 每个 Agent 通过 Adapter 模式接入

class AgentAdapter(ABC):
    @abstractmethod
    async def execute(self, task: Task) -> TaskResult:
        """执行一个任务"""
        pass

    @abstractmethod
    async def monitor(self, session_id: str) -> AgentStatus:
        """监控执行状态"""
        pass

    @abstractmethod
    async def cancel(self, session_id: str):
        """取消执行"""
        pass

class HermesAdapter(AgentAdapter):
    async def execute(self, task):
        # 方式1: delegate_task (轻量子任务)
        # 方式2: tmux + hermes -q (重量级任务)
        # 方式3: hermes -w (worktree 隔离)
        pass

class CodexAdapter(AgentAdapter):
    async def execute(self, task):
        # codex exec "task description" --full-auto
        pass

class ClaudeCodeAdapter(AgentAdapter):
    async def execute(self, task):
        # claude -p "task" --output-format json --max-turns 15
        pass

class OpenCodeAdapter(AgentAdapter):
    async def execute(self, task):
        # opencode run "task description"
        pass
```

---

## 五、工作流示例

### 5.1 完整开发流程

```
用户: "用 FastAPI + React 构建一个带用户认证的 Todo 应用"

Step 1: Orchestrator 分析需求
  → 生成任务 DAG (10 个任务, 4 个并行组)

Step 2: 第一波并行 (基础层)
  ├─ [Hermes] T1: 创建项目结构 + 数据库 schema
  ├─ [Claude] T2: FastAPI 基础配置 + 路由框架
  └─ [Codex]  T3: React 项目初始化 + 路由配置

Step 3: 第二波并行 (核心功能, 依赖 T1)
  ├─ [Claude] T4: 用户注册/登录 API
  ├─ [Claude] T5: JWT 认证中间件
  └─ [Codex]  T6: Todo CRUD API

Step 4: 第三波并行 (前端, 依赖 T4, T6)
  ├─ [Claude] T7: 登录/注册页面
  └─ [Codex]  T8: Todo 列表页面

Step 5: 交叉审查
  ├─ [Hermes] 审查 T4, T5, T6 的后端代码
  └─ [Hermes] 审查 T7, T8 的前端代码

Step 6: 集成
  ├─ [Hermes] 合并所有 worktree 分支
  ├─ [Hermes] 运行集成测试
  └─ [Hermes] 生成最终报告

Step 7: 用户审批
  → 展示变更摘要, 等待用户确认
  → merge to main
```

### 5.2 代码审查流程

```
用户: "审查 PR #42"

Step 1: [Hermes] 获取 PR diff
Step 2: 并行审查
  ├─ [Claude] 安全审查 (SQL injection, XSS, auth flaws)
  ├─ [Codex]  性能审查 (N+1 queries, memory leaks)
  └─ [Hermes] 代码风格 + 测试覆盖
Step 3: [Hermes] 汇总审查报告
Step 4: 用户决定是否自动修复
Step 5: [Claude] 修复问题 → 重新审查
```

---

## 六、数据模型

### 6.1 核心表结构

```sql
-- 项目
CREATE TABLE projects (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    repo_path TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 任务
CREATE TABLE tasks (
    id TEXT PRIMARY KEY,
    project_id TEXT REFERENCES projects(id),
    parent_task_id TEXT,          -- NULL = 顶层任务
    title TEXT NOT NULL,
    description TEXT,
    status TEXT DEFAULT 'pending', -- pending/assigned/running/review/failed/done
    assigned_agent TEXT,           -- hermes/codex/claude-code/opencode
    worktree_path TEXT,
    git_branch TEXT,
    result_json TEXT,              -- TaskResult JSON
    cost_usd REAL DEFAULT 0,
    created_at TIMESTAMP,
    started_at TIMESTAMP,
    completed_at TIMESTAMP
);

-- 任务依赖
CREATE TABLE task_deps (
    task_id TEXT REFERENCES tasks(id),
    depends_on TEXT REFERENCES tasks(id),
    PRIMARY KEY (task_id, depends_on)
);

-- Agent 执行日志
CREATE TABLE agent_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id TEXT REFERENCES tasks(id),
    agent TEXT NOT NULL,
    session_id TEXT,
    log_type TEXT,  -- stdout/stderr/tool_call/result
    content TEXT,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 审查记录
CREATE TABLE reviews (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id TEXT REFERENCES tasks(id),
    reviewer_agent TEXT,
    verdict TEXT,  -- approve/request_changes/needs_discussion
    comments_json TEXT,
    created_at TIMESTAMP
);
```

---

## 七、配置设计

```yaml
# ~/.hermes/macdp.yaml (或集成到 config.yaml)

macdp:
  enabled: true

  # 默认编排器
  orchestrator: hermes

  # Agent 配置
  agents:
    hermes:
      enabled: true
      role: orchestrator
      max_concurrent: 3

    codex:
      enabled: true
      role: worker
      entrypoint: codex
      flags: ["--full-auto"]
      max_concurrent: 5
      strengths: [prototyping, single_file, scripts]

    claude-code:
      enabled: true
      role: worker
      entrypoint: claude
      flags: ["--output-format json", "--max-turns 15"]
      max_concurrent: 3
      strengths: [complex_code, refactoring, review]

    opencode:
      enabled: false  # 可选
      role: worker
      entrypoint: opencode
      max_concurrent: 3

  # 任务调度
  scheduler:
    max_parallel_tasks: 5
    task_timeout_seconds: 600
    retry_on_failure: true
    max_retries: 2

  # Git 管理
  git:
    worktree_dir: ".macdp/worktrees"
    auto_commit: true
    branch_prefix: "macdp/"
    merge_strategy: "squash"  # squash/merge/rebase

  # 审查
  review:
    enabled: true
    cross_review: true        # 交叉审查
    auto_lint: true
    auto_test: true
    approval_required: false  # 是否需要用户审批
    max_review_rounds: 3

  # 成本控制
  budget:
    max_per_task_usd: 5.0
    max_per_project_usd: 50.0
    alert_threshold_usd: 30.0
```

---

## 八、UI 设计 (Web Dashboard)

### 8.1 页面结构

```
┌─────────────────────────────────────────────────────────┐
│  MACDP Dashboard                          [sci-fi theme]│
├─────────────────────────────────────────────────────────┤
│                                                         │
│  ┌─ Project: my-todo-app ─────────────────────────────┐ │
│  │                                                     │ │
│  │  📊 Overview                                        │ │
│  │  ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐              │ │
│  │  │ 10   │ │ 3    │ │ 2    │ │ $2.5 │              │ │
│  │  │Tasks │ │Active│ │Done  │ │Cost  │              │ │
│  │  └──────┘ └──────┘ └──────┘ └──────┘              │ │
│  │                                                     │ │
│  │  📋 Task DAG (可视化依赖图)                         │ │
│  │  ┌───┐    ┌───┐    ┌───┐                           │ │
│  │  │T1 │───>│T4 │───>│T7 │                           │ │
│  │  └───┘    └───┘    └───┘                           │ │
│  │                                                     │ │
│  │  🤖 Active Agents                                  │ │
│  │  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐  │ │
│  │  │ 🟢 Hermes   │ │ 🟢 Claude   │ │ 🟡 Codex    │  │ │
│  │  │ T5:running  │ │ T4:running  │ │ T6:queued   │  │ │
│  │  │ ████░░ 60%  │ │ ██████ 85%  │ │ ░░░░░░ 0%   │  │ │
│  │  └─────────────┘ └─────────────┘ └─────────────┘  │ │
│  │                                                     │ │
│  │  📝 Live Logs                                      │ │
│  │  ┌─────────────────────────────────────────────┐   │ │
│  │  │ [Claude] Reading src/auth/models.py...      │   │ │
│  │  │ [Hermes] Running pytest tests/test_auth.py  │   │ │
│  │  │ [Codex] Creating src/components/Login.tsx   │   │ │
│  │  └─────────────────────────────────────────────┘   │ │
│  └─────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────┘
```

---

## 九、实现路线图

### Phase 1: MVP (最小可行产品) — 2 周
- [ ] 任务分解器 (LLM 驱动, 输出 JSON DAG)
- [ ] Agent Adapter: Hermes + Claude Code (print mode)
- [ ] Git worktree 管理 (创建/合并/清理)
- [ ] 串行 + 简单并行调度
- [ ] CLI 界面 (Hermes slash command: /macdp)
- [ ] 基本结果汇总

### Phase 2: 增强 — 2 周
- [ ] Agent Adapter: Codex + OpenCode
- [ ] 交叉代码审查
- [ ] 失败重试 + 错误处理
- [ ] 成本追踪
- [ ] 审批门控 (approval gate)

### Phase 3: Web UI — 2 周
- [ ] React Dashboard (任务 DAG 可视化)
- [ ] WebSocket 实时日志
- [ ] Agent 状态面板
- [ ] 科幻风 UI (霓虹 + 暗色主题)

### Phase 4: 高级功能 — 2 周
- [ ] OpenClaw Lobster 集成 (确定性工作流)
- [ ] Hermes Kanban 集成 (多 Agent 任务看板)
- [ ] 持久化会话 (暂停/恢复)
- [ ] 学习系统 (记录成功模式, 优化未来调度)

---

## 十、风险与挑战

| 风险 | 影响 | 缓解措施 |
|------|------|---------|
| Agent 输出质量不可控 | 低质量代码需要大量审查 | 交叉审查 + 自动测试门控 |
| Git 冲突频繁 | 并行修改同文件时冲突 | 细粒度任务分解 + 文件级锁定 |
| 成本爆炸 | 多 Agent 并行消耗大量 token | 预算上限 + 任务级成本追踪 |
| Agent CLI 不稳定 | 不同版本行为不一致 | Adapter 层抽象 + 版本检测 |
| 上下文丢失 | Agent 之间缺乏共享上下文 | 每个任务携带足够 context |
| 长任务超时 | 复杂任务超出 timeout | 分阶段执行 + 断点续传 |

---

## 十一、与现有系统的集成点

### Hermes 集成
- 使用 `delegate_task` 做轻量子任务
- 使用 `profile` 隔离不同 Agent 的配置
- 使用 `kanban` 做多 Agent 任务队列
- 使用 `cronjob` 做定时任务
- 使用 `skill` 系统存储最佳实践

### OpenClaw 集成
- 通过 OpenClaw Gateway 实现多渠道触发
- 使用 Lobster DSL 定义确定性审查流程
- 通过 OpenClaw 的 agent 系统管理 Agent 配置

### 外部 Agent 集成
- Codex: `codex exec` (one-shot) + PTY (interactive)
- Claude Code: `claude -p` (print mode, 首选) + tmux (interactive)
- OpenCode: `opencode run` (one-shot) + PTY (interactive)
- 可扩展: 任何 CLI Agent 通过 Adapter 接入
