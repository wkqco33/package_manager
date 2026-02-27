package ui

import (
	"fmt"
	"io"
	"strings"
	"time"
)

// ProgressBar 데이터 중심 구조체 (진행률 표시)
type ProgressBar struct {
	Total   int64
	Current int64
	Width   int
	Prefix  string
}

// NewProgressBar는 초기화된 상태의 ProgressBar를 반환합니다.
func NewProgressBar(total int64, width int, prefix string) *ProgressBar {
	return &ProgressBar{
		Total:   total,
		Current: 0,
		Width:   width,
		Prefix:  prefix,
	}
}

// Update는 진행 상황 데이터를 갱신하고 화면을 다시 그립니다.
func (p *ProgressBar) Update(current int64) {
	p.Current = current
	if p.Total > 0 && p.Current > p.Total {
		p.Current = p.Total
	}
	p.Render()
}

// Increment는 1만큼 진행도를 올립니다.
func (p *ProgressBar) Increment() {
	p.Update(p.Current + 1)
}

// Finish는 100% 진행으로 설정하고 줄을 바꿉니다.
func (p *ProgressBar) Finish() {
	if p.Total > 0 {
		p.Update(p.Total)
	}
	fmt.Println()
}

// Render는 현재 상태 데이터를 토대로 ANSI 문자를 이용해 화면에 출력합니다.
func (p *ProgressBar) Render() {
	if p.Total <= 0 {
		// 진행률을 모를 때의 출력 방식 (단순 바이트 단위 등)
		fmt.Printf("\r\033[K%s %d bytes", Info(p.Prefix), p.Current)
		return
	}

	percent := float64(p.Current) / float64(p.Total) * 100
	completedWidth := int(float64(p.Width) * (float64(p.Current) / float64(p.Total)))

	completed := strings.Repeat("█", completedWidth)
	empty := strings.Repeat("-", p.Width-completedWidth)

	// \r: 커서를 줄 맨 앞으로 이동
	// \033[K: 커서 위치부터 줄 끝까지 지움
	fmt.Printf("\r\033[K%s [%s%s] %3.0f%% (%d/%d)",
		Info(p.Prefix),
		Highlight(completed),
		Gray+empty+Reset,
		percent,
		p.Current,
		p.Total,
	)
}

// Spinner 애니메이션 프레임 데이터 (상수형 배열)
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// Spinner 구조체 (다운로드 등 진행도를 알 수 없을 때 사용)
type Spinner struct {
	CurrentIdx int
	Prefix     string
	StopChan   chan struct{}
}

// NewSpinner는 새로운 Spinner 구조체를 생성합니다.
func NewSpinner(prefix string) *Spinner {
	return &Spinner{
		CurrentIdx: 0,
		Prefix:     prefix,
		StopChan:   make(chan struct{}),
	}
}

// Start는 고루틴을 통해 스피너를 출력합니다. 채널 데이터로 종료 흐름을 제어합니다.
func (s *Spinner) Start() {
	go func() {
		for {
			select {
			case <-s.StopChan:
				return
			default:
				fmt.Printf("\r\033[K%s %s", Highlight(spinnerFrames[s.CurrentIdx]), s.Prefix)
				s.CurrentIdx = (s.CurrentIdx + 1) % len(spinnerFrames)
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()
}

// Stop은 채널을 닫아서 스피너를 멈추고 현재 줄을 지웁니다.
func (s *Spinner) Stop() {
	close(s.StopChan)
	fmt.Print("\r\033[K")
}

// ProgressReader는 io.Reader를 감싸서 읽을 때마다 ProgressBar를 업데이트하는 래퍼입니다.
type ProgressReader struct {
	Reader io.ReadCloser
	Bar    *ProgressBar
}

func (pr *ProgressReader) Read(p []byte) (n int, err error) {
	n, err = pr.Reader.Read(p)
	if n > 0 {
		pr.Bar.Update(pr.Bar.Current + int64(n))
	}
	return
}

func (pr *ProgressReader) Close() error {
	pr.Bar.Finish()
	return pr.Reader.Close()
}
