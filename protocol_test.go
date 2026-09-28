package gs2

import (
	"bytes"
	"math"
	"testing"
)

// 校验 buildPacket 生成的报文与开发手册示例一致。
func TestBuildPacket(t *testing.T) {
	cases := []struct {
		name string
		addr byte
		cmd  byte
		data []byte
		want []byte
	}{
		{"获取地址", AddrAll, CmdGetAddress, nil, []byte{0xA5, 0xA5, 0xA5, 0xA5, 0x00, 0x60, 0x00, 0x00, 0x60}},
		{"获取参数", AddrAll, CmdGetParams, nil, []byte{0xA5, 0xA5, 0xA5, 0xA5, 0x00, 0x61, 0x00, 0x00, 0x61}},
		{"获取版本", AddrAll, CmdGetVersion, nil, []byte{0xA5, 0xA5, 0xA5, 0xA5, 0x00, 0x62, 0x00, 0x00, 0x62}},
		{"开始扫描", AddrAll, CmdStartScan, nil, []byte{0xA5, 0xA5, 0xA5, 0xA5, 0x00, 0x63, 0x00, 0x00, 0x63}},
		{"停止扫描", AddrAll, CmdStopScan, nil, []byte{0xA5, 0xA5, 0xA5, 0xA5, 0x00, 0x64, 0x00, 0x00, 0x64}},
	}
	for _, c := range cases {
		got := buildPacket(c.addr, c.cmd, c.data)
		if !bytes.Equal(got, c.want) {
			t.Errorf("%s: 报文错误\n got %X\nwant %X", c.name, got, c.want)
		}
	}
}

func TestParseParams(t *testing.T) {
	// K0=12345/10000=1.2345, B0=6000/10000=0.6, K1=9876/10000=0.9876, B1=15000/10000=1.5, Bias=-3/10=-0.3
	data := []byte{
		0x39, 0x30, // 12345
		0x70, 0x17, // 6000
		0x94, 0x26, // 9876
		0x98, 0x3A, // 15000
		0xFD, // -3
	}
	p, err := parseParams(data)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(p.K0-1.2345) > 1e-6 ||
		math.Abs(p.B0-0.6) > 1e-6 ||
		math.Abs(p.K1-0.9876) > 1e-6 ||
		math.Abs(p.B1-1.5) > 1e-6 ||
		math.Abs(p.Bias+0.3) > 1e-6 {
		t.Errorf("参数解析错误: %+v", p)
	}
}

func TestParseVersion(t *testing.T) {
	data := make([]byte, 19)
	data[0] = 0x01                            // hw
	data[1] = 0x12                            // fw 高字节（大端）
	data[2] = 0x34                            // fw 低字节 -> 0x1234
	copy(data[3:], []byte("SN1234567890123")) // 16 bytes
	v, err := parseVersion(0x02, data)
	if err != nil {
		t.Fatal(err)
	}
	if v.Address != 0x02 || v.HWVersion != 1 || v.FWVersion != 0x1234 {
		t.Errorf("版本解析错误: %+v", v)
	}
	if v.SerialString() != "SN1234567890123" {
		t.Errorf("序列号解析错误: %q", v.SerialString())
	}
}

func TestTransformPoint(t *testing.T) {
	p := &DeviceParams{K0: 0.5, B0: 0.2, K1: 0.5, B1: 0.2, Bias: 0.1}
	for i := 0; i < PointCount; i++ {
		angle, dist := transformPoint(float64(100), i, p)
		if math.IsNaN(angle) || math.IsInf(angle, 0) {
			t.Errorf("点 %d 角度非法: %v", i, angle)
		}
		if angle < 0 || angle >= 360 {
			t.Errorf("点 %d 角度越界: %v", i, angle)
		}
		if math.IsNaN(dist) || math.IsInf(dist, 0) || dist < 0 {
			t.Errorf("点 %d 距离非法: %v", i, dist)
		}
	}
}

// 校验 GS2 距离/强度掩码：低 9 位距离 + 高 7 位强度。
func TestParseFrameMask(t *testing.T) {
	data := make([]byte, PointCloudDataLen)
	// 环境数据（小端）
	data[0], data[1] = 0x34, 0x12
	// 第 5 个点（index 4）raw = 0x8010：高 7 位 = 0x40(64)，低 9 位 = 0x10(16)
	off := EnvDataLen + 4*PointDataLen
	data[off], data[off+1] = 0x10, 0x80

	p := &DeviceParams{K0: 0.5, B0: 0.2, K1: 0.5, B1: 0.2, Bias: 0.1}
	frame, err := parseFrame(0x01, data, p)
	if err != nil {
		t.Fatal(err)
	}
	if frame.Env != 0x1234 {
		t.Errorf("环境数据解析错误: 0x%X", frame.Env)
	}
	if len(frame.Points) != PointCount {
		t.Fatalf("点数量错误: %d", len(frame.Points))
	}
	if frame.Points[4].Intensity != 64 {
		t.Errorf("强度掩码错误: 期望 64(高7位), 实际 %d", frame.Points[4].Intensity)
	}
}
