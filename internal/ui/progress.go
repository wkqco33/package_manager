package ui

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

// stdoutMu는 병렬 설치 시 터미널 출력이 섞이지 않도록 직렬화합니다.
var stdoutMu sync.Mutex

const renderInterval = 100 * time.Millisecond

// ProgressBar 데이터 중심 구조체 (진행률 표시)
type ProgressBar struct {
	Total      int64
	Current    int64
	Width      int
	Prefix     string
	lastRender time.Time
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

// Update는 진행 상황 데이터를 갱신하고, renderInterval마다 화면을 다시 그립니다.
func (p *ProgressBar) Update(current int64) {
	p.Current = current
	if p.Total > 0 && p.Current > p.Total {
		p.Current = p.Total
	}
	if time.Since(p.lastRender) >= renderInterval {
		p.Render()
		p.lastRender = time.Now()
	}
}

// Increment는 1만큼 진행도를 올립니다.
func (p *ProgressBar) Increment() {
	p.Update(p.Current + 1)
}

// Finish는 100% 진행으로 설정하고 줄을 바꿉니다. stdoutMu를 보유한 채 렌더와 개행을 원자적으로 처리합니다.
func (p *ProgressBar) Finish() {
	stdoutMu.Lock()
	defer stdoutMu.Unlock()
	if p.Total > 0 {
		p.Current = p.Total
	}
	p.render()
	fmt.Println()
}

// Render는 stdoutMu를 잡고 현재 상태를 화면에 출력합니다.
func (p *ProgressBar) Render() {
	stdoutMu.Lock()
	defer stdoutMu.Unlock()
	p.render()
}

// render는 mutex 없이 실제 출력을 수행합니다. 반드시 stdoutMu를 보유한 상태에서 호출하세요.
func (p *ProgressBar) render() {
	if p.Total <= 0 {
		// 진행률을 모를 때의 출력 방식 (단순 바이트 단위 등)
		fmt.Printf("\r\033[K%s %d bytes", Info(p.Prefix), p.Current)
		return
	}

	percent := float64(p.Current) / float64(p.Total) * 100
	completedWidth := int(float64(p.Width) * (float64(p.Current) / float64(p.Total)))

	// 진행 바 문자: ▰(채움), ▱(비움)
	completed := strings.Repeat("▰", completedWidth)
	empty := strings.Repeat("▱", p.Width-completedWidth)

	// \r: 커서를 줄 맨 앞으로 이동
	// \033[K: 커서 위치부터 줄 끝까지 지움
	fmt.Printf("\r\033[K%s  %s%s%s %3.0f%% %s",
		Label(p.Prefix),
		Highlight(completed),
		Muted(empty),
		Reset,
		percent,
		Muted(fmt.Sprintf("(%d/%d)", p.Current, p.Total)),
	)
}

// Spinner 애니메이션 프레임
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// Spinner 구조체 (다운로드 등 진행도를 알 수 없을 때 사용)
type Spinner struct {
	CurrentIdx  int
	Prefix      string
	StopChan    chan struct{}
	doneChan    chan struct{}
	startedChan chan struct{}
	startOnce   sync.Once
	stopOnce    sync.Once
}

// NewSpinner는 새로운 Spinner 구조체를 생성합니다.
func NewSpinner(prefix string) *Spinner {
	return &Spinner{
		CurrentIdx:  0,
		Prefix:      prefix,
		StopChan:    make(chan struct{}),
		doneChan:    make(chan struct{}),
		startedChan: make(chan struct{}),
	}
}

// Start는 고루틴을 통해 스피너를 출력합니다. 채널 데이터로 종료 흐름을 제어합니다.
func (s *Spinner) Start() {
	s.startOnce.Do(func() {
		select {
		case <-s.StopChan:
			close(s.startedChan)
			close(s.doneChan)
			return
		default:
		}
		close(s.startedChan)
		go func() {
			defer close(s.doneChan)
			ticker := time.NewTicker(80 * time.Millisecond)
			defer ticker.Stop()

			for {
				select {
				case <-s.StopChan:
					return
				default:
				}

				stdoutMu.Lock()
				fmt.Printf("\r\033[K%s %s", Accent(spinnerFrames[s.CurrentIdx]), Muted(s.Prefix))
				stdoutMu.Unlock()
				s.CurrentIdx = (s.CurrentIdx + 1) % len(spinnerFrames)

				select {
				case <-s.StopChan:
					return
				case <-ticker.C:
				}
			}
		}()
	})
}

// Stop은 채널을 닫아서 스피너를 멈추고 현재 줄을 지웁니다.
func (s *Spinner) Stop() {
	s.stopOnce.Do(func() {
		close(s.StopChan)
		select {
		case <-s.startedChan:
			<-s.doneChan
		default:
			return
		}
		stdoutMu.Lock()
		fmt.Print("\r\033[K")
		stdoutMu.Unlock()
	})
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
