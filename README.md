# fontbm-go

A Go library and CLI for generating Garmin-compatible bitmap fonts from TrueType fonts. It writes a BMFont text descriptor (`.txt`) and a single-page PNG atlas. Atlas output defaults to grayscale and can optionally use RGB.

## CLI

```sh
go run ./cmd/fontbm -font ./MyFont.ttf -size 32 -output ./dist/myfont -chars '0123456789:.-'
```

This creates `./dist/myfont.txt` and `./dist/myfont_0.png`. If `-chars` and `-chars-file` are omitted, printable ASCII is included.

Use `-chars-file` for a UTF-8 character set file. The two character options cannot be combined. `-max-size` sets the maximum power-of-two atlas dimensions (default `4096`). `-color grayscale` selects the Garmin-friendly grayscale PNG (the default); `-color rgb` selects opaque RGB output.

## Go library

```go
fontData, err := os.ReadFile("MyFont.ttf")
if err != nil {
    log.Fatal(err)
}
result, err := fontbm.Generate(fontbm.Options{
    FontData: fontData,
    FontName: "myfont",
    PageName: "myfont_0.png",
    Chars: "0123456789:.-",
    PixelSize: 32,
    ColorMode: "grayscale", // or "rgb"
})
if err != nil {
    log.Fatal(err)
}
if err := os.WriteFile("myfont.txt", []byte(result.Descriptor), 0644); err != nil {
    log.Fatal(err)
}
if err := os.WriteFile("myfont_0.png", result.PNG, 0644); err != nil {
    log.Fatal(err)
}
```

Import path: `github.com/ericschmar/fontbm-go`. `MaxAtlasSize` defaults to 4096, omitted `Chars` selects printable ASCII, and omitted `ColorMode` selects grayscale. Set `ColorMode` to `"rgb"` for RGB output. `ReadChars` validates UTF-8 for character files.

## Garmin use

Place the descriptor and PNG in your Connect IQ resources and reference the descriptor from `fonts.xml`, for example:

```xml
<font id="my_font" filename="myfont.txt" antialias="true" />
```

The generated descriptor references the PNG by the page name supplied to the library; the CLI sets it to `<basename>_0.png`.

## Test

```sh
go test ./...
```

The rendering pipeline uses `golang.org/x/image` for TrueType parsing and rasterization.