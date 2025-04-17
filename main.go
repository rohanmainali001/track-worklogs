package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

var digits = map[rune][]string{
	'0': {" ███ ", "█   █", "█   █", "█   █", " ███ "},
	'1': {"  █  ", " ██  ", "  █  ", "  █  ", " ███ "},
	'2': {" ███ ", "    █", " ███ ", "█    ", "█████"},
	'3': {"████ ", "    █", " ███ ", "    █", "████ "},
	'4': {"█  █ ", "█  █ ", "█████", "   █ ", "   █ "},
	'5': {"█████", "█    ", "████ ", "    █", "████ "},
	'6': {" ███ ", "█    ", "████ ", "█   █", " ███ "},
	'7': {"█████", "   █ ", "  █  ", " █   ", " █   "},
	'8': {" ███ ", "█   █", " ███ ", "█   █", " ███ "},
	'9': {" ███ ", "█   █", " ████", "    █", " ███ "},
	':': {"     ", "  █  ", "     ", "  █  ", "     "},
}

type TaskEntry struct {
	Task     string
	Duration time.Duration
}

func clearScreen() {
	fmt.Print("\033[2J\033[H")
}

func renderTime(d time.Duration, paused bool) {
	clearScreen()
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	timeStr := fmt.Sprintf("%02d:%02d:%02d", h, m, s)

	rows := make([]string, 5)
	for _, ch := range timeStr {
		for i := 0; i < 5; i++ {
			rows[i] += digits[ch][i] + "  "
		}
	}
	for _, row := range rows {
		fmt.Println(row)
	}

	if paused {
		fmt.Println("\n⏸️  Paused - Press 'p' to resume | 'q' to end task")
	} else {
		fmt.Println("\n▶️  Tracking - Press 'p' to pause | 'q' to end task")
	}
}

func inputPrompt(prompt string) string {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')
	return strings.TrimSpace(text)
}

func writeMarkdown(project string, entries []TaskEntry) {
	year, month, day := time.Now().Date()
	filename := fmt.Sprintf("%04d-%02d-%02d_%s.md", year, month, day, project)

	// Build full path: ~/Desktop/rohan/league-rohan
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("❌ Could not determine home directory:", err)
		return
	}

	saveDir := filepath.Join(homeDir, "Desktop", "rohan", "league-rohan")
	err = os.MkdirAll(saveDir, os.ModePerm)
	if err != nil {
		fmt.Println("❌ Could not create directory:", err)
		return
	}

	fullPath := filepath.Join(saveDir, filename)
	file, err := os.Create(fullPath)
	if err != nil {
		fmt.Println("❌ Error writing Markdown:", err)
		return
	}
	defer file.Close()

	// Write Markdown content
	fmt.Fprintf(file, "---\ntags: [work-log, %s]\ndate: %04d-%02d-%02d\nproject: %s\n---\n\n",
		strings.ToLower(project), year, month, day, project)
	fmt.Fprintf(file, "# 📝 Work Log for %s (%04d-%02d-%02d)\n\n", project, year, month, day)

	for _, entry := range entries {
		fmt.Fprintf(file, "- **Task**: %s\n  - ⏱️ **Duration**: %s\n", entry.Task, entry.Duration.Round(time.Second))
	}

	fmt.Println("✅ Markdown log saved to", fullPath)
}

func runSession() (TaskEntry, bool, bool) {
	start := time.Now()
	elapsed := time.Duration(0)
	paused := false
	endTask := false
	quitApp := false

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT)

	go func() {
		reader := bufio.NewReader(os.Stdin)
		for {
			b, err := reader.ReadByte()
			if err != nil {
				continue
			}
			switch b {
			case 'p', 'P':
				paused = !paused
				if !paused {
					start = time.Now().Add(-elapsed)
				}
			case 'q', 'Q':
				quitApp = true
				endTask = true
				return
			}
		}
	}()

loop:
	for {
		select {
		case <-sigChan:
			endTask = true
			break loop
		default:
			if !paused {
				elapsed = time.Since(start)
			}
			renderTime(elapsed, paused)
			time.Sleep(1 * time.Second)

			if endTask || quitApp {
				break loop
			}
		}
	}

	fmt.Print("\n")
	task := inputPrompt("📝 What task did you just finish? ")
	return TaskEntry{Task: task, Duration: elapsed}, quitApp, true
}

func main() {
	projectFlag := flag.String("project", "League", "Name of the project")
	flag.Parse()
	project := *projectFlag

	var entries []TaskEntry

	for {
		entry, quit, valid := runSession()
		if valid {
			entries = append(entries, entry)
		}
		if quit {
			fmt.Println("👋 Quit early with 'q'. See you next time!")
		}

		answer := strings.ToLower(inputPrompt("✅ Done for the day? (yes/no): "))
		if answer == "yes" || answer == "y" {
			writeMarkdown(project, entries)
			fmt.Println("👋 Session complete. See you next time!")
			return
		}
	}
}
