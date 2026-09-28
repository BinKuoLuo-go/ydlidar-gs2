# ydlidar-gs2

YDLIDAR GS2 固态激光雷达的 Go SDK。基于官方 GS2 开发手册与官方开源 SDK 源码实现完整的 golang 版本串口通信协议与点云解析，供其他 go 项目集成。

## 特性

- 完整系统命令：获取地址 / 版本 / 参数、开始扫描、停止扫描、软重启、设置波特率、设置沿边模式（GS2 特有）
- 点云解析：距离（mm）、角度（度）、强度、环境光强度
- 角度计算采用官方 SDK 机械参数（`Angle_Px=1.22`、`Angle_Py=5.315`、俯仰角 `22.5°`）
- 距离/强度掩码：低 9 位距离 + 高 7 位强度
- 左右相机越界点过滤（与官方 SDK 一致）
- 支持级联（最多 3 个模组，地址 0x01/0x02/0x04）
- 自动初始化（推荐流程：获取地址 → 获取参数）

## 安装

```bash
go get github.com/BinKuoLuo-go/ydlidar-gs2
```

> 依赖 `go.bug.st/serial.v1` 串口库，会随模块自动拉取。

## 快速开始

```go
package main

import (
	"log"

	gs2 "github.com/BinKuoLuo-go/ydlidar-gs2"
)

func main() {
	dev, err := gs2.Open("COM3")
	if err != nil {
		log.Fatal(err)
	}
	defer dev.Close()

	// 启动扫描
	if err := dev.StartScan(); err != nil {
		log.Fatal(err)
	}

	// 读取点云
	for {
		frame, err := dev.GrabFrame()
		if err != nil {
			log.Fatal(err)
		}
		for _, p := range frame.Points {
			// p.Angle(度) / p.Distance(mm) / p.Intensity(0~127)
		}
	}
}
```

完整示例见 [`examples/main.go`](examples/main.go)。

## API

| 方法 | 说明 |
| --- | --- |
| `Open(port string, opts ...Option) (*Device, error)` | 打开串口并初始化 |
| `(*Device).GetDeviceAddress() (byte, error)` | 获取级联地址（0x01/0x02/0x04 对应 1/2/3 个模组） |
| `(*Device).GetVersion() ([]*VersionInfo, error)` | 获取所有设备版本信息 |
| `(*Device).GetParams() (map[byte]*DeviceParams, error)` | 获取所有设备计算参数 |
| `(*Device).StartScan() error` | 进入扫描模式 |
| `(*Device).StopScan() error` | 停止扫描 |
| `(*Device).GrabFrame() (*Frame, error)` | 读取一帧点云（阻塞） |
| `(*Device).SetEdgeMode(addr byte, mode uint8) (uint8, error)` | 设置/读取沿边模式（GS2 特有） |
| `(*Device).SoftReset(addr byte) error` | 软重启指定模组 |
| `(*Device).SetBaudRate(code uint8) error` | 设置波特率（0~3） |
| `(*Device).Close() error` | 关闭设备 |

## 沿边模式

GS2 支持沿边模式（强抗扰模式），通过 `SetEdgeMode` 设置：

```go
// 设置沿边模式（插座朝上），返回当前模式
mode, err := dev.SetEdgeMode(gs2.AddrDev1, gs2.EdgeModeUp)

// 读取当前模式
mode, err = dev.SetEdgeMode(gs2.AddrDev1, gs2.EdgeModeRead)
```

| 常量 | 值 | 说明 |
| --- | --- | --- |
| `EdgeModeAvoid` | 0x00 | 避障模式（出厂默认） |
| `EdgeModeUp` | 0x01 | 沿边模式（插座朝上） |
| `EdgeModeDown` | 0x02 | 沿边模式（插座朝下） |
| `EdgeModeRead` | 0xFF | 读取当前模式 |

## 配置项

```go
dev, _ := gs2.Open("COM3",
	gs2.WithBaudRate(921600),            // 波特率
	gs2.WithLogger(log.Default()),       // 调试日志
)
```

## 数据类型

- `DeviceParams`：设备角度参数 `K0/B0/K1/B1/Bias`
- `Point`：`Index`、`Angle`（度，顺时针，0° 为正前方）、`Distance`（mm）、`Intensity`（0~127）
- `Frame`：`Address`、`Env`（环境光）、`Points`（160 个点）
- `VersionInfo`：`HWVersion`、`FWVersion`、`SerialNo`

## 注意事项

1. **并发**：`Device` 的方法并非并发安全，串口读写请在单个 goroutine 中调用。
2. **阻塞**：`GrabFrame`、命令应答读取均为阻塞操作，需确保设备已连接。
3. **扫描模式**：除停止扫描外，其他命令不能在扫描模式下交互。
4. **波特率**：三模块级联需 ≥ 921600（代号 2/3）；设置波特率后 SDK 会自动软重启并重配串口。
