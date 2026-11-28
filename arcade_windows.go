package main

import (
	"fmt"
	"time"
	"unsafe"
)

const (
	foodEffect   = 550 * time.Millisecond
	impactEffect = 260 * time.Millisecond
	ink          = 0x382043
	paper        = 0xF7ECFA
	white        = 0xFFFFFF
	green        = 0x65E44D
	pink         = 0x8258FF
	cream        = 0xBCEDFF
)

type control struct {
	area    rect
	text    string
	key     uintptr
	enabled bool
}

func pageSize(bounds rect) int {
	return max(1, int((bounds.bottom-230)/40))
}

func pageCount(bounds rect) int {
	size := pageSize(bounds)
	return max(1, (len(history)+size-1)/size)
}

func controls(bounds rect) []control {
	right := bounds.right - 24
	if showHistory {
		y := bounds.bottom - 70
		page := min(historyPage, pageCount(bounds)-1)
		return []control{
			{rect{right - 110, 22, right, 58}, "返回  Esc", 'H', true},
			{rect{24, y, 124, y + 36}, "上一頁", 0x21, page > 0},
			{rect{right - 100, y, right, y + 36}, "下一頁", 0x22, page+1 < pageCount(bounds)},
		}
	}
	audio := "音效 開"
	if muted {
		audio = "音效 關"
	}
	buttons := []control{
		{rect{right - 198, 22, right - 108, 58}, audio, 'M', true},
		{rect{right - 98, 22, right, 58}, "歷史紀錄", 'H', true},
	}
	if current.ended {
		x, y := bounds.right/2, bounds.bottom/2+61
		buttons = append(buttons,
			control{rect{x - 145, y, x - 5, y + 37}, "再玩一次", 'R', true},
			control{rect{x + 5, y, x + 145, y + 37}, "本機紀錄", 'H', true},
		)
	}
	return buttons
}

func handleKey(window, key uintptr, now time.Time) {
	switch key {
	case 'M':
		toggleMute()
		return
	case 0x71: // F2
		playPickup()
		return
	case 'H', 0x1B:
		if showHistory {
			showHistory = false
		} else if key == 'H' {
			if current.started && !current.paused && !current.ended {
				pauseRound(now)
			}
			showHistory = true
			historyPage = 0
		}
		return
	}
	if showHistory {
		switch key {
		case 0x21, 0x25:
			historyPage = max(0, historyPage-1)
		case 0x22, 0x27:
			var bounds rect
			getClientRect.Call(window, uintptr(unsafe.Pointer(&bounds)))
			historyPage = min(pageCount(bounds)-1, historyPage+1)
		}
		return
	}
	switch key {
	case 0x25, 'A':
		current.turn(cell{-1, 0})
	case 0x26, 'W':
		current.turn(cell{0, -1})
	case 0x27, 'D':
		current.turn(cell{1, 0})
	case 0x28, 'S':
		current.turn(cell{0, 1})
	case ' ':
		if current.started && !current.ended {
			if current.paused {
				current.paused = false
				clock.resumed = now
			} else {
				pauseRound(now)
			}
		}
	case 'R', 0x0D:
		restart(now)
	}
}

func drawScene(dc uintptr, bounds rect) {
	fill(dc, bounds, paper)
	if bounds.right < 490 || bounds.bottom < 490 {
		label(dc, bounds, "請放大視窗", 20, ink)
		return
	}
	if showHistory {
		drawHistory(dc, bounds)
	} else {
		drawArcade(dc, bounds, time.Now())
	}
	for _, button := range controls(bounds) {
		background, foreground := uintptr(ink), uintptr(white)
		if !button.enabled {
			background, foreground = 0xE6D7EB, 0x785A80
		}
		rounded(dc, button.area, background, 14)
		label(dc, button.area, button.text, 15, foreground)
	}
	if saveError {
		label(dc, rect{0, bounds.bottom - 26, bounds.right, bounds.bottom}, "無法儲存紀錄，請確認資料夾可寫入或關閉其他遊戲視窗", 14, 0x42429C)
	}
}

func drawArcade(dc uintptr, bounds rect, now time.Time) {
	w, h := bounds.right, bounds.bottom
	label(dc, rect{24, 13, 201, 66}, "貪食蛇", 31, ink)
	for i, text := range []string{
		fmt.Sprintf("分數  %d", current.score),
		fmt.Sprintf("最高分  %d", current.best),
		fmt.Sprintf("速度  Lv.%d", speedLevel(current.score)),
		"時間  " + durationText(clock.elapsed(now)),
	} {
		left := 24 + int32(i)*(w-48)/4
		card := rect{left, 72, left + (w-48)/4 - 6, 111}
		rounded(dc, card, []uintptr{green, cream, 0xF2D6C7, white}[i], 14)
		label(dc, card, text, 16, ink)
	}
	size := min((w-56)/columns, (h-209)/rows)
	x, y := (w-size*columns)/2, 120+(h-209-size*rows)/2
	board := rect{x, y, x + size*columns, y + size*rows}
	border := uintptr(0xAC79AE)
	if now.Sub(impactAt) < impactEffect {
		border = pink
	}
	rounded(dc, rect{board.left - 5, board.top - 5, board.right + 5, board.bottom + 5}, border, 16)
	for row := range int32(rows) {
		for col := range int32(columns) {
			shade := uintptr(0x574257)
			if (row+col)%2 == 0 {
				shade = 0x614C61
			}
			left, top := x+col*size, y+row*size
			fill(dc, rect{left, top, left + size, top + size}, shade)
		}
	}
	box := func(p cell, inset int32) rect {
		left, top := x+int32(p.x)*size, y+int32(p.y)*size
		return rect{left + inset, top + inset, left + size - inset, top + size - inset}
	}
	if !current.won {
		drawFruit(dc, box(current.food, max(1, size/6)), current.fruit)
	}
	for i := len(current.snake) - 1; i >= 0; i-- {
		part := box(current.snake[i], 1)
		color := uintptr(0x43BA79)
		if i == 0 {
			color = green
		}
		rounded(dc, part, color, size/2)
	}
	head := current.snake[0]
	cx, cy := x+int32(head.x)*size+size/2, y+int32(head.y)*size+size/2
	dx, dy := int32(current.direction.x), int32(current.direction.y)
	for _, side := range []int32{-1, 1} {
		ex, ey := cx+(dx+side*dy)*size/5, cy+(dy+side*dx)*size/5
		radius := max(1, size/9)
		rounded(dc, rect{ex - radius, ey - radius, ex + radius + 1, ey + radius + 1}, ink, radius*2)
	}
	if elapsed := now.Sub(foodAt); elapsed < foodEffect && !current.ended {
		rise := int32(20 * elapsed / foodEffect)
		top := max(board.top, cy-32-rise)
		area := rect{cx - 23, top, cx + 23, top + 26}
		rounded(dc, area, cream, 13)
		label(dc, area, fmt.Sprintf("+%d", foodBonus), 18, ink)
	}
	status := current.status()
	if current.ended {
		mid := h / 2
		rounded(dc, rect{w/2 - 166, mid - 83, w/2 + 166, mid + 111}, white, 24)
		title := "遊戲結束"
		if current.won {
			title = "填滿棋盤！"
		}
		if newBest {
			title = "刷新最高分！"
		}
		label(dc, rect{0, mid - 72, w, mid - 9}, title, 31, ink)
		label(dc, rect{0, mid - 6, w, mid + 40}, fmt.Sprintf("分數  %d     時間  %s", current.score, durationText(clock.elapsed(now))), 20, ink)
	}
	label(dc, rect{0, h - 81, w, h - 52}, status, 17, ink)
	if len(history) > 0 {
		recent := history[0]
		label(dc, rect{0, h - 51, w, h - 26}, fmt.Sprintf("最近一局   %s   %d 分   %s", recent.Started.Local().Format("01/02 15:04"), recent.Score, durationText(recent.Duration)), 14, ink)
	}
	if !saveError {
		label(dc, rect{0, h - 26, w, h}, "圓形 +1　菱形 +5　星形 +10", 14, ink)
	}
}

func drawHistory(dc uintptr, bounds rect) {
	w, h := bounds.right, bounds.bottom
	label(dc, rect{24, 15, 210, 65}, "歷史紀錄", 30, ink)
	if len(history) == 0 {
		label(dc, rect{0, 92, w, h - 90}, "完成第一局後，紀錄會顯示在這裡", 20, ink)
		return
	}
	split := []int32{24, w * 40 / 100, w * 59 / 100, w * 72 / 100, w - 24}
	row := func(y int32, values []string, color uintptr) {
		for i, value := range values {
			label(dc, rect{split[i], y, split[i+1], y + 38}, value, 16, color)
		}
	}
	rounded(dc, rect{24, 78, w - 24, 116}, ink, 14)
	row(78, []string{"開局日期時間", "遊玩時長", "分數", "結果"}, white)
	total := pageCount(bounds)
	page := min(historyPage, total-1)
	size := pageSize(bounds)
	for i, record := range history[page*size : min((page+1)*size, len(history))] {
		y := 120 + int32(i)*40
		rounded(dc, rect{24, y, w - 24, y + 38}, white, 12)
		row(y, []string{record.Started.Local().Format("2006/01/02 15:04:05"), durationText(record.Duration), fmt.Sprint(record.Score), record.Result}, ink)
	}
	label(dc, rect{0, h - 70, w, h - 34}, fmt.Sprintf("第 %d / %d 頁    共 %d 局", page+1, total, len(history)), 16, ink)
}

func durationText(duration time.Duration) string {
	seconds := int64(duration / time.Second)
	return fmt.Sprintf("%02d:%02d", seconds/60, seconds%60)
}

func drawFruit(dc uintptr, area rect, kind fruit) {
	if kind == ordinary {
		rounded(dc, area, pink, area.right-area.left)
		return
	}
	x, y := (area.left+area.right)/2, (area.top+area.bottom)/2
	d := max(1, (area.right-area.left)/5)
	color := uintptr(0x54CCFF)
	points := []point{{x, area.top}, {area.right, y}, {x, area.bottom}, {area.left, y}}
	if kind == legendary {
		color = 0xFFE393
		points = []point{{x, area.top}, {x + d, y - d}, {area.right, y}, {x + d, y + d},
			{x, area.bottom}, {x - d, y + d}, {area.left, y}, {x - d, y - d}}
	}
	brush, _, _ := createBrush.Call(color)
	defer deleteObject.Call(brush)
	old, _, _ := selectObject.Call(dc, brush)
	defer selectObject.Call(dc, old)
	gdi.NewProc("Polygon").Call(dc, uintptr(unsafe.Pointer(&points[0])), uintptr(len(points)))
}

func rounded(dc uintptr, area rect, color uintptr, radius int32) {
	brush, _, _ := createBrush.Call(color)
	defer deleteObject.Call(brush)
	pen, _, _ := gdi.NewProc("GetStockObject").Call(8) // NULL_PEN
	oldPen, _, _ := selectObject.Call(dc, pen)
	defer selectObject.Call(dc, oldPen)
	oldBrush, _, _ := selectObject.Call(dc, brush)
	defer selectObject.Call(dc, oldBrush)
	gdi.NewProc("RoundRect").Call(dc, uintptr(area.left), uintptr(area.top), uintptr(area.right), uintptr(area.bottom), uintptr(radius), uintptr(radius))
}
