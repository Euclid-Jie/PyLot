# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目简介

PyLot — Python 脚本调度桌面管理系统（Windows 专用）。技术栈：Go + Wails v2 + Vue3 + SQLite。

Go 模块名：`script-manager`（go.mod 历史遗留，勿改）。

## 常用命令

```bash
# 开发模式（热重载）
wails dev

# 打包（无控制台黑框）
wails build -platform windows/amd64 -ldflags "-H windowsgui"
& "$env:USERPROFILE\go\bin\wails.exe" build -platform windows/amd64 -ldflags "-H windowsgui"
# 输出：build/bin/PyLot.exe

# 仅编译 Go（验证语法）
go build ./...
```

## 新机器环境搭建

```powershell
# 1. 安装工具链（重开终端使 PATH 生效）
winget install GoLang.Go
winget install OpenJS.NodeJS.LTS

# 2. 配置 Go 代理（国内必须）
go env -w GOPROXY=https://goproxy.cn,direct
go env -w GONOSUMDB=*

# 3. 安装 Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# 4. 安装依赖
go mod download
cd frontend && npm install && cd ..
```

不需要 GCC/MinGW（用纯 Go SQLite 驱动 `modernc.org/sqlite`）。WebView2 在 Win10/11 系统自带。

## 架构

### 后端（Go）

- `main.go` — Wails 启动 + systray 托盘（`goruntime.LockOSThread()` 保证消息泵稳定）。Wails `SingleInstanceLock` 防多开并在二次启动时显示已有窗口；双托盘图标（`icon_free.ico` / `icon_busy.ico`）随运行状态切换；关闭窗口触发 `OnBeforeClose` 隐藏到托盘；托盘右键菜单：显示主窗口 / 定时任务 / 退出（`os.Exit(0)`）。
- `app.go` — 所有暴露给前端的方法（Wails bind）。包含文件对话框、VS Code 打开工作目录、脚本推断、窗口大小读写、Workflow CRUD、Service CRUD/控制/日志等。
- `internal/db/` — SQLite 初始化建表（11张表）+ WAL 模式（防并发写丢失）+ `busy_timeout=5000`；`write.go` 对脚本日志、运行状态等核心写入做进程内串行化和 SQLite busy/locked 短重试；`workflow_run_nodes` 保存工作流批次的节点快照、状态、耗时及对应 `run_records`；`script_lists` 保存用户脚本列表，`scripts.list_id` 为空表示未分类；`schema_migrations` 保证旧 category 到动态列表的迁移只执行一次；`global_config` 含 `lark_cli_path`/`lark_open_id` 字段；`services` 表保存服务命令、工作目录、跟随 PyLot 启动开关、访问端口和协议；日志清理（7天）。数据库文件在 exe 同目录。
- `internal/scriptlist/` — 脚本列表 CRUD。列表名称唯一；删除列表时只将脚本移入“未分类”，不会删除脚本。
- `internal/commandline/` — Windows 命令行解析工具，封装 `windows.DecomposeCommandLine`，供脚本 custom 模式、固定参数解析和服务命令复用；相对可执行文件优先按 WorkDir 解析。
- `internal/script/runner.go` — 进程启动，注入 `PYTHONIOENCODING=utf-8` 统一编码，`SysProcAttr{HideWindow: true}` 隐藏黑框，支持卡死超时检测；module 模式自动将绝对路径转换为相对 WorkDir 的点号模块名；custom 模式直接解析并执行用户填写的完整命令。
- `internal/service/manager.go` — 长期运行服务进程管理。维护 `starting/running/stopping/exited/failed/stopped` 运行态、PID、启动/停止时间、退出码、最近错误和本次会话最近 1000 行日志；使用 Windows `DecomposeCommandLine` 解析命令，通过 `processutil.KillTree` 隐藏执行 `taskkill /F /T /PID` 并停止进程树；`port_windows.go` 通过 Windows TCP 表查询监听 PID、进程名和路径，沿父 PID 链识别其所属服务进程树，并在结束占用进程前复核端口与 PID。
- `internal/notify/feishu.go` — 调用 `lark-cli` 发飞书消息。`Feishu(cliPath, openID, text)` fire-and-forget；`StatusLabel(status)` 返回中文状态文字。脚本和工作流执行结束后均触发通知。
- `internal/scheduler/` — robfig/cron v3 封装，管理定时任务注册/移除，并提供 5 位 cron 表达式标准化与校验。`script_id < 0` 表示工作流定时任务（`-workflowId`）。
- `internal/env/` — .env 文件解析，支持全局 env + 脚本私有 env 双层合并。
- `internal/workflow/executor.go` — Kahn 拓扑排序 + 按层并发执行，任意节点失败则终止后续层；每次运行保存节点名称快照、状态、耗时和脚本运行记录关联，未进入执行层的后续节点标记为 `skipped`；保存/运行前校验节点、脚本引用、连线与循环依赖。`StopWorkflow` 通过独立 context 取消当前工作流并停止运行节点，最终状态为 `killed`。

### 前端（Vue3 + Pinia）

- `stores/main.js` — 全局状态。`selectedScriptListId` 保存当前脚本列表，`scriptListVersion` 刷新列表与脚本栏；`selectedWorkflowId` 控制 WorkflowEditor 加载哪个工作流；`setScriptFromWorkflow` 跳转脚本配置时自动加载最近一次运行日志；`isDirty`/`navigationBlocked` 统一处理未保存页面的离开确认。
- `Sidebar.vue` — 主导航 + 脚本列表管理 + 工作流分组。脚本列表支持新增、重命名、上移、下移和删除，只展示列表名称与脚本数，不展开脚本；服务/定时任务位于顶部主导航，设置固定在底部。
- `ScriptListPane.vue` — 当前列表的紧凑脚本栏，提供搜索、运行状态和新增入口。宽屏与详情组成三栏，窗口宽度不超过 1050px 时在脚本栏与详情间切换。
- `ScriptConfig.vue` — 脚本配置表单。名称下方提供持久化的程序说明；支持 script、module、custom 三种启动模式；选择脚本路径后自动调用 `InferFromScriptPath` 推断虚拟环境解释器和工作目录；custom 模式只显示工作目录和完整命令输入；工作目录可直接用 VS Code 打开。
- `WorkflowEditor.vue` — 拖拽画布（Vue Flow）。左侧脚本列表支持搜索和拖入，节点双击跳转脚本配置并加载最近日志。支持自动布局、复制、定时设置、真实停止和未保存保护。
- `TimerModal.vue` — 定时规则配置弹窗。脚本和工作流编辑页打开时先展示当前目标已有规则，右侧提供启用/禁用、编辑、删除，并在下方新增；脚本视图还会解析工作流 graph，列出所有包含该脚本的工作流定时，以工作流名称标记且保持只读；Schedule 总览支持选择脚本/工作流目标；支持快捷规则（每日一次、每天多时刻、每周、工作日、循环间隔）和自定义 5 位 cron；自定义 cron 支持多行导入，空行忽略，重复行会拦截，同一目标已有启用规则或导入内容之间存在时间交集时提示可能重复触发；每天多时刻和多行导入会保存为多条 `schedules` 记录。
- `ServicesView.vue` — 服务管理控制台。左侧服务列表，右侧服务详情/启动停止可靠重启/编辑/删除/跟随 PyLot 启动开关/工作目录快捷打开/访问链接/端口监听状态/冲突处理/实时日志；日志从后端服务缓冲读取，避免切换页面后丢失。
- `ScheduleView.vue` — 定时任务总览和管理入口。“即将运行”展开未来 24 小时内最多 50 个预计触发实例，同一 cron 可重复出现并按计划运行时间升序排列；“今日运行 / 已停止 / 全部”仍按定时规则每条一行展示；支持新增、编辑、启用/禁用和删除；下方合并展示脚本/工作流最近运行情况，脚本记录直接查看历史日志，工作流记录按节点展示状态、耗时和对应脚本日志，窄窗口使用节点下拉选择器。脚本用蓝色竖线标识，工作流用橙色竖线标识。
- `SettingsView.vue` — 设置页：主题（深色/浅色）、字体、全局 .env 路径、飞书通知（lark-cli 路径 + Open ID）。设置持久化到 `localStorage`（外观）或 DB（env/lark）。
- `LogPanel.vue` — 脚本实时日志，支持搜索、仅看错误、自动滚动、历史记录、折叠，以及用 VS Code 打开工作目录。全局底部日志只在已保存脚本详情显示；工作流运行页使用按节点聚合日志，服务页和定时页使用各自日志区域，设置页不显示输出。

### Wails 事件

后端通过 `runtime.EventsEmit` 推送事件给前端：
- `log:line` — 日志行（含 `isError` 标志）
- `task:status` — 任务状态变更（running/success/error/timeout/killed）
- `task:alert` — 异常弹窗通知
- `workflow:node-status` — 工作流节点状态变更
- `workflow:status` — 工作流整体完成/失败
- `workflow:log` — 工作流聚合日志行（含 `workflowId`、`scriptID`、`isError` 与 `timestamp`）
- `service:log` — 服务 stdout/stderr 日志行（含 `isError` 与 `timestamp`）
- `service:status` — 服务状态变更（含 `status`、`running`、`pid`、`started_at`、`stopped_at`、`exit_code`、`last_error`）
- `tray:schedule` — 托盘点击"定时任务"，前端切换到 Schedule 视图

## 关键约定

- module 模式运行时，后端自动将绝对脚本路径转换为相对 WorkDir 的点号模块名（去 `.py` 后缀，路径分隔符换成 `.`）。
- custom 模式运行时，`script_path` 字段存完整命令，前端只要求工作目录和命令；后端用 Windows 命令行规则解析命令，适合 `python -m package subcommand --flag` 这类入口。
- script/module 固定参数也使用 Windows 命令行规则解析，不再用空格拆分，带引号参数可正常传递。
- 所有子进程（Python 脚本、VSCode）均设置 `HideWindow: true` 避免黑框。
- 脚本停止、服务停止和端口占用进程清理统一调用 `processutil.KillTree`，其 `taskkill` 子进程必须设置 `HideWindow: true`，避免手动停止或退出 PyLot 时闪出控制台窗口。
- Python 脚本注入 `PYTHONIOENCODING=utf-8`，统一 UTF-8 输出，无需 GBK 解码。
- 托盘双图标：`build/windows/icon_free.ico`（空闲）/ `build/windows/icon_busy.ico`（有脚本运行），通过 `//go:embed` 内嵌，`script.OnRunningChange` 回调切换。
- systray goroutine 必须 `goruntime.LockOSThread()`，否则休眠唤醒后消息泵失效。
- SQLite 使用 WAL 模式 + `busy_timeout=5000`，并通过 `db.ExecWrite` 串行化核心写入、对 busy/locked 做短重试，防止并发日志写入与状态更新互相阻塞导致状态停在 running。
- 脚本归属以 `scripts.list_id` 为准；`list_id IS NULL` 表示“未分类”。“全部脚本”和“未分类”是前端系统列表，不写入 `script_lists`；删除用户列表只清空关联脚本的 `list_id`。
- 程序说明保存在 `scripts.description`；旧数据库启动时自动补列，说明参与脚本新增、编辑和复制。
- 工作流定时任务在 `schedules` 表中用负数 `script_id`（`-workflowId`）存储，`addScheduleJob` 统一处理正负数分发。
- 工作流节点历史以 `workflow_run_nodes.workflow_run_id + node_id` 区分，同一脚本在一个工作流中出现多次时不得只按 `script_id` 或时间范围推断日志归属；节点日志通过 `run_record_id` 复用 `run_records.log_output`。
- 删除脚本/工作流时必须同步移除关联 schedule 和内存 scheduler job；被工作流引用的脚本禁止直接删除。定时任务注册失败必须回滚数据库与旧 scheduler 状态。
- 新建或有未保存修改的脚本/工作流不能运行；运行中不能删除或修改结构性配置。所有异步操作必须立即显示处理中状态并阻止重复提交。
- 服务配置存储在 `services` 表；`auto_start=1` 表示跟随 PyLot 启动。`startup()` 在 DB 初始化和调度器加载后调用 `autoStartServices()` 拉起自启服务。
- 服务运行态只保存在内存中，不落库；服务页通过 `ListServices()` 获取当前快照，通过 `GetServiceLogs()` 获取本次会话后端日志缓冲。
- 服务命令使用 Windows 命令行规则解析，支持带空格路径和引号参数；相对可执行文件若能在 WorkDir 下找到，会解析为 WorkDir 相对路径，否则交给系统 PATH 查找。
- 服务的 `port=0` 表示不启用端口管理；配置端口后仅生成 `http(s)://127.0.0.1:{port}` 访问入口并检测 TCP 监听，不自动修改服务命令。端口冲突必须展示占用进程并经用户确认，结束前再次核对 PID；重启必须等待进程退出和端口释放。
- 窗口大小通过 `localStorage`（`winW`/`winH`）持久化，启动时通过 `SetWindowSize` 恢复。
- 主题/字体通过 `localStorage`（`theme`/`font`）持久化，启动时设置 `data-theme` 属性和 `--font` CSS 变量。
- 飞书通知配置（`lark_cli_path`/`lark_open_id`）存 `global_config` 表；`notify.Feishu` 参数为空时静默跳过，不影响正常运行。
- PyLot 启动和退出时调用 `script.CleanupStaleRuns()` 收敛脚本运行态：遗留的 `running` 记录会标记为 `killed`，`running_tasks` 会清空，避免异常中断后历史日志假卡住。
