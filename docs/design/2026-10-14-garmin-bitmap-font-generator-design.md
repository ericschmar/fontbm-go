# Garmin Bitmap Font Generator Design

## Goal

Create a Go tool that converts a TTF font into a Garmin-compatible bitmap font: a single-page PNG atlas and a BMFont text descriptor saved with a `.txt` extension. The same generation pipeline is available as a Go library and as a CLI.

## Scope

- Input one TTF font and a target pixel size.
- Rasterize a selected Unicode character set. If no character selection is supplied, generate printable ASCII.
- Deduplicate repeated characters while preserving deterministic output order.
- Generate one PNG atlas page and a BMFont text descriptor referring to that page.
- Expose in-memory generation through a Go API; have the CLI write the returned outputs to files.
- Exclude kerning, XML/JSON formats, sprite-sheet/frame inputs, multiple atlas pages, and automatic inclusion of every glyph in the font.

## Approaches considered

- Use `golang.org/x/image` font parsing/rasterization with a small in-house single-page rectangle packer. This uses established Go image APIs and avoids native runtime requirements; recommended.
- Use a higher-level font/atlas package. This may reduce implementation effort, but adds dependency and Garmin-compatibility behavior to validate.
- Use a native FreeType binding. This provides a mature rasterizer, but adds native build/runtime requirements and reduces portability.

## Architecture and data flow

- The library accepts font bytes, generation options, and selected characters, then parses the TTF, rasterizes the requested glyphs, packs them into a single atlas, writes the PNG in memory, and serializes BMFont text metadata.
- The API returns the descriptor text and PNG bytes together with useful output metadata. The CLI calls this same API rather than duplicating the generation logic.
- CLI arguments include font path, pixel size, output basename, maximum atlas size, and optional character string or UTF-8 character file. The character options are mutually exclusive. If neither is supplied, printable ASCII is used.
- The CLI writes `<basename>.txt` and `<basename>_0.png`. The page filename stored in the descriptor is relative to the descriptor and matches the PNG name.

## Rendering and BMFont output

- Rasterize requested characters at the selected pixel size and preserve each glyph's advance and bearing metrics.
- A requested character missing from the font is an error; do not silently substitute a fallback glyph.
- Spaces and other glyphs with no visible ink remain represented with zero-sized atlas bounds and their proper advance.
- Pack visible glyphs onto one power-of-two atlas page, growing it as necessary up to a configurable maximum. Report an error if the glyphs cannot fit.
- Encode glyph coverage in grayscale RGB with fully opaque alpha, following the documented Garmin-compatible approach. Set BMFont channel metadata consistently with glyph coverage being present in RGB, not alpha.
- Emit BMFont text tags for `info`, `common`, `page`, `chars`, and each `char`, including Unicode codepoint IDs, atlas rectangles, offsets, advances, page, and channel. Do not emit kerning records.
- Set `common` baseline and line-height values consistently from the rasterizer's font metrics, and use the atlas's actual dimensions for `scaleW` and `scaleH`.

## Errors

Return clear errors for unreadable or invalid fonts, invalid generation options, unsupported requested characters, atlas overflow, and filesystem read/write failures. The library returns errors without writing files; file I/O errors are handled by the CLI boundary.

## Testing

- Use a deterministic bundled test font so tests do not depend on installed system fonts.
- Library tests cover descriptor fields, glyph positions and advances, baseline/line-height metrics, spaces, Unicode character IDs, opaque grayscale atlas pixels, duplicate characters, missing glyph errors, and atlas overflow.
- CLI tests cover argument validation, printable-ASCII default, explicit character input, and writing both expected output files with consistent descriptor page references.
- Verify generated output against BMFont text-format conventions and, where practical, load it in a Garmin Connect IQ project or simulator.

## Repository context

The repository currently contains a title-only README and an MIT license. The reference `bitmaptext-creator` project builds atlases from supplied character images and emits Phaser-oriented metadata; this design instead renders glyphs directly from a TTF and targets Garmin BMFont text metadata.

## Compatibility references

- AngelCode BMFont file format documentation: https://www.angelcode.com/products/bmfont/doc/file_format.html
- Garmin custom-font workflow and opaque RGB antialiasing note: https://jeffchen.dev/posts/Garmin-Watch-Faces-Custom-Fonts-On-MacOS/
