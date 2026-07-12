package main

import (
	"context"
	"embed"
	"os"
	goruntime "runtime"

	"script-manager/internal/script"

	"github.com/getlantern/systray"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/windows/icon_free.ico
var trayIcon []byte

//go:embed build/windows/icon_busy.ico
var trayIconBusy []byte

func main() {
	// Wails 单实例锁会把后续启动请求转发给当前实例。
	app := NewApp()
	ctxReady := make(chan struct{})

	script.OnRunningChange = func(count int) {
		if count > 0 {
			systray.SetIcon(trayIconBusy)
		} else {
			systray.SetIcon(trayIcon)
		}
	}

	go func() {
		goruntime.LockOSThread()
		systray.Run(func() {
			systray.SetIcon(trayIcon)
			systray.SetTitle("PyLot")
			systray.SetTooltip("PyLot")

			show := systray.AddMenuItem("显示主窗口", "")
			schedule := systray.AddMenuItem("定时任务", "")
			systray.AddSeparator()
			quit := systray.AddMenuItem("退出程序", "")

			go func() {
				<-ctxReady // 等 ctx 就绪再处理点击
				for {
					select {
					case <-show.ClickedCh:
						runtime.WindowShow(app.ctx)
					case <-schedule.ClickedCh:
						runtime.WindowShow(app.ctx)
						runtime.EventsEmit(app.ctx, "tray:schedule")
					case <-quit.ClickedCh:
						app.shutdown(app.ctx)
						os.Exit(0)
					}
				}
			}()
		}, nil)
	}()

	err := wails.Run(&options.App{
		Title:  "PyLot",
		Width:  1280,
		Height: 800,
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "pylot-single-instance",
			OnSecondInstanceLaunch: func(_ options.SecondInstanceData) {
				<-ctxReady
				runtime.WindowShow(app.ctx)
			},
		},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup: func(ctx context.Context) {
			app.startup(ctx)
			close(ctxReady)
		},
		OnShutdown: app.shutdown,
		OnBeforeClose: func(ctx context.Context) bool {
			runtime.WindowHide(ctx)
			return true
		},
		Bind: []interface{}{app},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
