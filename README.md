# PyLot

Python 脚本调度桌面管理系统，适合需要定期运行多个 Python 脚本的场景。

## 功能

- 按分类管理 Python 脚本，支持 script 和 module 两种启动模式，自动识别虚拟环境解释器
- 实时日志输出（UTF-8，支持中文）
- cron 定时调度，脚本和工作流均可设置
- 全局 .env + 脚本私有环境变量双层合并
- 卡死超时检测与强杀
- **Workflow**：拖拽编排多脚本依赖关系，支持并发执行和错误终止
- **飞书通知**：脚本或工作流执行结束后通过 lark-cli 发送飞书消息
- 关闭窗口最小化到系统托盘（右键菜单：定时任务 / 退出），有脚本运行时图标变色
- 深色/浅色主题切换，字体可选，窗口大小记忆
- 单实例运行（命名 Mutex 防多开）

## 技术栈

Go + [Wails v2](https://wails.io) + Vue3 + SQLite（纯 Go 驱动，无需 GCC）

## 开发

```bash
# 安装依赖
go mod download
cd frontend && npm install && cd ..

# 开发模式（热重载）
wails dev

# 打包
wails build -platform windows/amd64 -ldflags "-H windowsgui"
```

详细环境搭建见 [CLAUDE.md](./CLAUDE.md)。

## 要求

- Windows 10/11
- WebView2（系统自带）
- Python 解释器路径在 UI 中配置
- 飞书通知可选：需安装 `lark-cli` 并在设置页填入路径和 Open ID

## 飞书通知配置

1. 安装 lark-cli：`npm install -g @larksuiteoapi/lark-cli`
2. 初始化：`lark-cli config init`（填入 App ID 和 App Secret）
3. 在 PyLot 设置页填入 lark-cli 路径和你的 Open ID
