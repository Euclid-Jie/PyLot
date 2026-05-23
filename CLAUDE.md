# PyLot — 开发文档

Python 脚本调度桌面管理系统。技术栈：Go + Wails v2 + Vue3 + SQLite（纯 Go 驱动）。

## 项目结构

```
PyLot/                                 # 仓库根目录（即项目目录）
├── main.go                        # 入口：Wails 启动 + systray 托盘
├── app.go                         # 所有暴露给前端的后端方法
├── internal/
│   ├── db/database.go             # SQLite 初始化、建表、日志清理
│   ├── db/models.go               # 数据结构定义
│   ├── script/manager.go          # 脚本 CRUD
│   ├── script/runner.go           # 进程启动、日志捕获、强杀、卡死检测
│   ├── env/loader.go              # .env 解析、双层环境合并
│   └── scheduler/scheduler.go    # cron 调度管理
├── frontend/src/
│   ├── App.vue                    # 主布局（无顶部 env-bar，已移到侧边栏）
│   ├── stores/main.js             # Pinia 状态（含 scriptListVersion 刷新信号）
│   └── components/
│       ├── Sidebar.vue            # 分组列表 + 全局.env + Schedule 按钮
│       ├── ScriptConfig.vue       # 脚本配置表单（含保存 toast、复制按钮）
│       ├── LogPanel.vue           # 实时日志面板
│       ├── ScheduleView.vue       # 定时任务总览
│       ├── HistoryModal.vue       # 历史日志弹窗
│       ├── TimerModal.vue         # 定时规则配置弹窗
│       └── TempArgsModal.vue      # 临时参数输入弹窗
└── build/
    └── windows/icon.ico           # 托盘图标（embed 进二进制）
```

## 关键设计决策

- **SQLite 驱动**：使用 `modernc.org/sqlite`（纯 Go），不需要 CGO/GCC，避免 Windows 编译依赖
- **中文日志**：进程 stdout/stderr 用 `golang.org/x/text/encoding/simplifiedchinese.GBK` 解码
- **隐藏子进程黑框**：`cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}`
- **托盘最小化**：`OnBeforeClose` 返回 `true` + `runtime.WindowHide`；systray 在独立 goroutine 运行
- **侧边栏实时刷新**：保存/删除脚本后调用 `store.refreshScriptList()`，Sidebar watch `scriptListVersion`

## 打包命令

```bash
wails build -platform windows/amd64 -ldflags "-H windowsgui"
```

输出：`build/bin/PyLot.exe`（无控制台黑框）

数据库文件 `PyLot.db` 在 exe 同目录自动创建。

---

## 新机器环境搭建

### 1. 安装工具链（需要网络，推荐用 winget）

```powershell
winget install GoLang.Go
winget install OpenJS.NodeJS.LTS
```

安装完成后**重开终端**使 PATH 生效。

### 2. 配置 Go 代理（国内必须）

```bash
go env -w GOPROXY=https://goproxy.cn,direct
go env -w GONOSUMDB=*
```

### 3. 安装 Wails CLI

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

### 4. 克隆代码后安装依赖

```bash
cd PyLot

# Go 依赖（自动从 go.mod 安装）
go mod download

# 前端依赖
cd frontend && npm install && cd ..
```

### 5. 开发模式运行

```bash
wails dev
```

### 6. 打包

```bash
wails build -platform windows/amd64 -ldflags "-H windowsgui"
```

### 注意

- 不需要安装 GCC/MinGW（已用纯 Go SQLite 驱动）
- 不需要安装 Python（Python 解释器路径由用户在 UI 里配置）
- WebView2 运行时：Windows 10/11 系统自带，无需额外安装
