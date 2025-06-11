package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

const (
	sampleRate = 44100
)

// Morse code table
var morseCodeMap = map[rune]string{
	'A': ".-", 'B': "-...", 'C': "-.-.", 'D': "-..", 'E': ".", 'F': "..-.",
	'G': "--.", 'H': "....", 'I': "..", 'J': ".---", 'K': "-.-", 'L': ".-..",
	'M': "--", 'N': "-.", 'O': "---", 'P': ".--.", 'Q': "--.-", 'R': ".-.",
	'S': "...", 'T': "-", 'U': "..-", 'V': "...-", 'W': ".--", 'X': "-..-",
	'Y': "-.--", 'Z': "--..",
	'0': "-----", '1': ".----", '2': "..---", '3': "...--", '4': "....-",
	'5': ".....", '6': "-....", '7': "--...", '8': "---..", '9': "----.",
	'/': "-..-.", '?': "..--..", '.': ".-.-.-", ',': "--..--",
}

// Pre-generated audio samples for each character
type morseAudio struct {
	dotSamples  []int16
	dashSamples []int16
	elementGap  []int16
	charGap     []int16
	wordGap     []int16
	charSamples map[rune][]int16
}

// calculateMorseTiming calculates timing from WPM using PARIS standard
func calculateMorseTiming(wpm int) (dotDuration, dashDuration, elementGap, charGap, wordGap int) {
	if wpm <= 0 {
		wpm = 20 // Default to 20 WPM
	}

	// 1 time unit duration in milliseconds
	timeUnit := 60000 / (wpm * 50) // 60 seconds * 1000 ms / (wpm * 50 units per PARIS)

	dotDuration = timeUnit
	dashDuration = timeUnit * 3
	elementGap = timeUnit
	charGap = timeUnit * 3
	wordGap = timeUnit * 7

	return
}

// newMorseAudio creates a new morseAudio instance with pre-generated samples
func newMorseAudio(wpm int, freq int) *morseAudio {
	dotDuration, dashDuration, elementGap, charGap, wordGap := calculateMorseTiming(wpm)

	// Convert durations from milliseconds to samples
	dotSamples := int(float64(dotDuration) * sampleRate / 1000)
	dashSamples := int(float64(dashDuration) * sampleRate / 1000)
	elementGapSamples := int(float64(elementGap) * sampleRate / 1000)
	charGapSamples := int(float64(charGap) * sampleRate / 1000)
	wordGapSamples := int(float64(wordGap) * sampleRate / 1000)

	// Generate basic elements
	dot := make([]int16, dotSamples)
	dash := make([]int16, dashSamples)
	elementGapAudio := make([]int16, elementGapSamples)
	charGapAudio := make([]int16, charGapSamples)
	wordGapAudio := make([]int16, wordGapSamples)

	// Generate tone for dot and dash
	for i := 0; i < dotSamples; i++ {
		dot[i] = int16(math.Sin(2*math.Pi*float64(freq)*float64(i)/float64(sampleRate)) * 32767)
	}
	for i := 0; i < dashSamples; i++ {
		dash[i] = int16(math.Sin(2*math.Pi*float64(freq)*float64(i)/float64(sampleRate)) * 32767)
	}

	// Pre-generate samples for each character
	charSamples := make(map[rune][]int16)
	for char, morse := range morseCodeMap {
		var samples []int16
		for i, element := range morse {
			if element == '.' {
				samples = append(samples, dot...)
			} else if element == '-' {
				samples = append(samples, dash...)
			}
			if i < len(morse)-1 {
				samples = append(samples, elementGapAudio...)
			}
		}
		charSamples[char] = samples
	}

	return &morseAudio{
		dotSamples:  dot,
		dashSamples: dash,
		elementGap:  elementGapAudio,
		charGap:     charGapAudio,
		wordGap:     wordGapAudio,
		charSamples: charSamples,
	}
}

// generateMorseAudio generates audio for a given text in Morse code
func generateMorseAudio(text string, wpm int, freq int) ([]int16, int) {
	morse := newMorseAudio(wpm, freq)

	// Calculate total duration needed
	totalSamples := 0
	for i, char := range strings.ToUpper(text) {
		if char == ' ' {
			totalSamples += len(morse.wordGap)
			continue
		}

		if samples, ok := morse.charSamples[char]; ok {
			totalSamples += len(samples)
			if i < len(text)-1 && text[i+1] != ' ' {
				totalSamples += len(morse.charGap)
			}
		}
	}

	// Generate the audio samples
	samples := make([]int16, totalSamples)
	currentSample := 0

	for i, char := range strings.ToUpper(text) {
		if char == ' ' {
			copy(samples[currentSample:], morse.wordGap)
			currentSample += len(morse.wordGap)
			continue
		}

		if charSamples, ok := morse.charSamples[char]; ok {
			copy(samples[currentSample:], charSamples)
			currentSample += len(charSamples)

			// Add character gap if not the last character
			if i < len(text)-1 && text[i+1] != ' ' {
				copy(samples[currentSample:], morse.charGap)
				currentSample += len(morse.charGap)
			}
		}
	}

	return samples, totalSamples
}

func main() {
	// Parse command line arguments
	text := flag.String("text", "SOS", "Text to convert to Morse code")
	wpm := flag.Int("wpm", 20, "Speed in words per minute")
	freq := flag.Int("freq", 600, "Tone frequency in Hz")
	flag.Parse()

	acontext := audio.NewContext(sampleRate)

	// Generate Morse code audio
	samples, totalSamples := generateMorseAudio(*text, *wpm, *freq)

	// Create a buffer and write WAV data
	buf := &bytes.Buffer{}
	writeWavHeader(buf, totalSamples*2, sampleRate)

	// Write PCM data
	for _, sample := range samples {
		binary.Write(buf, binary.LittleEndian, sample)
	}

	// Create a reader from the buffer
	reader := bytes.NewReader(buf.Bytes())

	// Play the sound
	p, err := wav.DecodeWithSampleRate(sampleRate, reader)
	if err != nil {
		panic(err)
	}

	player, err := acontext.NewPlayer(p)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Playing '%s' in Morse code at %d WPM, %d Hz\n", *text, *wpm, *freq)
	player.Play()

	// Calculate total duration and wait for playback to complete
	totalDuration := time.Duration(float64(totalSamples) / float64(sampleRate) * float64(time.Second))
	time.Sleep(totalDuration)
}

func writeWavHeader(w *bytes.Buffer, dataSize int, sampleRate int) {
	// RIFF header
	w.Write([]byte("RIFF"))
	binary.Write(w, binary.LittleEndian, uint32(36+dataSize))
	w.Write([]byte("WAVE"))

	// fmt chunk
	w.Write([]byte("fmt "))
	binary.Write(w, binary.LittleEndian, uint32(16)) // fmt chunk size
	binary.Write(w, binary.LittleEndian, uint16(1))  // audio format (1 for PCM)
	binary.Write(w, binary.LittleEndian, uint16(1))  // number of channels
	binary.Write(w, binary.LittleEndian, uint32(sampleRate))
	binary.Write(w, binary.LittleEndian, uint32(sampleRate*2)) // byte rate
	binary.Write(w, binary.LittleEndian, uint16(2))            // block align
	binary.Write(w, binary.LittleEndian, uint16(16))           // bits per sample

	// data chunk
	w.Write([]byte("data"))
	binary.Write(w, binary.LittleEndian, uint32(dataSize))
}
