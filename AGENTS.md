# Repository Guidelines

## 協作語言

本專案的 Git commit 訊息、Pull Request 標題與說明，以及 AI 協作者的進度更新和最終回覆，一律使用繁體中文。程式識別字、檔名、指令和原始錯誤訊息保留原文。

## Project Structure

MySnake is a native Windows Snake game written in Go 1.24. The repository uses a single Go module and keeps its source, tests, and assets at the root.

- `main.go` contains the game state, movement, collision detection, scoring, and food placement.
- `window_windows.go` handles the Win32 window, keyboard input, timer, and rendering. Keep game rules in `main.go` so tests can exercise them without opening a window.
- `main_test.go` contains gameplay tests.
- `snake.png` is the source artwork. `snake.ico` and `snake.rc` define the Windows icon, and `snake_windows_amd64.syso` embeds it in Windows amd64 builds. Update the icon resources together when replacing the artwork.

## Build, Test, and Run

Run these commands from the repository root on Windows:

- `go run .` opens the game for local development.
- `go build -ldflags=-H=windowsgui -o mysnake.exe .` produces a Windows executable that opens without a console. Launch it with `.\mysnake.exe` in PowerShell.
- `go test ./...` runs the gameplay tests.
- `go vet ./...` checks the code for common mistakes.

Keep `mysnake.exe` local. Git ignores this build output.

## Coding Style

Run `gofmt -w` on changed Go files and use its standard tab indentation. Follow Go naming conventions. Keep internal identifiers unexported, and name platform-specific files with the appropriate suffix, such as `_windows.go`.

## Testing

Use Go’s standard `testing` package. Put tests in `*_test.go` files and name them after the behavior, such as `TestRapidTurns`. Set up deterministic game state when testing movement or collisions. The project has no configured coverage threshold. For window, input, or icon changes, build the executable and verify the affected behavior manually.

## Commits and Pull Requests

提交訊息使用簡短、具體的繁體中文，例如「修正快速連按時轉向遺失」。Pull Request 請用繁體中文說明行為變更與已執行的檢查，有相關 Issue 時附上連結。視窗或圖示有可見變更時附上截圖。
