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

// generateMorseAudio generates audio for a given text in Morse code
func generateMorseAudio(text string, wpm int, freq int) ([]int16, int) {
	dotDuration, dashDuration, elementGap, charGap, wordGap := calculateMorseTiming(wpm)

	// Convert durations from milliseconds to samples
	dotSamples := int(float64(dotDuration) * sampleRate / 1000)
	dashSamples := int(float64(dashDuration) * sampleRate / 1000)
	elementGapSamples := int(float64(elementGap) * sampleRate / 1000)
	charGapSamples := int(float64(charGap) * sampleRate / 1000)
	wordGapSamples := int(float64(wordGap) * sampleRate / 1000)

	// Calculate total duration needed
	totalSamples := 0
	for i, char := range strings.ToUpper(text) {
		if char == ' ' {
			totalSamples += wordGapSamples
			continue
		}

		if morse, ok := morseCodeMap[char]; ok {
			for j, element := range morse {
				if element == '.' {
					totalSamples += dotSamples
				} else if element == '-' {
					totalSamples += dashSamples
				}
				if j < len(morse)-1 {
					totalSamples += elementGapSamples
				}
			}
			if i < len(text)-1 && text[i+1] != ' ' {
				totalSamples += charGapSamples
			}
		}
	}

	// Generate the audio samples
	samples := make([]int16, totalSamples)
	currentSample := 0

	for i, char := range strings.ToUpper(text) {
		if char == ' ' {
			currentSample += wordGapSamples
			continue
		}

		if morse, ok := morseCodeMap[char]; ok {
			for j, element := range morse {
				duration := dotSamples
				if element == '-' {
					duration = dashSamples
				}

				// Generate tone for the element
				for k := 0; k < duration; k++ {
					if currentSample+k < len(samples) {
						samples[currentSample+k] = int16(math.Sin(2*math.Pi*float64(freq)*float64(k)/float64(sampleRate)) * 32767)
					}
				}
				currentSample += duration

				// Add element gap if not the last element
				if j < len(morse)-1 {
					currentSample += elementGapSamples
				}
			}

			// Add character gap if not the last character
			if i < len(text)-1 && text[i+1] != ' ' {
				currentSample += charGapSamples
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
