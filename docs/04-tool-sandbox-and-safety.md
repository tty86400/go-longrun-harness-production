# 04. Tool Sandbox and Safety

## 工具层是最大风险面

LLM 本身只会“说话”；一旦加了 tools，它就可以：

- 改文件
- 跑命令
- 读 Git 状态
- 拉远程资源
- 跑 benchmark

这时 harness 的安全性，主要取决于 tool sandbox，而不是 prompt 写得多漂亮。

## 文件工具的边界

文件工具统一通过 `ResolveWithinBase()` 把路径限制在 workspace 内。

这解决的是：

- `../` 逃逸
- 绝对路径写系统目录
- 把 run 状态写到不受控位置

相关代码：`internal/util/paths.go`

## 为什么 shell.exec 用 argv 而不是 command string

这是一个非常关键的生产级选择。

如果 tool 接口长这样：

```json
{"command":"go test ./... && rm -rf /"}
```

那你几乎等于把 shell parser 的全部复杂性都暴露给模型了。

本工程改成：

```json
{"argv":["go","test","./..."]}
```

这样有几个直接好处：

- 更容易做 allowlist / denylist
- 不经过 shell 展开
- 参数边界清晰
- 更容易审计与重放

## Shell allowlist / denylist

`shell.exec` 支持：

- `allowed_commands`
- `denied_commands`
- `env_allowlist`
- `default_timeout_seconds`
- `max_output_bytes`

这意味着你可以按环境收紧：

- 开发环境允许 `go`, `git`, `make`
- 生产环境完全禁止 `docker`, `kubectl`, `sudo`
- 只给最小环境变量

## Benchmark tool 的设计思路

benchmark 不是让模型自由发命令，而是让模型只能选择 **预配置脚本名字**。

这样模型只能说：

```json
{"tool":"benchmark.run","args":{"name":"unit-tests"}}
```

真正执行哪条命令，由 operator 在 config 里决定。

这是一种很典型的生产设计：

- 模型负责选择策略
- 系统负责约束能力边界

## 还应该补哪些外部隔离

即使已经有这些内建限制，我依然建议真正上线时再补外层隔离：

- 容器隔离
- seccomp / AppArmor / SELinux
- 只读根文件系统
- 独立临时目录
- 网络策略
- 专用 service account

原因很简单：

进程内检查是第一道防线，不是最后一道。

## 安全审计时你应该重点看什么

- shell allowlist 是否过宽
- HTTP 允许域名是否过宽
- benchmark 脚本是否可被模型间接注入
- workspace 是否与宿主关键目录共用
- secrets 是否被暴露给命令执行环境
- prompt / raw provider payload 是否包含敏感数据
