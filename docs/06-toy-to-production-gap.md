# 06. From Toy to Production: Gap Analysis

这份文档直接回答你最关心的问题：

> 为什么前一版像 toy，而这一版更像生产级 harness？

## 1. 以前的问题：缺“运行时”

toy 版常见特征是：

- 只有一个 loop，没有清晰状态机。
- 没有 checkpoint / resume。
- 没有 run lock。
- 没有审计目录。
- 没有 metrics / health。
- tool 只是“能调”，不是“受控能调”。
- provider 只是“能连上某个 API”，不是稳定抽象。

这样代码看起来能跑，但并不能支撑真的长时间工程任务。

## 2. 这一版补上的东西

### 状态持久化

补了：

- `state.json`
- `checkpoints/`
- `transcript.jsonl`
- `RUN_REPORT.md`

### 可恢复能力

补了：

- `-resume-run`
- step boundary checkpoint
- `.lock`

### provider 工程化

补了：

- `openai_compatible`
- `generic_json`
- `mock`
- retry / backoff / jitter
- strict JSON extraction

### 工具安全边界

补了：

- workspace path jail
- argv 执行
- allowlist / denylist
- benchmark 预注册脚本

### 运维可观测性

补了：

- structured logging
- `/healthz`
- `/debug/vars`
- 可选 pprof

### 测试

补了：

- provider JSON extraction test
- path safety test
- config env expansion test
- harness integration test

## 3. 仍然需要你按自己环境再补的部分

即便如此，这份代码也不是“无需思考直接扔公司生产”的万能成品。

你仍应按自己的基础设施补这些：

- 真正的隔离执行环境
- secrets 管理
- 成本预算与限流
- 分布式任务调度
- 多 worker 协调
- 集中日志与告警
- 组织级访问控制

## 4. 更准确的说法

如果你严格区分术语，那么这份工程更准确的表述是：

- **不是玩具**
- **是 production-oriented baseline**
- **已经具备生产级 harness 的关键工程骨架**
- **离企业落地只差环境集成，而不是差主框架本身**

这也是我认为对你最有价值的交付方式：

不是给你一个“演示脚本”，而是给你一套 **你可以继续接到真实模型和真实工具链上的主干**。
