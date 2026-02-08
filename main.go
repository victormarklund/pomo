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
	// Get input from flags
	blocks := flag.Int("x", 3, "x: the amount of focus blocks.")
	focusDuration := flag.Int("f", 25, "f: the focus duration per block.")
	breakDuration := flag.Int("b", 5, "b: the break duration per block.")
	debug := flag.Bool("debug", false, "debug: turns time inputs into seconds instead of minutes.")
	version := flag.Bool("v", false, "v: prints the current version of pomo.")
	flag.Parse()

	// Show version if -v was passed
	if *version {
		fmt.Println("pomo", VERSION)
		return
	}

	PrintPomoASCII()
	fmt.Println("")

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
			RunTimer(config.FocusDuration, state, TimeFactor)
			state = FinishBlock(state)
			fmt.Println("")
			if state.CurrentBlock == config.Blocks {
				break
			}
		} else {
			RunTimer(config.BreakDuration, state, TimeFactor)
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

func RunTimer(duration int, state PomoState, timeFactor time.Duration) error {
	start := time.Now()
	timer := time.NewTimer(time.Duration(duration) * timeFactor)
	ticker := time.NewTicker(time.Second)
	done := make(chan bool)

	durationRaw := time.Until(start.Add(time.Duration(duration) * timeFactor))
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
	msg := fmt.Sprintf("pomo: %s done.", statusStr)
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

func PrintPomoASCII() {
	fmt.Println("[0m[38;2;173;86;66m:[0m[38;2;173;88;65mc[0m[38;2;173;90;64mc[0m[38;2;173;92;62mc[0m[38;2;173;94;61mc[0m[38;2;173;96;61mc[0m[38;2;173;98;59mc[0m[38;2;173;100;58mc[0m[38;2;174;103;57mc[0m[38;2;159;96;52m:[0m[38;2;97;59;31m'[0m[38;2;10;10;10m [0m[38;2;0;0;0m [0m[38;2;0;0;0m [0m[38;2;10;10;10m [0m[38;2;97;65;28m'[0m[38;2;160;109;45mc[0m[38;2;203;141;55md[0m[38;2;218;154;58mx[0m[38;2;217;156;57mx[0m[38;2;218;159;56mx[0m[38;2;218;162;54mx[0m[38;2;218;164;53mx[0m[38;2;203;156;49mx[0m[38;2;161;125;37ml[0m[38;2;77;60;17m.[0m[38;2;0;0;0m [0m[38;2;0;0;0m [0m[38;2;0;0;0m [0m[38;2;41;35;10m.[0m[38;2;164;146;44mo[0m[38;2;160;147;47mo[0m[38;2;155;148;50mo[0m[38;2;151;150;53mo[0m[38;2;135;138;51ml[0m[38;2;1;1;0m [0m[38;2;0;0;0m [0m[38;2;0;0;0m [0m[38;2;11;11;11m [0m[38;2;114;144;65ml[0m[38;2;120;158;74mo[0m[38;2;116;159;77mo[0m[38;2;112;160;80mo[0m[38;2;107;161;83mo[0m[38;2;28;43;23m.[0m[38;2;0;0;0m [0m[38;2;0;0;0m [0m[38;2;0;0;0m [0m[38;2;36;69;41m.[0m[38;2;76;154;92ml[0m[38;2;91;197;120mx[0m[38;2;92;213;133mx[0m[38;2;86;215;138mx[0m[38;2;79;216;141mx[0m[38;2;74;218;145mx[0m[38;2;69;219;149mx[0m[38;2;60;206;142md[0m[38;2;43;163;114mc[0m[38;2;25;102;72m,[0m[38;2;11;11;11m [0m")
	fmt.Println("[0m[38;2;216;106;83mo[0m[38;2;216;109;81mo[0m[38;2;216;111;80mo[0m[38;2;215;113;79mo[0m[38;2;216;116;77mo[0m[38;2;216;119;76mo[0m[38;2;217;122;75mo[0m[38;2;216;125;73md[0m[38;2;216;127;72md[0m[38;2;216;129;70md[0m[38;2;217;132;69md[0m[38;2;155;96;49mc[0m[38;2;0;0;0m [0m[38;2;0;0;0m [0m[38;2;181;119;54ml[0m[38;2;218;146;63mx[0m[38;2;217;148;61mx[0m[38;2;218;151;60mx[0m[38;2;218;153;59mx[0m[38;2;217;155;57mx[0m[38;2;218;158;56mx[0m[38;2;218;160;55mx[0m[38;2;218;164;54mx[0m[38;2;218;166;52mk[0m[38;2;219;169;51mk[0m[38;2;219;172;50mk[0m[38;2;134;107;30mc[0m[38;2;0;0;0m [0m[38;2;0;0;0m [0m[38;2;59;50;14m.[0m[38;2;207;182;54mk[0m[38;2;202;183;58mk[0m[38;2;197;185;61mk[0m[38;2;191;186;65mk[0m[38;2;186;188;68mk[0m[38;2;143;150;56mo[0m[38;2;11;12;4m [0m[38;2;4;4;2m [0m[38;2;122;145;63ml[0m[38;2;158;195;88mk[0m[38;2;153;196;91mk[0m[38;2;147;198;95mk[0m[38;2;142;199;98mk[0m[38;2;137;201;102mk[0m[38;2;40;62;32m.[0m[38;2;0;0;0m [0m[38;2;0;0;0m [0m[38;2;64;115;65m;[0m[38;2;110;208;120mk[0m[38;2;105;210;124mk[0m[38;2;100;211;127mk[0m[38;2;95;212;131mx[0m[38;2;88;214;135mx[0m[38;2;82;216;140mx[0m[38;2;77;217;143mx[0m[38;2;72;218;147mx[0m[38;2;66;220;150mx[0m[38;2;61;221;154mx[0m[38;2;56;223;158mx[0m[38;2;42;187;134mo[0m")
	fmt.Println("[0m[38;2;216;105;84mo[0m[38;2;216;107;82mo[0m[38;2;216;110;81mo[0m[38;2;216;112;79mo[0m[38;2;171;91;62m:[0m[38;2;41;41;41m [0m[38;2;41;41;41m [0m[38;2;76;43;26m.[0m[38;2;217;126;72md[0m[38;2;217;128;71md[0m[38;2;217;131;70md[0m[38;2;217;134;69md[0m[38;2;0;0;0m [0m[38;2;3;2;1m [0m[38;2;217;141;65md[0m[38;2;217;144;64mx[0m[38;2;217;146;62mx[0m[38;2;218;149;61mx[0m[38;2;181;126;50mo[0m[38;2;54;54;54m [0m[38;2;47;47;47m [0m[38;2;47;47;47m [0m[38;2;134;99;33m:[0m[38;2;218;165;53mk[0m[38;2;218;167;51mk[0m[38;2;218;170;50mk[0m[38;2;192;152;43md[0m[38;2;0;0;0m [0m[38;2;0;0;0m [0m[38;2;60;50;14m.[0m[38;2;210;181;52mk[0m[38;2;205;182;55mk[0m[38;2;200;184;59mk[0m[38;2;194;185;63mk[0m[38;2;189;187;66mk[0m[38;2;184;188;70mk[0m[38;2;162;172;66mx[0m[38;2;142;157;64mo[0m[38;2;166;193;82mk[0m[38;2;161;194;86mk[0m[38;2;156;196;89mk[0m[38;2;150;197;93mk[0m[38;2;145;198;96mk[0m[38;2;140;200;100mk[0m[38;2;42;62;32m.[0m[38;2;0;0;0m [0m[38;2;0;0;0m [0m[38;2;104;180;100md[0m[38;2;113;207;118mk[0m[38;2;108;209;122mk[0m[38;2;103;210;125mk[0m[38;2;98;212;129mk[0m[38;2;65;65;65m [0m[38;2;48;48;48m [0m[38;2;56;56;56m [0m[38;2;56;164;109ml[0m[38;2;69;219;148mx[0m[38;2;64;221;152mx[0m[38;2;59;222;156mx[0m[38;2;53;224;159mx[0m")
	fmt.Println("[0m[38;2;216;103;84mo[0m[38;2;216;106;83mo[0m[38;2;216;109;81mo[0m[38;2;216;111;80mo[0m[38;2;216;114;79mo[0m[38;2;81;43;29m.[0m[38;2;81;45;29m.[0m[38;2;156;87;54m:[0m[38;2;216;124;73md[0m[38;2;217;127;72md[0m[38;2;217;130;71md[0m[38;2;199;121;64mo[0m[38;2;0;0;0m [0m[38;2;3;2;1m [0m[38;2;217;140;66md[0m[38;2;217;142;64md[0m[38;2;217;145;63mx[0m[38;2;217;147;62mx[0m[38;2;181;125;50mo[0m[38;2;0;0;0m [0m[38;2;0;0;0m [0m[38;2;0;0;0m [0m[38;2;112;83;28m;[0m[38;2;218;164;54mx[0m[38;2;218;166;52mk[0m[38;2;218;169;51mk[0m[38;2;192;150;44md[0m[38;2;0;0;0m [0m[38;2;0;0;0m [0m[38;2;60;50;13m.[0m[38;2;213;180;50mk[0m[38;2;208;182;53mk[0m[38;2;203;183;57mk[0m[38;2;197;185;61mk[0m[38;2;107;104;36m;[0m[38;2;187;187;68mk[0m[38;2;181;188;71mk[0m[38;2;175;191;76mk[0m[38;2;169;192;80mk[0m[38;2;95;113;49m:[0m[38;2;159;195;87mk[0m[38;2;154;196;91mk[0m[38;2;148;198;94mk[0m[38;2;143;199;98mk[0m[38;2;42;61;31m.[0m[38;2;0;0;0m [0m[38;2;0;0;0m [0m[38;2;106;179;98md[0m[38;2;116;206;116mk[0m[38;2;111;208;120mk[0m[38;2;106;209;123mk[0m[38;2;100;210;127mk[0m[38;2;0;0;0m [0m[38;2;0;0;0m [0m[38;2;0;0;0m [0m[38;2;58;162;107mc[0m[38;2;72;218;146mx[0m[38;2;67;219;150mx[0m[38;2;62;221;154mx[0m[38;2;56;222;157mx[0m")
	fmt.Println("[0m[38;2;216;102;85mo[0m[38;2;216;104;84mo[0m[38;2;216;107;82mo[0m[38;2;216;110;81mo[0m[38;2;216;112;80mo[0m[38;2;216;115;79mo[0m[38;2;216;117;77mo[0m[38;2;216;120;76mo[0m[38;2;216;122;74md[0m[38;2;217;125;73md[0m[38;2;135;80;45m;[0m[38;2;94;94;94m [0m[38;2;0;0;0m [0m[38;2;3;2;1m [0m[38;2;217;139;66md[0m[38;2;217;141;65md[0m[38;2;217;143;64md[0m[38;2;217;146;62mx[0m[38;2;182;124;51mo[0m[38;2;0;0;0m [0m[38;2;0;0;0m [0m[38;2;0;0;0m [0m[38;2;113;82;29m;[0m[38;2;218;162;54mx[0m[38;2;218;165;53mx[0m[38;2;219;167;52mk[0m[38;2;191;148;44md[0m[38;2;0;0;0m [0m[38;2;0;0;0m [0m[38;2;61;49;13m.[0m[38;2;216;179;48mk[0m[38;2;211;181;51mk[0m[38;2;206;182;55mk[0m[38;2;200;183;58mk[0m[38;2;27;26;9m [0m[38;2;123;123;123m [0m[38;2;29;30;11m [0m[38;2;27;28;11m [0m[38;2;115;115;115m [0m[38;2;47;54;23m.[0m[38;2;162;194;85mk[0m[38;2;156;195;88mk[0m[38;2;151;197;92mk[0m[38;2;146;198;96mk[0m[38;2;43;61;30m.[0m[38;2;0;0;0m [0m[38;2;0;0;0m [0m[38;2;109;179;97md[0m[38;2;119;206;114mk[0m[38;2;114;207;117mk[0m[38;2;109;209;121mk[0m[38;2;103;209;124mk[0m[38;2;0;0;0m [0m[38;2;0;0;0m [0m[38;2;0;0;0m [0m[38;2;61;162;106ml[0m[38;2;75;217;144mx[0m[38;2;70;219;148mx[0m[38;2;65;221;152mx[0m[38;2;60;222;155mx[0m")
	fmt.Println("[0m[38;2;216;101;86ml[0m[38;2;215;103;84mo[0m[38;2;216;106;83mo[0m[38;2;216;108;82mo[0m[38;2;180;92;67mc[0m[38;2;81;81;81m [0m[38;2;81;81;81m [0m[38;2;77;77;77m [0m[38;2;62;62;62m [0m[38;2;56;56;56m [0m[38;2;28;28;28m [0m[38;2;0;0;0m [0m[38;2;0;0;0m [0m[38;2;36;36;36m [0m[38;2;208;131;64mo[0m[38;2;217;139;65md[0m[38;2;217;142;64md[0m[38;2;217;145;63mx[0m[38;2;218;147;62mx[0m[38;2;104;71;29m,[0m[38;2;93;65;26m'[0m[38;2;93;66;25m'[0m[38;2;195;141;50md[0m[38;2;218;160;55mx[0m[38;2;218;163;54mx[0m[38;2;219;166;52mk[0m[38;2;182;140;43mo[0m[38;2;0;0;0m [0m[38;2;0;0;0m [0m[38;2;61;49;13m.[0m[38;2;218;178;46mk[0m[38;2;214;180;49mk[0m[38;2;209;181;53mk[0m[38;2;204;183;56mk[0m[38;2;22;20;7m [0m[38;2;0;0;0m [0m[38;2;17;17;17m [0m[38;2;19;19;19m [0m[38;2;0;0;0m [0m[38;2;42;48;20m.[0m[38;2;165;193;83mk[0m[38;2;159;194;86mk[0m[38;2;154;196;90mk[0m[38;2;149;198;94mk[0m[38;2;44;61;30m.[0m[38;2;0;0;0m [0m[38;2;0;0;0m [0m[38;2;101;161;86mo[0m[38;2;123;205;112mk[0m[38;2;117;206;116mk[0m[38;2;112;208;119mk[0m[38;2;106;209;122mk[0m[38;2;51;105;63m,[0m[38;2;41;91;56m'[0m[38;2;43;102;64m,[0m[38;2;84;215;139mx[0m[38;2;79;217;142mx[0m[38;2;73;218;146mx[0m[38;2;68;220;149mx[0m[38;2;63;221;153mx[0m")
	fmt.Println("[0m[38;2;215;99;86ml[0m[38;2;215;101;85mo[0m[38;2;216;104;84mo[0m[38;2;216;107;82mo[0m[38;2;180;91;68mc[0m[38;2;0;0;0m [0m[38;2;0;0;0m [0m[38;2;0;0;0m [0m[38;2;0;0;0m [0m[38;2;0;0;0m [0m[38;2;0;0;0m [0m[38;2;0;0;0m [0m[38;2;0;0;0m [0m[38;2;0;0;0m [0m[38;2;108;108;108m [0m[38;2;145;92;44m:[0m[38;2;218;141;65md[0m[38;2;217;143;64md[0m[38;2;218;146;63mx[0m[38;2;218;148;61mx[0m[38;2;218;151;60mx[0m[38;2;217;153;59mx[0m[38;2;218;156;57mx[0m[38;2;218;159;56mx[0m[38;2;218;161;54mx[0m[38;2;109;82;27m,[0m[38;2;98;98;98m [0m[38;2;0;0;0m [0m[38;2;0;0;0m [0m[38;2;58;46;13m.[0m[38;2;219;177;47mk[0m[38;2;217;179;47mk[0m[38;2;212;181;51mk[0m[38;2;206;182;54mk[0m[38;2;23;20;6m [0m[38;2;0;0;0m [0m[38;2;0;0;0m [0m[38;2;0;0;0m [0m[38;2;0;0;0m [0m[38;2;43;48;20m.[0m[38;2;168;193;81mk[0m[38;2;163;194;84mk[0m[38;2;157;195;88mk[0m[38;2;152;197;92mk[0m[38;2;43;58;28m.[0m[38;2;0;0;0m [0m[38;2;0;0;0m [0m[38;2;99;99;99m [0m[38;2;60;98;53m,[0m[38;2;120;205;113mk[0m[38;2;115;207;117mk[0m[38;2;109;208;120mk[0m[38;2;103;210;125mk[0m[38;2;97;211;129mk[0m[38;2;92;213;133mx[0m[38;2;87;215;137mx[0m[38;2;82;216;140mx[0m[38;2;76;217;143mx[0m[38;2;49;151;101mc[0m[38;2;117;117;117m [0m")
}
