package main

import (
	"flag"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const VERSION string = "v0.0.1"

type PomoConfig struct {
	Blocks        int
	FocusDuration int
	BreakDuration int
	Debug         bool
}

type PomoState struct {
	IsBreak         bool
	CurrentBlock    int
	CurrentBreak    int
	RemainingBlocks int
	RemainingBreaks int
}

func main() {
	fmt.Println("")
	PrintfAtLevel(1, fmt.Sprintf("* pomo %s *\n", VERSION))

	// Get input from flags
	blocks := flag.Int("x", 3, "x: the amount of focus blocks.")
	focusDuration := flag.Int("f", 25, "f: the focus duration per block.")
	breakDuration := flag.Int("b", 5, "b: the break duration per block.")
	debug := flag.Bool("debug", false, "debug: turns time inputs into seconds instead of minutes.")
	flag.Parse()

	// Config
	config := PomoConfig{
		Blocks:        *blocks,
		FocusDuration: *focusDuration,
		BreakDuration: *breakDuration,
		Debug:         *debug,
	}
	TimeFactor := time.Minute
	if config.Debug {
		TimeFactor = time.Second
		PrintfAtLevel(1, fmt.Sprintf("DEBUG: %v\n", config.Debug))
	}

	// Show start time when pomo was initialized and run
	start := time.Now()
	PrintfAtLevel(2, fmt.Sprintf("started pomo at: %s\n", start.Format(time.DateTime)))

	// State
	state := InitState(config)

	// TODO: Fix correct plural usage for blocks, minutes, breaks, etc.
	blockPluralSafe := "block"
	if state.RemainingBlocks > 1 {
		blockPluralSafe = "blocks"
	}

	timeUntilCurrentBlockIsDone := start.Add(time.Duration(config.FocusDuration) * TimeFactor)
	if config.Blocks > 1 {
		timeUntilCurrentBlockIsDone = timeUntilCurrentBlockIsDone.Add(time.Duration(config.BreakDuration) * TimeFactor)
	}

	totalProjectedTime := (config.Blocks * config.FocusDuration) + ((config.Blocks - 1) * config.BreakDuration)
	totalProjectedTimeWhenDone := start.Add(time.Duration(totalProjectedTime) * TimeFactor)

	// Show information about current running configuration
	configStr := fmt.Sprintf("config: %v %s of %v minutes focus", config.Blocks, blockPluralSafe, config.FocusDuration)
	if config.Blocks > 1 {
		configStr = fmt.Sprintf("%s and %v minutes break.\n", configStr, config.BreakDuration)
	} else {
		configStr = fmt.Sprintf("%s.\n", configStr)
	}
	PrintfAtLevel(2, configStr)
	fmt.Println("")

	// Show the remaining blocks and breaks in the current running pomo setup
	if state.RemainingBlocks > 1 {
		PrintfAtLevel(2, fmt.Sprintf("first block is done at: %s\n", timeUntilCurrentBlockIsDone.Format(time.TimeOnly)))
	}
	if state.RemainingBlocks > 0 || state.RemainingBreaks > 0 {
		PrintfAtLevel(2, fmt.Sprintf("remaining: %v blocks, %v breaks.\n", state.RemainingBlocks, state.RemainingBreaks))
	}
	PrintfAtLevel(2, fmt.Sprintf("pomo session is done at: %v.\n", totalProjectedTimeWhenDone.Format(time.TimeOnly)))
	fmt.Println("")

	// Start a separate goroutine that keeps track of the timer
	// Start running the main timer functionality
	// This current process updates a stdout line with the time remaining of the current block

	// Main runner logic
	for config.Blocks >= state.CurrentBlock {
		if !state.IsBreak {
			RunTimer(config, state, TimeFactor)
			state = FinishBlock(state)
			fmt.Println("")
			if state.CurrentBlock == config.Blocks {
				break
			}
		} else {
			RunTimer(config, state, TimeFactor)
			state = FinishBreak(state)
			fmt.Println("")
		}
	}
	PrintfAtLevel(1, "pomo finished.")

	// TODO: If user inputs "s", then prompt to skip current timer.
	// Also, provide option to pause. Maybe next version? Start simple.

	// When a block is done or a break is done, require user input to start the next block. (y/n)
	// Keep track of extra time passing while awaiting user input

	// Update projected time spent with the extra time spent waiting
}

func InitState(config PomoConfig) PomoState {
	return PomoState{
		IsBreak:         false,
		CurrentBlock:    1,
		CurrentBreak:    0,
		RemainingBlocks: config.Blocks - 1,
		RemainingBreaks: config.Blocks - 1,
	}
}

func PrintfAtLevel(level int, string string) error {
	indent := strings.Repeat("  ", level)
	fmt.Printf("%s%s", indent, string)
	return nil
}

func FormatDuration(duration time.Duration) string {
	h := duration / time.Hour
	m := (duration / time.Minute) % 60
	s := (duration / time.Second) % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

func RunTimer(config PomoConfig, state PomoState, timeFactor time.Duration) error {
	start := time.Now()
	timer := time.NewTimer(time.Duration(config.FocusDuration) * timeFactor)
	ticker := time.NewTicker(time.Second)
	done := make(chan bool)

	durationRaw := time.Until(start.Add(time.Duration(config.FocusDuration) * timeFactor))
	statusStr := fmt.Sprintf("block %v", state.CurrentBlock)
	if state.IsBreak {
		statusStr = fmt.Sprintf("break %v", state.CurrentBreak)
	}

	go func() {
		for {
			select {
			case <-done:
				return
			case t := <-ticker.C:
				fmt.Printf("\r    %s: %s", statusStr, FormatDuration(durationRaw-t.Sub(start)))
			}
		}
	}()

	<-timer.C
	ticker.Stop()
	done <- true
	msg := fmt.Sprintf("%s done.", statusStr)
	cmdNotify := exec.Command("notify-send", "-u", "critical", "-t", "0", msg)
	cmdNotify.Run()
	fmt.Println("")
	return nil
}

func FinishBlock(state PomoState) PomoState {
	return PomoState{
		IsBreak:         true,
		CurrentBlock:    state.CurrentBlock,
		CurrentBreak:    state.CurrentBreak + 1,
		RemainingBlocks: state.RemainingBlocks,
		RemainingBreaks: state.RemainingBreaks - 1,
	}
}

func FinishBreak(state PomoState) PomoState {
	return PomoState{
		IsBreak:         false,
		CurrentBlock:    state.CurrentBlock + 1,
		CurrentBreak:    state.CurrentBreak,
		RemainingBlocks: state.RemainingBlocks - 1,
		RemainingBreaks: state.RemainingBreaks,
	}
}
