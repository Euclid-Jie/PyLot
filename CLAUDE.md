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

- `main.go` — Wails 启动 + systray 托盘（独立 goroutine）。关闭窗口触发 `OnBeforeClose` 隐藏到托盘；托盘右键菜单：显示主窗口 / 定时任务 / 退出（`os.Exit(0)`）。
- `app.go` — 所有暴露给前端的方法（Wails bind）。包含文件对话框、VSCode 打开、脚本推断、窗口大小读写、Workflow CRUD 等。
- `internal/db/` — SQLite 初始化建表（7张表：scripts/schedules/run_records/running_tasks/global_config/workflows/workflow_runs）、日志清理（7天）。数据库文件在 exe 同目录。
- `internal/script/runner.go` — 进程启动，stdout/stderr 用 GBK 解码，`SysProcAttr{HideWindow: true}` 隐藏黑框，支持卡死超时检测。
- `internal/scheduler/` — robfig/cron v3 封装，管理定时任务注册/移除。`script_id < 0` 表示工作流定时任务（`-workflowId`）。
- `internal/env/` — .env 文件解析，支持全局 env + 脚本私有 env 双层合并。
- `internal/workflow/executor.go` — Kahn 拓扑排序 + 按层并发执行，任意节点失败则终止后续层。

### 前端（Vue3 + Pinia）

- `stores/main.js` — 全局状态。`scriptListVersion` 刷新侧边栏；`selectedWorkflowId` 控制 WorkflowEditor 加载哪个工作流；`setScriptFromWorkflow` 跳转脚本配置时自动加载最近一次运行日志。
- `Sidebar.vue` — 脚本管理（含分类列表）+ 工作流管理，各有独立 header 和 + 按钮。底部：📅 Schedule 和 ⚙ 设置并排。
- `ScriptConfig.vue` — 脚本配置表单。选择脚本路径后自动调用 `InferFromScriptPath` 推断虚拟环境解释器和工作目录。
- `WorkflowEditor.vue` — 拖拽画布（Vue Flow）。左侧脚本列表可拖入，节点双击跳转脚本配置并加载最近日志。支持自动布局、复制、定时设置。
- `ScheduleView.vue` — 定时任务总览，显示所有任务（含禁用）。脚本用蓝色竖线标识，工作流用橙色竖线标识。
- `SettingsView.vue` — 设置页：主题（深色/浅色）、字体、全局 .env 路径。设置持久化到 `localStorage`。
- `LogPanel.vue` — 实时日志，含 VSCode 图标按钮（调用 `OpenInVSCode(workDir)`）。

### Wails 事件

后端通过 `runtime.EventsEmit` 推送事件给前端：
- `log:line` — 日志行（含 `isError` 标志）
- `task:status` — 任务状态变更（running/success/error/timeout/killed）
- `task:alert` — 异常弹窗通知
- `workflow:node-status` — 工作流节点状态变更
- `workflow:status` — 工作流整体完成/失败
- `tray:schedule` — 托盘点击"定时任务"，前端切换到 Schedule 视图

## 关键约定

- module 模式运行时，后端自动去掉 `.py` 后缀并取文件名作为模块名（从 WorkDir 运行）。
- 所有子进程（Python 脚本、VSCode）均设置 `HideWindow: true` 避免黑框。
- 托盘图标 `build/windows/icon.ico` 通过 `//go:embed` 内嵌进二进制。
- 工作流定时任务在 `schedules` 表中用负数 `script_id`（`-workflowId`）存储，`addScheduleJob` 统一处理正负数分发。
- 窗口大小通过 `localStorage`（`winW`/`winH`）持久化，启动时通过 `SetWindowSize` 恢复。
- 主题/字体通过 `localStorage`（`theme`/`font`）持久化，启动时设置 `data-theme` 属性和 `--font` CSS 变量。
