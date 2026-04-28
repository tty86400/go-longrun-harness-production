# Codebase Overview (English + 简体中文)

## English

### What this project is

This repository is a **production-oriented long-running agent harness in Go**. It is designed for multi-step tasks with reliability and operations in mind rather than one-shot chat completion. Core goals include:

- strict action protocol,
- resumability,
- auditing,
- observability,
- provider abstraction,
- tool safety boundaries.

### General structure

- `cmd/harness/`: CLI entrypoint and runtime wiring.
- `internal/harness/`: core loop, protocol, state/session persistence.
- `internal/config/`: config loading, env expansion, defaults, validation.
- `internal/provider/`: model adapters (`openai_compatible`, `generic_json`, `mock`) + retry wrapper.
- `internal/tools/`: tools (`files`, `shell`, `git`, `http`, `benchmark`).
- `internal/observability/`: `/healthz`, `/debug/vars`, optional pprof.
- `internal/util/`: path/file safety primitives.
- `examples/`: sample configs and tasks.
- `docs/`: architecture and operations guides.

### Runtime flow to learn first

1. Read `cmd/harness/main.go`: parse flags, load config, create/resume session, build tools/providers, start observability server, run engine.
2. Read `internal/harness/engine.go`: step loop, timeouts/deadlines, actor decision, tool execution, reviewer/summarizer, persistence, run report.
3. Read `internal/harness/protocol.go`: strict JSON extraction/decoding and decision validation.
4. Read `internal/harness/session.go`: run directories, lock, state/checkpoint/transcript/prompt/artifact persistence.

### Important things newcomers should know

- **Durability is core design**: each run has a dedicated artifact directory (`state.json`, `checkpoints/`, `transcript.jsonl`, `prompts/`, `artifacts/`, `RUN_REPORT.md`).
- **Tool layer is risk boundary**: file operations are workspace-bounded; shell uses argv (not free shell strings) with allow/deny controls.
- **Provider abstraction keeps portability**: engine depends on provider interface instead of a single vendor SDK.
- **Configuration is policy**: loop limits, retries, memory cadence, and tool constraints come from config defaults + validation.

### What to learn next

Recommended order:

1. `internal/harness/engine.go`
2. `internal/harness/session.go`
3. `internal/provider/`
4. `internal/tools/`

Then:

- run mock example,
- inspect run artifacts,
- review deployment checklist before production rollout.

---

## 简体中文

### 这个项目是什么

这个仓库是一个 **Go 实现的生产导向长程 Agent Harness**。它不是一次性问答脚本，而是面向多步长任务的运行时系统。核心目标包括：

- 严格动作协议，
- 可恢复，
- 可审计，
- 可观测，
- Provider 抽象，
- 工具安全边界。

### 总体结构

- `cmd/harness/`：CLI 入口与组件装配。
- `internal/harness/`：主循环、协议、状态与会话持久化。
- `internal/config/`：配置加载、环境变量展开、默认值和校验。
- `internal/provider/`：模型适配器（`openai_compatible`、`generic_json`、`mock`）与重试封装。
- `internal/tools/`：工具层（文件、命令、Git、HTTP、benchmark）。
- `internal/observability/`：`/healthz`、`/debug/vars`、可选 pprof。
- `internal/util/`：路径与文件安全基础能力。
- `examples/`：示例配置与任务。
- `docs/`：架构与运维文档。

### 新人优先掌握的运行主线

1. 先读 `cmd/harness/main.go`：参数解析、配置加载、创建/恢复 session、构建 tools/providers、启动观测服务、进入 engine。
2. 再读 `internal/harness/engine.go`：step 循环、超时和截止控制、actor 决策、工具执行、review/summarize、持久化、run report。
3. 读 `internal/harness/protocol.go`：严格 JSON 提取/解码与决策校验。
4. 读 `internal/harness/session.go`：run 目录、锁机制、state/checkpoint/transcript/prompt/artifact 持久化。

### 新人必须知道的重点

- **可恢复能力是核心设计**：每次 run 都有独立产物目录（`state.json`、`checkpoints/`、`transcript.jsonl`、`prompts/`、`artifacts/`、`RUN_REPORT.md`）。
- **工具层是主要风险边界**：文件操作严格限制在 workspace 内；shell 采用 argv（非自由 shell 字符串），并支持 allow/deny 控制。
- **Provider 抽象保证可迁移性**：engine 依赖统一 provider 接口，不绑定单一厂商 SDK。
- **配置就是运行策略**：步数/超时、重试、记忆压缩节奏、工具限制由配置默认值与校验共同定义。

### 下一步学习建议

推荐阅读顺序：

1. `internal/harness/engine.go`
2. `internal/harness/session.go`
3. `internal/provider/`
4. `internal/tools/`

然后：

- 跑一遍 mock 示例，
- 对照产物目录理解完整闭环，
- 上线前逐项核对部署清单。
