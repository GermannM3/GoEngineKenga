// Репро: RequestDevice + CreateCommandEncoder через cogentcore-биндинг.
package main

import (
	"fmt"
	"time"

	"github.com/cogentcore/webgpu/wgpuglfw"
	"github.com/go-gl/glfw/v3.3/glfw"

	"github.com/cogentcore/webgpu/wgpu"
)

func main() {
	if err := glfw.Init(); err != nil {
		fmt.Println("INIT ERR:", err)
		return
	}
	defer glfw.Terminate()
	glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)
	window, err := glfw.CreateWindow(640, 360, "repro", nil, nil)
	if err != nil {
		fmt.Println("WINDOW ERR:", err)
		return
	}
	defer window.Destroy()

	instance := wgpu.CreateInstance(nil)
	surface := instance.CreateSurface(wgpuglfw.GetSurfaceDescriptor(window))
	adapter, err := instance.RequestAdapter(&wgpu.RequestAdapterOptions{CompatibleSurface: surface})
	if err != nil {
		fmt.Println("ADAPTER ERR:", err)
		return
	}
	device, err := adapter.RequestDevice(&wgpu.DeviceDescriptor{Label: "dev"})
	if err != nil {
		fmt.Println("DEVICE ERR:", err)
		return
	}
	fmt.Println("device.HasFeature(F16):", device.HasFeature(wgpu.FeatureNameShaderF16))

	caps := surface.GetCapabilities(adapter)
	w, h := window.GetSize()
	surface.Configure(adapter, device, &wgpu.SurfaceConfiguration{
		Usage: wgpu.TextureUsageRenderAttachment, Format: caps.Formats[0],
		Width: uint32(w), Height: uint32(h),
		PresentMode: wgpu.PresentModeFifo, AlphaMode: caps.AlphaModes[0],
	})

	tex, err := surface.GetCurrentTexture()
	if err != nil {
		fmt.Println("TEX ERR:", err)
		return
	}
	view, err := tex.CreateView(nil)
	if err != nil {
		fmt.Println("VIEW ERR:", err)
		return
	}
	enc, err := device.CreateCommandEncoder(&wgpu.CommandEncoderDescriptor{Label: "enc"})
	if err != nil {
		fmt.Println("ENC ERR:", err)
		return
	}
	rp := enc.BeginRenderPass(&wgpu.RenderPassDescriptor{
		ColorAttachments: []wgpu.RenderPassColorAttachment{
			{View: view, LoadOp: wgpu.LoadOpClear, StoreOp: wgpu.StoreOpStore, ClearValue: wgpu.Color{R: 0.2, G: 0.4, B: 0.8, A: 1}},
		},
	})
	rp.End()
	rp.Release()
	cb, err := enc.Finish(nil)
	if err != nil {
		fmt.Println("FINISH ERR:", err)
		return
	}
	device.GetQueue().Submit(cb)
	surface.Present()
	fmt.Println("REPRO FULL FRAME OK")
	time.Sleep(300 * time.Millisecond)
	time.Sleep(300 * time.Millisecond)
}
