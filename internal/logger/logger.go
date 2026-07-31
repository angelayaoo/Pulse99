package logger

import (
	"fmt"
	"time"
)

const (
	reset  = "\033[0m"
	bold   = "\033[1m"
	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
	cyan   = "\033[36m"
	gray   = "\033[90m"
)

func Banner() {
	fmt.Printf("%spulse99%s %suptime daemon v2.0.0%s %s[starting]%s\n\n",
		cyan, reset, bold, reset, gray, reset)
}

func ScanSweepStart(iteration int, targetCount int) {
	ts := time.Now().Format("15:04:05")
	fmt.Printf("%s[%s]%s %scheck%s sweep %s#%d%s %s(%d targets)%s\n",
		gray, ts, reset, green, reset, bold, iteration, reset, gray, targetCount, reset)
}

func NodeStable(name string, latency time.Duration, statusCode int) {
	ts := time.Now().Format("15:04:05")
	lat := fmt.Sprintf("%dms", latency.Milliseconds())
	fmt.Printf("%s[%s]%s %sOK    %s %-20s %s%s%s %s%d%s\n",
		gray, ts, reset, green, reset, name,
		green, lat, reset, green, statusCode, reset,
	)
}

func NodeWarning(name string, failures int, threshold int, reason string) {
	ts := time.Now().Format("15:04:05")
	count := fmt.Sprintf("%d/%d", failures, threshold)
	fmt.Printf("%s[%s]%s %sWARN  %s %-20s %s%s%s %s%s%s\n",
		gray, ts, reset, yellow, reset, name,
		yellow, count, reset, yellow, reason, reset,
	)
}

func NodeCritical(name string, reason string) {
	ts := time.Now().Format("15:04:05")
	fmt.Printf("%s[%s]%s %sDOWN  %s %-20s %s%s%s\n",
		gray, ts, reset, red, reset, name,
		red, reason, reset,
	)
}

func NodeRecovered(name string, latency time.Duration) {
	ts := time.Now().Format("15:04:05")
	lat := fmt.Sprintf("%dms", latency.Milliseconds())
	fmt.Printf("%s[%s]%s %sUP    %s %-20s %s%s%s\n",
		gray, ts, reset, green, reset, name,
		green, lat, reset,
	)
}

func SweepSummary(up int, down int, unstable int) {
	ts := time.Now().Format("15:04:05")
	fmt.Printf("%s[%s]%s %ssweep done%s %s|%s %s%d up%s  %s%d down%s  %s%d degraded%s\n",
		gray, ts, reset, cyan, reset,
		cyan, reset,
		green, up, reset,
		red, down, reset,
		yellow, unstable, reset,
	)
	fmt.Println()
}

func ConfigLoaded(targetCount int, interval int, threshold int) {
	fmt.Printf("%sconfig:%s %d targets | scan interval %ds | failure threshold %d\n\n",
		gray, reset, targetCount, interval, threshold)
}

func Shutdown() {
	fmt.Printf("\n%spulse99: shutdown complete%s\n", cyan, reset)
}

func AlertSuppressed(name, kind string) {
	ts := time.Now().Format("15:04:05")
	fmt.Printf("%s[%s]%s %salert suppressed%s %-20s %s(%s response, cooldown active)%s\n",
		gray, ts, reset, gray, reset, name,
		gray, kind, reset,
	)
}

func NotifyRetry(channel string, attempt int, err error) {
	ts := time.Now().Format("15:04:05")
	fmt.Printf("%s[%s]%s %sretry%s %s #%d: %s%v%s\n",
		gray, ts, reset, yellow, reset, channel,
		attempt, gray, err, reset,
	)
}

func NotifyFailed(channel, target string, err error) {
	ts := time.Now().Format("15:04:05")
	fmt.Printf("%s[%s]%s %snotify failed%s %s -> %s%s%s: %s%v%s\n",
		gray, ts, reset, red, reset,
		channel,
		red, target, reset,
		gray, err, reset,
	)
}

func DashboardStarted(port int) {
	fmt.Printf("%sdashboard:%s http://0.0.0.0:%d\n",
		gray, reset, port,
	)
}
