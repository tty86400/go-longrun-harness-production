# 07. Deployment Checklist

下面这份清单不是代码实现细节，而是你把这套 harness 真正放进环境时该核对的事项。

## 隔离与权限

- harness 进程是否运行在单独容器或沙箱里。
- workspace 是否挂在独立目录，而不是宿主关键目录。
- shell / git / benchmark 工具是否按环境做了最小权限约束。
- provider API key 是否通过环境变量或 secrets 系统注入。

## 可靠性

- `run.runs_dir` 是否落在持久卷上。
- 失败后是否有人或系统会触发 `-resume-run`。
- 是否有对 run 状态的监控与告警。
- `max_steps`、`max_wall_clock_minutes` 是否有业务级默认值。

## 观测与审计

- 是否采集 `run.log`。
- 是否采集 `/debug/vars`。
- 是否需要保留 raw provider payload。
- 是否需要对 transcript 与 prompt 审计做脱敏。

## 成本与资源

- 是否给 provider 请求做了额度限制。
- 是否对 shell / benchmark 运行时间做了上限。
- 是否限制了 workspace 大小。
- 是否定义了 run 归档与清理策略。

## 回归测试

每次改 harness 主循环后，至少应重跑：

```bash
go test ./...
go run ./cmd/harness -config examples/config.mock.json -task examples/tasks/mock_demo.md
```

这样可以同时覆盖：

- 单元测试
- integration test
- 真实 CLI 启动路径
- prompt / checkpoint / report 落盘路径
