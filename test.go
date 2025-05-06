package main

import (
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	_ "image/png" // Import for decoding PNGs
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/font" // Required for text overlay
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"

	// For resizing
	xdraw "golang.org/x/image/draw"

	// For blurring (optional, requires external package or implementation)
	// Example using github.com/disintegration/imaging:
	// "github.com/disintegration/imaging"
	// Or implement a simple box blur or use x/image/blur if suitable
)

// Target dimensions and safe zone based on the Python script logic
const (
	targetContentW = 1016 // Core content width (1080 - 2 * safeZoneW)
	targetContentH = 1350 // Core content height (matches final tile height)
	safeZoneW      = 32   // Width of the safe zone (padding or blur) on each side
	finalTileW     = targetContentW + 2*safeZoneW // 1080
	finalTileH     = targetContentH             // 1350 (4:5 aspect ratio)
	jpegQuality    = 95
)

// Helper function: Min for integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Helper function: Max for integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func main() {
	inPath := flag.String("in", "", "Input image path")
	rows := flag.Int("r", 1, "Number of rows to split into (default: 1)")
	cols := flag.Int("c", 1, "Number of columns to split into (default: 1)")
	outDir := flag.String("out", "output", "Output directory path (default: ./output)")
	edgeMode := flag.String("edge-mode", "pad", "Safe zone mode: 'pad' (white) or 'blur'") // Default 'pad' seems safer if blur isn't perfect
	resizeMode := flag.String("resize-mode", "resize", "Action if image is smaller than grid: 'resize' or 'pad'")
	interactive := flag.Bool("interactive", false, "Use interactive prompts (not implemented)")
	flag.Parse()

	// --- Basic Validation ---
	if *inPath == "" {
		if !*interactive {
			fatal(errors.New("flag -in is required"))
		} else {
			fatal(errors.New("interactive mode not implemented, please provide -in flag"))
		}
	}
	if *rows <= 0 || *cols <= 0 {
		fatal(errors.New("rows and columns must be positive integers"))
	}
	if *edgeMode != "pad" && *edgeMode != "blur" {
		fatal(errors.New("edge-mode must be 'pad' or 'blur'"))
	}
	if *resizeMode != "resize" && *resizeMode != "pad" {
		fatal(errors.New("resize-mode must be 'resize' or 'pad'"))
	}

	// --- Create Output Directory ---
	if err := os.MkdirAll(*outDir, 0755); err != nil {
		fatal(err)
	}

	// --- Load Source Image ---
	src := load(*inPath)
	origBounds := src.Bounds()
	origW := origBounds.Dx()
	origH := origBounds.Dy()
	fmt.Printf("Loaded image: %s (%d x %d)\n", *inPath, origW, origH)

	// --- Calculate Required Grid Content Size ---
	totalContentW := targetContentW * (*cols)
	totalContentH := targetContentH * (*rows)
	fmt.Printf("Required content size for %d x %d grid: %d x %d\n", *rows, *cols, totalContentW, totalContentH)

	var processedSrc image.Image = src // Start with the original image

	// --- Handle Image Size Mismatch (Based on Python Logic) ---
	if origW < totalContentW || origH < totalContentH {
		action := *resizeMode
		if *interactive {
			// Interactive prompt would go here
			fmt.Println("Warning: Original image is smaller than the required grid content size.")
			fmt.Printf("Using specified resize-mode: %s\n", action)
		} else {
			fmt.Printf("Warning: Original image (%dx%d) is smaller than required grid content size (%dx%d).\n", origW, origH, totalContentW, totalContentH)
			fmt.Printf("Applying resize-mode: %s\n", action)
		}

		if action == "resize" {
			fmt.Printf("Resizing image to fit content area: %d x %d\n", totalContentW, totalContentH)
			resizedImg := image.NewRGBA(image.Rect(0, 0, totalContentW, totalContentH))
			xdraw.CatmullRom.Scale(resizedImg, resizedImg.Bounds(), src, src.Bounds(), draw.Over, nil)
			processedSrc = resizedImg
		} else if action == "pad" {
			fmt.Println("Padding image to fit content area...")
			padW := max(0, totalContentW-origW)
			padH := max(0, totalContentH-origH)
			leftPad := padW / 2
			topPad := padH / 2

			paddedImg := image.NewRGBA(image.Rect(0, 0, totalContentW, totalContentH))
			draw.Draw(paddedImg, paddedImg.Bounds(), image.Black, image.Point{}, draw.Src) // Fill background black
			draw.Draw(paddedImg, image.Rect(leftPad, topPad, leftPad+origW, topPad+origH), src, image.Point{}, draw.Over)
			processedSrc = paddedImg
		}
	} else if origW > totalContentW || origH > totalContentH {
		fmt.Println("Image larger than required content size, center cropping...")
		cropX := (origW - totalContentW) / 2
		cropY := (origH - totalContentH) / 2
		cropRect := image.Rect(cropX, cropY, cropX+totalContentW, cropY+totalContentH)

		croppedImg := image.NewRGBA(image.Rect(0, 0, totalContentW, totalContentH))
		draw.Draw(croppedImg, croppedImg.Bounds(), src, cropRect.Min, draw.Src)
		processedSrc = croppedImg
	} else {
		fmt.Println("Image size matches required content size exactly.")
		// Ensure processedSrc is drawable if it came directly from decode
		if _, ok := processedSrc.(draw.Image); !ok {
			rgba := image.NewRGBA(processedSrc.Bounds())
			draw.Draw(rgba, rgba.Bounds(), processedSrc, image.Point{}, draw.Src)
			processedSrc = rgba
		}
	}

	// --- Process and Split Tiles ---
	numTiles := (*rows) * (*cols)
	processedBounds := processedSrc.Bounds()
	processedW := processedBounds.Dx()
	processedH := processedBounds.Dy()

	fmt.Printf("Processing tiles from source sized: %d x %d\n", processedW, processedH)
	fmt.Printf("Target content size per tile: %d x %d\n", targetContentW, targetContentH)
	fmt.Printf("Final tile size (with safe zones): %d x %d\n", finalTileW, finalTileH)

	allFinalTiles := make([][]image.Image, *rows)
	for r := range allFinalTiles {
		allFinalTiles[r] = make([]image.Image, *cols)
	}

	for r := 0; r < *rows; r++ {
		for c := 0; c < *cols; c++ {
			contentX0 := c * targetContentW
			contentY0 := r * targetContentH
			contentX1 := contentX0 + targetContentW
			contentY1 := contentY0 + targetContentH
			contentRect := image.Rect(contentX0, contentY0, contentX1, contentY1)

			contentTile := image.NewRGBA(image.Rect(0, 0, targetContentW, targetContentH))
			draw.Draw(contentTile, contentTile.Bounds(), processedSrc, contentRect.Min, draw.Src)

			finalTile := image.NewRGBA(image.Rect(0, 0, finalTileW, finalTileH))

			if *edgeMode == "pad" {
				draw.Draw(finalTile, finalTile.Bounds(), image.White, image.Point{}, draw.Src)
				pastePoint := image.Point{X: safeZoneW, Y: 0}
				draw.Draw(finalTile, contentTile.Bounds().Add(pastePoint), contentTile, image.Point{0, 0}, draw.Over)
			} else if *edgeMode == "blur" {
				if targetContentW < safeZoneW*2 {
					fmt.Fprintf(os.Stderr, "Warning: Tile content width (%d) is too small for blur zones (%d). Falling back to padding for tile (%d,%d).\n", targetContentW, safeZoneW*2, r, c)
					draw.Draw(finalTile, finalTile.Bounds(), image.White, image.Point{}, draw.Src)
					pastePoint := image.Point{X: safeZoneW, Y: 0}
					draw.Draw(finalTile, contentTile.Bounds().Add(pastePoint), contentTile, image.Point{0, 0}, draw.Over)
				} else {
					leftEdgeRect := image.Rect(0, 0, safeZoneW, targetContentH)
					leftEdge := image.NewRGBA(leftEdgeRect)
					draw.Draw(leftEdge, leftEdge.Bounds(), contentTile, leftEdgeRect.Min, draw.Src)

					rightEdgeRect := image.Rect(targetContentW-safeZoneW, 0, targetContentW, targetContentH)
					rightEdge := image.NewRGBA(image.Rect(0, 0, safeZoneW, targetContentH))
					draw.Draw(rightEdge, rightEdge.Bounds(), contentTile, rightEdgeRect.Min, draw.Src)

					// Consider replacing boxBlur with a more robust blur if artifacts persist
					blurredLeft := boxBlur(leftEdge, 10)
					blurredRight := boxBlur(rightEdge, 10)

					draw.Draw(finalTile, blurredLeft.Bounds(), blurredLeft, image.Point{0, 0}, draw.Src)
					contentPasteRect := image.Rect(safeZoneW, 0, safeZoneW+targetContentW, targetContentH)
					draw.Draw(finalTile, contentPasteRect, contentTile, image.Point{0, 0}, draw.Src) // Use Src to overwrite potentially overlapping blur
					rightPasteRect := image.Rect(safeZoneW+targetContentW, 0, finalTileW, finalTileH)
					draw.Draw(finalTile, rightPasteRect, blurredRight, image.Point{0, 0}, draw.Src)
				}
			}

			tileNumber := numTiles - (r*(*cols) + c)
			outName := fmt.Sprintf("tile_%d.jpg", tileNumber)
			outPath := filepath.Join(*outDir, outName)

			if finalTile.Bounds().Empty() {
				fmt.Fprintf(os.Stderr, "Warning: Skipping empty tile %s\n", outName)
			} else {
				saveJPEG(outPath, finalTile)
				fmt.Printf("✔ Saved tile %s (%d x %d)\n", outPath, finalTileW, finalTileH)
			}
			allFinalTiles[r][c] = finalTile
		}
	}

	// --- Stitch Tiles for Preview (Optional) ---
	stitchOutputPath := filepath.Join(*outDir, "stitched_preview.jpg")
	if len(allFinalTiles) > 0 && len(allFinalTiles[0]) > 0 && allFinalTiles[0][0] != nil {
		fmt.Println("Stitching final tiles for preview...")
		stitchFinalTiles(stitchOutputPath, allFinalTiles, *rows, *cols)
		fmt.Println("✔ Stitched preview:", stitchOutputPath)
	} else {
		fmt.Fprintf(os.Stderr, "Skipping stitch: No valid tiles generated.\n")
	}

	fmt.Printf("Processing complete. Output saved to '%s'\n", *outDir)
}

// --- Utility functions ---

func load(path string) image.Image {
	f, err := os.Open(path)
	if err != nil {
		fatal(err)
	}
	defer f.Close()

	img, format, err := image.Decode(f)
	if err != nil {
		fatal(fmt.Errorf("error decoding image '%s': %w", path, err))
	}
	fmt.Printf("Decoded image format: %s\n", format)
	return img
}

// Simple Box Blur implementation (consider replacing with Gaussian blur for quality)
func boxBlur(src *image.RGBA, radius int) *image.RGBA {
	if radius <= 0 {
		return src // No blur
	}
	bounds := src.Bounds()
	dst := image.NewRGBA(bounds)
	w, h := bounds.Dx(), bounds.Dy()

	// Temporary buffer for horizontal pass
	temp := image.NewRGBA(bounds)

	// --- Horizontal Pass ---
	for y := 0; y < h; y++ {
		var rSum, gSum, bSum, aSum uint32 = 0, 0, 0, 0
		// Initialize sum for the first pixel segment
		for x := -radius; x <= radius; x++ {
			px := clamp(x, 0, w-1) + bounds.Min.X // Use absolute coordinates
			py := y + bounds.Min.Y
			// Use At which returns color.Color, then RGBA()
			pr, pg, pb, pa := src.At(px, py).RGBA()
			rSum += pr
			gSum += pg
			bSum += pb
			aSum += pa
		}

		div := uint32(2*radius + 1)

		for x := 0; x < w; x++ {
			// Convert average uint32 (0-65535 range) back to uint8 (0-255 range)
			temp.SetRGBA(x+bounds.Min.X, y+bounds.Min.Y, color.RGBA{uint8(rSum / div >> 8), uint8(gSum / div >> 8), uint8(bSum / div >> 8), uint8(aSum / div >> 8)})


			// Efficiently update sum: subtract outgoing, add incoming
			outX := clamp(x-radius, 0, w-1) + bounds.Min.X
			inX := clamp(x+radius+1, 0, w-1) + bounds.Min.X
			py := y + bounds.Min.Y

			prOut, pgOut, pbOut, paOut := src.At(outX, py).RGBA()
			rSum -= prOut
			gSum -= pgOut
			bSum -= pbOut
			aSum -= paOut


			prIn, pgIn, pbIn, paIn := src.At(inX, py).RGBA()
			rSum += prIn
			gSum += pgIn
			bSum += pbIn
			aSum += paIn

		}
	}

	// --- Vertical Pass ---
	for x := 0; x < w; x++ {
		var rSum, gSum, bSum, aSum uint32 = 0, 0, 0, 0
		// Initialize sum for the first pixel segment
		for y := -radius; y <= radius; y++ {
			px := x + bounds.Min.X
			py := clamp(y, 0, h-1) + bounds.Min.Y
			// Read from horizontal pass result (temp)
			pr, pg, pb, pa := temp.At(px, py).RGBA()
			rSum += pr
			gSum += pg
			bSum += pb
			aSum += pa
		}

		div := uint32((radius*2 + 1))


		for y := 0; y < h; y++ {
			// Convert average back to uint8
			dst.SetRGBA(x+bounds.Min.X, y+bounds.Min.Y, color.RGBA{uint8(rSum / div >> 8), uint8(gSum / div >> 8), uint8(bSum / div >> 8), uint8(aSum / div >> 8)})


			// Update sum
			outY := clamp(y-radius, 0, h-1) + bounds.Min.Y
			inY := clamp(y+radius+1, 0, h-1) + bounds.Min.Y
			px := x + bounds.Min.X

			prOut, pgOut, pbOut, paOut := temp.At(px, outY).RGBA()
			rSum -= prOut
			gSum -= pgOut
			bSum -= pbOut
			aSum -= paOut


			prIn, pgIn, pbIn, paIn := temp.At(px, inY).RGBA()
			rSum += prIn
			gSum += pgIn
			bSum += pbIn
			aSum += paIn

		}
	}

	return dst
}

// Helper for blur calculation
func clamp(val, minVal, maxVal int) int {
	if val < minVal {
		return minVal
	}
	if val > maxVal {
		return maxVal
	}
	return val
}

// stitchFinalTiles creates a single image by combining the final generated tiles.
func stitchFinalTiles(outputPath string, tiles [][]image.Image, rows, cols int) {
	if rows == 0 || cols == 0 || len(tiles) != rows || len(tiles[0]) != cols {
		fmt.Fprintf(os.Stderr, "Error: Invalid tile data for stitching.\n")
		return
	}

	tileW := finalTileW
	tileH := finalTileH
	margin := 1 // 1-pixel margin between tiles

	totalW := cols*tileW + (cols-1)*margin
	totalH := rows*tileH + (rows-1)*margin

	if totalW <= 0 || totalH <= 0 {
		fmt.Fprintf(os.Stderr, "Error: Invalid dimensions for stitched image (%dx%d).\n", totalW, totalH)
		return
	}

	stitchedImage := image.NewRGBA(image.Rect(0, 0, totalW, totalH))
	// Fill background with white for the margins
	draw.Draw(stitchedImage, stitchedImage.Bounds(), image.White, image.Point{}, draw.Src)

	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			tile := tiles[r][c]
			if tile == nil || tile.Bounds().Empty() {
				fmt.Fprintf(os.Stderr, "Warning: Skipping missing/empty tile at row %d, col %d for stitching.\n", r, c)
				continue
			}

			destX := c * (tileW + margin)
			destY := r * (tileH + margin)
			destRect := image.Rect(destX, destY, destX+tileW, destY+tileH)

			// Use draw.Over instead of draw.Src here. While functionally similar for opaque
			// sources on an opaque background, draw.Over is the standard for composing layers
			// and might handle edge cases slightly differently in some graphics libraries or viewers.
			// It's less likely to be the cause, but worth standardizing.
			draw.Draw(stitchedImage, destRect, tile, image.Point{0, 0}, draw.Over)
		}
	}

	// Add tile numbers overlay (optional)
	addTileNumbersOverlay(stitchedImage, rows, cols, tileW, tileH, margin)

	saveJPEG(outputPath, stitchedImage)
}

// Optional: Adds numbers to the stitched preview
func addTileNumbersOverlay(dst *image.RGBA, rows, cols, tileW, tileH, margin int) {
	numTiles := rows * cols
	textColor := image.Black // Use black text
	bgColor := color.RGBA{R: 255, G: 255, B: 255, A: 180} // Semi-transparent white background for text

	d := &font.Drawer{
		Dst:  dst,
		Src:  textColor,
		Face: basicfont.Face7x13,
		Dot:  fixed.Point26_6{},
	}

	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			tileNumber := numTiles - (r*cols + c)
			text := fmt.Sprintf("%d", tileNumber)

			centerX := c*(tileW+margin) + tileW/2
			centerY := r*(tileH+margin) + tileH/2

			textWidth := d.MeasureString(text).Ceil()
			textHeight := d.Face.Metrics().Height.Ceil()

			// Calculate background rectangle for the text
			bgPadding := 3
			bgX0 := centerX - textWidth/2 - bgPadding
			bgY0 := centerY - textHeight/2 - bgPadding
			bgX1 := centerX + textWidth/2 + bgPadding
			bgY1 := centerY + textHeight/2 + bgPadding
			bgRect := image.Rect(bgX0, bgY0, bgX1, bgY1)

			// Draw text background
			draw.Draw(dst, bgRect, &image.Uniform{bgColor}, image.Point{}, draw.Over)

			// Position and draw text
			startX := centerX - textWidth/2
			startY := centerY + textHeight/2 // Adjust for font baseline
			d.Dot = fixed.P(startX, startY)
			d.DrawString(text)
		}
	}
}


func saveJPEG(path string, img image.Image) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		fatal(fmt.Errorf("failed to create directory for %s: %w", path, err))
	}

	f, err := os.Create(path)
	if err != nil {
		fatal(fmt.Errorf("failed to create file %s: %w", path, err))
	}
	defer f.Close()

	// Convert to RGBA if necessary before encoding
	var imgToEncode image.Image = img
	if _, ok := img.(*image.RGBA); !ok {
		fmt.Printf("Converting image for JPEG encoding: %s\n", path)
		bounds := img.Bounds()
		rgbaImg := image.NewRGBA(bounds)
		draw.Draw(rgbaImg, bounds, img, bounds.Min, draw.Src)
		imgToEncode = rgbaImg
	}


	if err := jpeg.Encode(f, imgToEncode, &jpeg.Options{Quality: jpegQuality}); err != nil {
		fatal(fmt.Errorf("failed to encode JPEG %s: %w", path, err))
	}
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	if strings.Contains(err.Error(), "flag") {
		fmt.Fprintf(os.Stderr, "\nUsage:\n")
		flag.PrintDefaults()
	}
	os.Exit(1)
}