package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/mattn/go-runewidth"
	"golang.org/x/term"
)

// GetTerminalWidth returns the terminal width, or 120 if detection fails
func GetTerminalWidth() int {
	// Try to get terminal width
	fd := int(os.Stdout.Fd())
	width, _, err := term.GetSize(fd)

	// Return default value if detection fails or width is unreasonable
	if err != nil || width < 40 {
		return 120
	}

	return width
}

// TruncateMiddle intelligently truncates a string, keeping front and back parts with "..." in the middle
// Considers CJK character display width, keeping approximately 60% front and 40% back
func TruncateMiddle(s string, maxWidth int) string {
	// Calculate actual display width of the string
	currentWidth := runewidth.StringWidth(s)

	// Return as-is if no truncation needed
	if currentWidth <= maxWidth {
		return s
	}

	// Return ellipsis if max width is too small for effective truncation
	if maxWidth < 10 {
		return "..."
	}

	// Extract file extension (if it's a filename)
	ext := filepath.Ext(s)
	nameWithoutExt := strings.TrimSuffix(s, ext)

	// Calculate display width of extension
	extWidth := runewidth.StringWidth(ext)

	// Ellipsis takes 3 character widths
	ellipsisWidth := 3

	// Available width for filename body
	availableWidth := maxWidth - ellipsisWidth - extWidth

	// If available width is too small, only keep extension
	if availableWidth < 5 {
		// Try to show partial extension
		if maxWidth > ellipsisWidth {
			return "..." + truncateToWidth(ext, maxWidth-ellipsisWidth)
		}
		return "..."
	}

	// Keep 60% front, 40% back
	frontWidth := int(float64(availableWidth) * 0.6)
	backWidth := availableWidth - frontWidth

	// Truncate front part
	frontPart := truncateToWidth(nameWithoutExt, frontWidth)

	// Truncate back part (from end)
	backPart := truncateFromEnd(nameWithoutExt, backWidth)

	return frontPart + "..." + backPart + ext
}

// truncateToWidth truncates a string to specified display width (from front to back)
func truncateToWidth(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}

	width := 0
	runes := []rune(s)

	for i, r := range runes {
		w := runewidth.RuneWidth(r)
		if width+w > maxWidth {
			return string(runes[:i])
		}
		width += w
	}

	return s
}

// truncateFromEnd truncates a string to specified display width from the end
func truncateFromEnd(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}

	runes := []rune(s)
	width := 0

	// Iterate from back to front
	for i := len(runes) - 1; i >= 0; i-- {
		w := runewidth.RuneWidth(runes[i])
		if width+w > maxWidth {
			return string(runes[i+1:])
		}
		width += w
	}

	return s
}

// FormatDownloadStatus formats download status with fixed-width progress on the left
// filename: filename to display
// bytesComplete: bytes downloaded
// bytesTotal: total bytes
// progress: progress ratio (0.0 to 1.0)
func FormatDownloadStatus(filename string, bytesComplete, bytesTotal int64, progress float64) string {
	// Get terminal width
	termWidth := GetTerminalWidth()

	// Add safety margin for emoji and special characters
	safetyMargin := 10
	effectiveWidth := termWidth - safetyMargin

	// Format bytes with fixed width (right-aligned)
	// Max display is "9999 GB" = 7 chars, so we use 8 chars for safety
	bytesCompleteStr := humanize.Bytes(uint64(bytesComplete))
	bytesTotalStr := humanize.Bytes(uint64(bytesTotal))

	// Format progress info with fixed width format: [percentage] size/total
	// Example: [ 26.90%]   76 MB/  284 MB
	progressInfo := fmt.Sprintf("[%6.2f%%] %8s/%8s",
		progress*100,
		bytesCompleteStr,
		bytesTotalStr)

	// Fixed width for progress section (9 + 1 + 8 + 1 + 8 = 27 chars)
	progressWidth := 27

	// Spacing between progress and filename
	spacing := 2

	// Calculate maximum width available for filename
	maxFilenameWidth := effectiveWidth - progressWidth - spacing

	// Use a reasonable minimum if available width is too small
	if maxFilenameWidth < 10 {
		maxFilenameWidth = 10
	}

	// Truncate filename if needed
	displayFilename := TruncateMiddle(filename, maxFilenameWidth)

	// Build final status line: progress + spacing + filename
	result := progressInfo + strings.Repeat(" ", spacing) + displayFilename

	return result
}

// CompressPath compresses directory names in a path (fish shell style)
// e.g., /aaa/bbb/ccc/file.mp4 -> /a/b/c/file.mp4
// Supports both Unix (/) and Windows (\) path separators
func CompressPath(path string, maxWidth int) string {
	// Calculate current display width
	currentWidth := runewidth.StringWidth(path)

	// If path fits, return as-is
	if currentWidth <= maxWidth {
		return path
	}

	// Detect path separator (Windows uses \, Unix uses /)
	separator := "/"
	if strings.Contains(path, "\\") {
		separator = "\\"
	}

	// Split path into components
	parts := strings.Split(path, separator)
	if len(parts) == 0 {
		return path
	}

	// Keep the filename (last part) intact
	filename := parts[len(parts)-1]
	dirParts := parts[:len(parts)-1]

	// Compress directory names to first character (or first rune for CJK)
	compressedParts := make([]string, len(dirParts))
	for i, part := range dirParts {
		if part == "" {
			// Empty part (e.g., leading / in Unix paths)
			compressedParts[i] = part
		} else if strings.HasSuffix(part, ":") {
			// Windows drive letter (e.g., C:, D:) - keep as-is
			compressedParts[i] = part
		} else {
			// Take first rune
			runes := []rune(part)
			if len(runes) > 0 {
				compressedParts[i] = string(runes[0])
			} else {
				compressedParts[i] = part
			}
		}
	}

	// Rebuild path with compressed directories
	compressedPath := strings.Join(compressedParts, separator)
	if compressedPath != "" && !strings.HasSuffix(compressedPath, ":") {
		compressedPath += separator
	} else if strings.HasSuffix(compressedPath, ":") {
		// For Windows drive letters, add separator after colon
		compressedPath += separator
	}
	compressedPath += filename

	// Check if compressed path fits
	compressedWidth := runewidth.StringWidth(compressedPath)
	if compressedWidth <= maxWidth {
		return compressedPath
	}

	// If still too long, truncate the filename part
	// Calculate available width for filename
	prefixWidth := runewidth.StringWidth(compressedPath) - runewidth.StringWidth(filename)
	availableFilenameWidth := maxWidth - prefixWidth

	if availableFilenameWidth < 10 {
		// If very little space, just truncate the whole path
		return TruncateMiddle(path, maxWidth)
	}

	// Truncate filename and rebuild
	truncatedFilename := TruncateMiddle(filename, availableFilenameWidth)

	// Rebuild with compressed dirs and truncated filename
	result := ""
	if len(compressedParts) > 0 {
		result = strings.Join(compressedParts, separator)
		if result != "" && !strings.HasSuffix(result, ":") {
			result += separator
		} else if strings.HasSuffix(result, ":") {
			result += separator
		}
	}
	result += truncatedFilename

	return result
}

// FormatCompletionMessage formats download completion message with path truncation
// filepath: full path to the downloaded file
func FormatCompletionMessage(filepath string) string {
	// Get terminal width
	termWidth := GetTerminalWidth()

	// Add safety margin for emoji and special characters
	safetyMargin := 5
	effectiveWidth := termWidth - safetyMargin

	// Prefix "Download saved to "
	prefix := "Download saved to "
	prefixWidth := runewidth.StringWidth(prefix)

	// Calculate maximum width available for filepath
	maxPathWidth := effectiveWidth - prefixWidth

	// Use a reasonable minimum if available width is too small
	if maxPathWidth < 20 {
		maxPathWidth = 20
	}

	// Compress and truncate filepath if needed
	displayPath := CompressPath(filepath, maxPathWidth)

	// Build final message
	result := prefix + displayPath

	// Final safety check
	resultWidth := runewidth.StringWidth(result)
	if resultWidth > effectiveWidth {
		// Recalculate with more aggressive truncation
		maxPathWidth = maxPathWidth - (resultWidth - effectiveWidth) - 5
		if maxPathWidth < 10 {
			maxPathWidth = 10
		}
		displayPath = CompressPath(filepath, maxPathWidth)
		result = prefix + displayPath
	}

	return result
}
