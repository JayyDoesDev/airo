package voice

import (
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/hraban/opus"
)

const (
	sampleRate  = 48000
	channels    = 2
	frameSizems = 20
	frameSize   = sampleRate * frameSizems / 1000
)

func Speak(vc *discordgo.VoiceConnection, text string) error {
	pcm, err := tts(text)
	if err != nil {
		fmt.Printf("[speak] tts error: %v\n", err)
		return err
	}
	fmt.Printf("[speak] tts ok, pcm bytes=%d\n", len(pcm))

	enc, err := opus.NewEncoder(sampleRate, channels, opus.Application(2048))
	if err != nil {
		fmt.Printf("[speak] opus encoder error: %v\n", err)
		return fmt.Errorf("opus encoder: %w", err)
	}

	vc.Speaking(true)
	defer vc.Speaking(false)

	buf := make([]byte, 4096)
	samples := make([]int16, frameSize*channels)

	for len(pcm) > 0 {
		need := frameSize * channels * 2
		chunk := pcm
		if len(chunk) > need {
			chunk = pcm[:need]
		}
		pcm = pcm[len(chunk):]

		for i := range samples {
			samples[i] = 0
		}
		for i := 0; i+1 < len(chunk); i += 2 {
			idx := i / 2
			if idx < len(samples) {
				samples[idx] = int16(binary.LittleEndian.Uint16(chunk[i:]))
			}
		}

		n, err := enc.Encode(samples, buf)
		if err != nil {
			return fmt.Errorf("opus encode: %w", err)
		}

		frame := make([]byte, n)
		copy(frame, buf[:n])

		select {
		case vc.OpusSend <- frame:
		case <-time.After(5 * time.Second):
			return fmt.Errorf("opus send timeout")
		}
	}

	return nil
}

func tts(text string) ([]byte, error) {
	modelPath := os.Getenv("PIPER_MODEL")
	if modelPath != "" {
		return ttsPiper(text, modelPath)
	}
	return ttsSay(text)
}

func ttsPiper(text, modelPath string) ([]byte, error) {
	modelRate := 22050
	if r := os.Getenv("PIPER_SAMPLE_RATE"); r != "" {
		if v, err := strconv.Atoi(r); err == nil {
			modelRate = v
		}
	}

	piperCmd := exec.Command("piper",
		"--model", modelPath,
		"--output_raw",
	)
	piperCmd.Stdin = strings.NewReader(text)

	piperOut, err := piperCmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := piperCmd.Start(); err != nil {
		return nil, fmt.Errorf("piper not found — is it installed? %w", err)
	}

	ffmpegCmd := exec.Command("ffmpeg",
		"-f", "s16le", "-ar", strconv.Itoa(modelRate), "-ac", "1",
		"-i", "pipe:0",
		"-f", "s16le", "-ar", strconv.Itoa(sampleRate), "-ac", strconv.Itoa(channels),
		"-loglevel", "quiet",
		"pipe:1",
	)
	ffmpegCmd.Stdin = piperOut

	pcm, err := ffmpegCmd.Output()
	piperCmd.Wait()
	if err != nil {
		return nil, fmt.Errorf("ffmpeg resample failed: %w", err)
	}
	return pcm, nil
}

func ttsSay(text string) ([]byte, error) {
	voice := os.Getenv("SAY_VOICE")
	if voice == "" {
		voice = "Samantha"
	}

	tmp, err := os.CreateTemp("", "aira-tts-*.aiff")
	if err != nil {
		return nil, err
	}
	tmp.Close()
	defer os.Remove(tmp.Name())

	sayArgs := []string{"-v", voice, "-o", tmp.Name(), text}
	if out, err := exec.Command("say", sayArgs...).CombinedOutput(); err != nil {
		return nil, fmt.Errorf("say failed: %w — %s", err, out)
	}

	ffmpegCmd := exec.Command("ffmpeg",
		"-i", tmp.Name(),
		"-f", "s16le", "-ar", strconv.Itoa(sampleRate), "-ac", strconv.Itoa(channels),
		"-loglevel", "quiet",
		"pipe:1",
	)
	pcm, err := ffmpegCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffmpeg convert failed: %w", err)
	}
	return pcm, nil
}
