# 05. Operations and Learning Guide

## 最快掌握这个 harness 的阅读顺序

### 第一步：看 `engine.go`

你要先回答一个问题：

> 一轮 step 从开始到结束，到底发生了哪些确定动作？

建议你按函数顺序读：

- `Run()`
- `askActor()`
- `askReviewer()`
- `summarize()`
- `executeAction()`
- `persistStep()`

看懂之后，你就已经掌握了 70% 的框架设计。

### 第二步：看 `session.go`

这里解决的是“为什么这不是聊天脚本”：

- 为什么有 `state.json`
- 为什么有 `checkpoints/`
- 为什么有 `prompts/`
- 为什么需要 run lock

### 第三步：看 `provider/`

这里解决的是“为什么这个 harness 不绑死某个模型供应商”。

### 第四步：看 `tools/`

这里解决的是“为什么模型真的可以做工程工作，以及为什么这件事又很危险”。

## 一次真实运行应该关注哪些文件

假设你跑完一个任务，先看这几个：

1. `RUN_REPORT.md`
2. `state.json`
3. `transcript.jsonl`
4. `prompts/step-XXXX-actor.json`
5. `artifacts/step-XXXX-action-YY-*.json`

它们分别回答的问题是：

- 这次 run 最终怎么样了？
- 系统当前“官方状态”是什么？
- 每一步到底发生了什么？
- 模型那一步到底看到了什么、说了什么？
- 工具调用的结构化结果是什么？

## 一次端到端 mock 运行会发生什么

`mock` provider 被设计成一个最小可验证闭环：

- step 1：写 `demo.txt`
- step 2：读 `demo.txt`
- step 3：返回完成

这不是为了“炫技”，而是为了把控制回路本身跑通：

- actor JSON 是否能被严格解析
- file tool 是否能写和读
- transcript / checkpoint / report 是否都能生成
- integration test 是否能稳定通过

## 运行参数调优建议

### max_actions_per_step

建议从 2 或 3 开始。

太大时会出现：

- 单步副作用过重
- 失败后不易定位
- 恢复边界变粗

### review.every_n_steps

建议从 3 到 5 开始。

太稀疏时，模型更容易 drift；太频繁时，token 成本会上升。

### summarize_every_n_steps

如果任务确实会跑很久，建议 6 到 10；
如果任务短而复杂，可以更小一些。

## 生产运维建议

- 给 run 目录做周期归档。
- 对 `/debug/vars` 接系统指标采集。
- 对失败状态做告警。
- 对 provider 的 429 / 5xx 做额外监控。
- 对平均 step 时长、tool failure rate、summary 频率做仪表盘。

## 学习练习

### 练习 1：加一个 browser-like fetch 工具

目标：在 `http.fetch` 基础上补一个只允许 GET 特定文档域名的只读工具。

### 练习 2：给 `generic_json` 增加 prompt 模板变量

比如支持：

- `{{messages}}`
- `{{json_hint}}`
- `{{task}}`
- `{{summary}}`

### 练习 3：把 step 结果发到 Kafka / DB

让 transcript 除了写本地 JSONL，还可以被集中收集。

这三类练习能帮助你真正掌握这个 harness 的扩展方式。
