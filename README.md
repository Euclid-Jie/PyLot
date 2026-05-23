# PyLot

Python 脚本调度桌面管理系统，适合需要定期运行多个 Python 脚本的场景。

## 功能

- 按分类管理 Python 脚本（爬取上传 / 数据处理 / 个人工具）
- 支持 script 和 module 两种启动模式，自动识别虚拟环境解释器
- 实时日志输出，支持中文（GBK 自动解码）
- cron 定时调度，支持每日/每周规则，脚本和工作流均可设置
- 全局 .env + 脚本私有环境变量双层合并
- 卡死超时检测与强杀
- **Workflow**：拖拽编排多脚本依赖关系，支持并发执行和错误终止
- 关闭窗口最小化到系统托盘（右键菜单：定时任务 / 退出）
- 深色/浅色主题切换，字体可选，窗口大小记忆

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
