// Диагностика: повторяет поток cogentcore-биндинга на чистом C.
// Если падает здесь — дефект внутри libwgpu_native.a (шим не поможет).
// Если работает — баг на слое Go-биндинга, и свой шим починит WebGPU.
#include <stdio.h>
#include "wgpu.h"

static void adapter_cb(WGPURequestAdapterStatus status, WGPUAdapter adapter,
                       const char *message, void *userdata) {
    (void)status; (void)message;
    *(WGPUAdapter *)userdata = adapter;
}

static void device_cb(WGPURequestDeviceStatus status, WGPUDevice device,
                      const char *message, void *userdata) {
    (void)status; (void)message;
    *(WGPUDevice *)userdata = device;
}

int main(void) {
    WGPUInstance inst = wgpuCreateInstance(NULL);
    if (!inst) { printf("FAIL: instance\n"); return 1; }

    WGPUAdapter adapter = NULL;
    WGPURequestAdapterOptions opts = {0};
    wgpuInstanceRequestAdapter(inst, &opts, adapter_cb, &adapter);
    if (!adapter) { printf("FAIL: adapter\n"); return 1; }
    printf("adapter ok\n");

    WGPUDevice device = NULL;
    wgpuAdapterRequestDevice(adapter, NULL, device_cb, &device);
    if (!device) { printf("FAIL: device (NULL)\n"); return 1; }
    printf("device ok, hasFeature(F16)=%d\n",
           wgpuDeviceHasFeature(device, WGPUFeatureName_ShaderF16));

    // Тот же вызов, что паникует через Go-биндинг:
    WGPUCommandEncoder enc = wgpuDeviceCreateCommandEncoder(device, NULL);
    if (!enc) { printf("FAIL: encoder is NULL\n"); return 1; }
    printf("ENCODER OK: %p\n", (void *)enc);

    // И полный кадр для верности: buffer + command buffer + submit пустого прохода нельзя
    // без поверхности, поэтому просто фиксируем, что энкодер создался.
    printf("C-LEVEL OK\n");
    return 0;
}
