package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"github.com/coroot/logparser"
	"io"
	"os"
	"sort"
	"strings"
	"time"
)

func main() {
	screenWidth := flag.Int("w", 120, "terminal width")
	flag.Parse()

	reader := bufio.NewReader(os.Stdin)
	ch := make(chan logparser.LogEntry)
	parser := logparser.NewParser(ch, nil)
	t := time.Now()
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if !errors.Is(err, io.EOF) {
				fmt.Println(err)
			}
			break
		}
		ch <- logparser.LogEntry{Content: strings.TrimSuffix(line, "\n"), Level: logparser.LevelUnknown}
	}
	d := time.Since(t)
	defer parser.Stop()

	counters := parser.GetCounters()

	order(counters)

	output(counters, *screenWidth, d)
}

func order(counters []logparser.LogCounter) {
	sort.Slice(counters, func(i, j int) bool {
		ci, cj := counters[i], counters[j]
		if ci.Level == cj.Level {
			return ci.Messages > cj.Messages
		}
		return ci.Level < cj.Level
	})
}

func output(counters []logparser.LogCounter, screenWidth int, duration time.Duration) {
	grandTotal, total, max := 0, 0, 0
	for _, c := range counters {
		grandTotal += c.Messages
		if c.Sample == "" {
			continue
		}
		total += c.Messages
		if c.Messages > max {
			max = c.Messages
		}
	}
	barWidth := 30
	lineWidth := screenWidth - barWidth
	for _, c := range counters {
		if c.Sample == "" {
			continue
		}
		w := c.Messages * barWidth / max
		bar := strings.Repeat("▇", w+1) + strings.Repeat(" ", barWidth-w)
		prefix := colorize(c.Level, "%s %d (%.2f%%)\t", bar, c.Messages, float64(c.Messages*100)/float64(total))
		sample := ""
		for i, line := range strings.Split(c.Sample, "\n") {
			if len(line) > lineWidth {
				line = line[:lineWidth] + "..."
			}
			sample += line + "\n" + strings.Repeat(" ", len(prefix))
			if i > 10 {
				sample += "...\n"
				break
			}
		}
		sample = strings.TrimRight(sample, "\n ")
		fmt.Printf("%s%s\n", prefix, sample)
	}

	byLevel := map[logparser.Level]int{}
	for _, c := range counters {
		byLevel[c.Level] += c.Messages
	}
	fmt.Println()
	fmt.Printf("%d messages processed in %.3f seconds:\n", grandTotal, duration.Seconds())
	for l, c := range byLevel {
		fmt.Printf("  %s: %d\n", l, c)
	}
	fmt.Println()
}

func colorize(level logparser.Level, format string, a ...interface{}) string {
	c := "\033[37m" // grey
	switch level {
	case logparser.LevelCritical, logparser.LevelError:
		c = "\033[31m" // red
	case logparser.LevelWarning:
		c = "\033[33m" // yellow
	case logparser.LevelInfo:
		c = "\033[32m" // green
	}
	return fmt.Sprintf(c+format+"\033[0m", a...)
}
