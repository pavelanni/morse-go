# Morse Code Audio Generator

A simple and efficient Morse code audio generator written in Go. This project is designed to be integrated into the [Field Day Registration Kiosk](https://github.com/pavelanni/field-day-go) to provide audio feedback of call signs in Morse code.

## Features

- Generates Morse code audio using sine wave tones
- Optimized for resource-constrained devices (like Orange Pi Zero)
- Pre-generates audio samples for maximum performance
- Configurable speed (WPM) and frequency
- Supports all standard Morse code characters (A-Z, 0-9, and common punctuation)

## Usage

```bash
go run . -text "SOS" -wpm 20 -freq 600
```

### Command Line Arguments

- `-text`: Text to convert to Morse code (default: "SOS")
- `-wpm`: Speed in words per minute (default: 20)
- `-freq`: Tone frequency in Hz (default: 600)

## Performance Optimization

The generator uses a pre-generation approach where all audio samples are created once and stored in memory. This provides several benefits:

- Eliminates repeated floating-point calculations
- Reduces CPU usage during playback
- Makes audio generation more predictable
- Particularly efficient for repeated playback of the same characters

Note that this optimization means the frequency and WPM parameters are fixed after initialization, which is ideal for the registration kiosk use case where these parameters remain constant.

## Integration with Field Day Registration Kiosk

This Morse code generator will be integrated into the [Field Day Registration Kiosk](https://github.com/pavelanni/field-day-go) project to provide audio feedback when visitors enter their call signs. The integration will help operators verify call signs by ear, adding an extra layer of validation to the registration process.

## Dependencies

- [Ebiten](https://github.com/hajimehoshi/ebiten) - For audio playback

## License

This project is part of the Field Day Registration Kiosk project and is subject to the same license terms.