package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ericschmar/fontbm-go"
)

func main() {
	fontPath := flag.String("font", "", "path to a TTF font (required)")
	output := flag.String("output", "font", "output basename")
	chars := flag.String("chars", "", "characters to include")
	charsFile := flag.String("chars-file", "", "UTF-8 file containing characters to include")
	pixelSize := flag.Float64("size", 32, "font size in pixels")
	maxSize := flag.Int("max-size", 4096, "maximum atlas width and height")
	flag.Parse()
	if err := run(*fontPath, *output, *chars, *charsFile, *pixelSize, *maxSize); err != nil {
		fmt.Fprintln(os.Stderr, "fontbm:", err)
		os.Exit(1)
	}
}

func run(fontPath, output, chars, charsFile string, pixelSize float64, maxSize int) error {
	if fontPath == "" {
		return fmt.Errorf("-font is required")
	}
	if chars != "" && charsFile != "" {
		return fmt.Errorf("-chars and -chars-file cannot be used together")
	}
	fontData, err := os.ReadFile(fontPath)
	if err != nil {
		return fmt.Errorf("read font: %w", err)
	}
	if charsFile != "" {
		data, err := os.ReadFile(charsFile)
		if err != nil {
			return fmt.Errorf("read character file: %w", err)
		}
		chars, err = fontbm.ReadChars(data)
		if err != nil {
			return err
		}
	}
	pageName := filepath.Base(output) + "_0.png"
	result, err := fontbm.Generate(fontbm.Options{FontData: fontData, FontName: filepath.Base(output), PageName: pageName, Chars: chars, PixelSize: pixelSize, MaxAtlasSize: maxSize})
	if err != nil {
		return err
	}
	if err := os.WriteFile(output+".txt", []byte(result.Descriptor), 0644); err != nil {
		return fmt.Errorf("write descriptor: %w", err)
	}
	if err := os.WriteFile(output+"_0.png", result.PNG, 0644); err != nil {
		return fmt.Errorf("write atlas: %w", err)
	}
	return nil
}
