# AGENTS.md

This file provides guidance to Codex (Codex.ai/code) when working with code in this repository.

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

- `main.go` — Wails 启动 + systray 托盘（`goruntime.LockOSThread()` 保证消息泵稳定）。命名 Mutex 防多开；双托盘图标（`icon_free.ico` / `icon_busy.ico`）随运行状态切换；关闭窗口触发 `OnBeforeClose` 隐藏到托盘；托盘右键菜单：显示主窗口 / 定时任务 / 退出（`os.Exit(0)`）。
- `app.go` — 所有暴露给前端的方法（Wails bind）。包含文件对话框、VSCode 打开、脚本推断、窗口大小读写、Workflow CRUD、Service CRUD/控制/日志等。
- `internal/db/` — SQLite 初始化建表（8张表）+ WAL 模式（防并发写丢失）+ `busy_timeout=5000`；`global_config` 含 `lark_cli_path`/`lark_open_id` 字段；`services` 表保存服务命令、工作目录、跟随 PyLot 启动开关；日志清理（7天）。数据库文件在 exe 同目录。
- `internal/script/runner.go` — 进程启动，注入 `PYTHONIOENCODING=utf-8` 统一编码，`SysProcAttr{HideWindow: true}` 隐藏黑框，支持卡死超时检测；module 模式自动将绝对路径转换为相对 WorkDir 的点号模块名。
- `internal/service/manager.go` — 长期运行服务进程管理。维护 `starting/running/stopping/exited/failed/stopped` 运行态、PID、启动/停止时间、退出码、最近错误和本次会话最近 1000 行日志；使用 Windows `DecomposeCommandLine` 解析命令，`taskkill /F /T /PID` 停止进程树。
- `internal/notify/feishu.go` — 调用 `lark-cli` 发飞书消息。`Feishu(cliPath, openID, text)` fire-and-forget；`StatusLabel(status)` 返回中文状态文字。脚本和工作流执行结束后均触发通知。
- `internal/scheduler/` — robfig/cron v3 封装，管理定时任务注册/移除，并提供 5 位 cron 表达式标准化与校验。`script_id < 0` 表示工作流定时任务（`-workflowId`）。
- `internal/env/` — .env 文件解析，支持全局 env + 脚本私有 env 双层合并。
- `internal/workflow/executor.go` — Kahn 拓扑排序 + 按层并发执行，任意节点失败则终止后续层。

### 前端（Vue3 + Pinia）

- `stores/main.js` — 全局状态。`scriptListVersion` 刷新侧边栏；`selectedWorkflowId` 控制 WorkflowEditor 加载哪个工作流；`setScriptFromWorkflow` 跳转脚本配置时自动加载最近一次运行日志。
- `Sidebar.vue` — 脚本管理（含分类列表）+ 工作流管理，各有独立 header 和 + 按钮。底部：服务 / 定时任务 / 设置入口并排，并按当前视图高亮。
- `ScriptConfig.vue` — 脚本配置表单。选择脚本路径后自动调用 `InferFromScriptPath` 推断虚拟环境解释器和工作目录。
- `WorkflowEditor.vue` — 拖拽画布（Vue Flow）。左侧脚本列表可拖入，节点双击跳转脚本配置并加载最近日志。支持自动布局、复制、定时设置。
- `TimerModal.vue` — 定时规则配置弹窗。支持新建和编辑已有 schedule，支持从 Schedule 总览选择脚本/工作流目标，支持快捷规则（每日一次、每天多时刻、每周、工作日、循环间隔）和自定义 5 位 cron；每天多时刻会保存为多条 `schedules` 记录。
- `ServicesView.vue` — 服务管理控制台。左侧服务列表，右侧服务详情/启动停止重启/编辑/删除/跟随 PyLot 启动开关/实时日志；日志从后端服务缓冲读取，避免切换页面后丢失。
- `ScheduleView.vue` — 定时任务总览和管理入口。任务列表默认展示即将运行的启用任务，并按下次运行时间升序排列；提供“即将运行 / 今日运行 / 已停止 / 全部”筛选；支持新增、编辑、启用/禁用和删除；下方合并展示脚本/工作流最近运行情况，脚本记录可查看历史日志。脚本用蓝色竖线标识，工作流用橙色竖线标识。
- `SettingsView.vue` — 设置页：主题（深色/浅色）、字体、全局 .env 路径、飞书通知（lark-cli 路径 + Open ID）。设置持久化到 `localStorage`（外观）或 DB（env/lark）。
- `LogPanel.vue` — 脚本实时日志，含 VSCode 图标按钮（调用 `OpenInVSCode(workDir)`）。全局底部日志面板不在 Workflow、Services、Schedule 视图显示；服务页使用服务自己的 stdout/stderr 日志，定时页使用最近运行情况面板。

### Wails 事件

后端通过 `runtime.EventsEmit` 推送事件给前端：
- `log:line` — 日志行（含 `isError` 标志）
- `task:status` — 任务状态变更（running/success/error/timeout/killed）
- `task:alert` — 异常弹窗通知
- `workflow:node-status` — 工作流节点状态变更
- `workflow:status` — 工作流整体完成/失败
- `service:log` — 服务 stdout/stderr 日志行（含 `isError` 与 `timestamp`）
- `service:status` — 服务状态变更（含 `status`、`running`、`pid`、`started_at`、`stopped_at`、`exit_code`、`last_error`）
- `tray:schedule` — 托盘点击"定时任务"，前端切换到 Schedule 视图

## 关键约定

- module 模式运行时，后端自动将绝对脚本路径转换为相对 WorkDir 的点号模块名（去 `.py` 后缀，路径分隔符换成 `.`）。
- 所有子进程（Python 脚本、VSCode）均设置 `HideWindow: true` 避免黑框。
- Python 脚本注入 `PYTHONIOENCODING=utf-8`，统一 UTF-8 输出，无需 GBK 解码。
- 托盘双图标：`build/windows/icon_free.ico`（空闲）/ `build/windows/icon_busy.ico`（有脚本运行），通过 `//go:embed` 内嵌，`script.OnRunningChange` 回调切换。
- systray goroutine 必须 `goruntime.LockOSThread()`，否则休眠唤醒后消息泵失效。
- SQLite 使用 WAL 模式 + `busy_timeout=5000`，防止并发日志写入与状态更新互相阻塞导致状态停在 running。
- 工作流定时任务在 `schedules` 表中用负数 `script_id`（`-workflowId`）存储，`addScheduleJob` 统一处理正负数分发。
- 服务配置存储在 `services` 表；`auto_start=1` 表示跟随 PyLot 启动。`startup()` 在 DB 初始化和调度器加载后调用 `autoStartServices()` 拉起自启服务。
- 服务运行态只保存在内存中，不落库；服务页通过 `ListServices()` 获取当前快照，通过 `GetServiceLogs()` 获取本次会话后端日志缓冲。
- 服务命令使用 Windows 命令行规则解析，支持带空格路径和引号参数；相对可执行文件若能在 WorkDir 下找到，会解析为 WorkDir 相对路径，否则交给系统 PATH 查找。
- 窗口大小通过 `localStorage`（`winW`/`winH`）持久化，启动时通过 `SetWindowSize` 恢复。
- 主题/字体通过 `localStorage`（`theme`/`font`）持久化，启动时设置 `data-theme` 属性和 `--font` CSS 变量。
- 飞书通知配置（`lark_cli_path`/`lark_open_id`）存 `global_config` 表；`notify.Feishu` 参数为空时静默跳过，不影响正常运行。
