# MySnake · 貪食蛇

<img src="./snake.png" alt="一隻盤起身體、露出笑容的小綠蛇" width="180">

用 Go 寫的原生 Windows 貪食蛇。控制小蛇吃食物、拉長身體，試著在撞牆或撞到自己之前刷新最高分。

## 開始遊戲

需要 **Windows 與 Go 1.24 以上版本**。在 PowerShell 執行：

```powershell
git clone https://github.com/chuhan-cheng/mysnake.git
cd mysnake
go run .
```

也可以編譯成獨立的 Windows 執行檔，再從檔案總管雙擊開啟：

```powershell
go build -ldflags=-H=windowsgui -o mysnake.exe .
```

`-ldflags` 會把 `-H=windowsgui` 傳給 Go 連結器，將執行檔標記為 Windows GUI 應用程式。從檔案總管雙擊時，Windows 會直接開啟遊戲視窗，省去額外的主控台視窗。遊戲視窗本身由程式透過 Win32 API 建立。

## 操作方式

| 按鍵 | 功能 |
| --- | --- |
| 方向鍵或 `W`、`A`、`S`、`D` | 開始遊戲並控制方向 |
| 空白鍵 | 暫停或繼續 |
| `R` 或 `Enter` | 重新開始 |
| `H` | 開啟或關閉歷史紀錄 |
| `Esc` | 從歷史紀錄返回棋盤 |
| `M` | 開啟或關閉音效 |

棋盤有 24 × 20 格。每吃到一份食物就加一分，身體也會變長一格。快速按下兩個有效方向時，小蛇會在接下來兩步依序轉向。填滿整個棋盤即獲勝。

## 遊戲紀錄

歷史紀錄依開局時間由新到舊排列，顯示日期時間、遊玩時長、分數和結果。從第一次移動開始計時，暫停期間不計入。遊玩途中開啟歷史會自動暫停，返回棋盤後按空白鍵繼續。

遊戲將紀錄與音效設定儲存在 `%APPDATA%\MySnake`。正常關閉視窗或重新開始時，進行中的對局會標記為「中途退出」。若遊戲意外關閉，下次啟動會讀取最近一次成功存檔。最高分以正常結束的對局計算。

## 開發與貢獻

遊戲規則在 [main.go](./main.go)，Windows 視窗、鍵盤輸入與繪圖在 [window_windows.go](./window_windows.go)。

```powershell
go test ./...
go vet ./...
```

提交變更前，請參閱 [貢獻指南](./AGENTS.md)。
