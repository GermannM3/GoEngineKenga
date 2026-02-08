@echo off
cd /d d:\GoEngineKenga
go run ./cmd/kenga run --project samples/hello --scene scenes/simple.scene.json --backend ebiten
pause