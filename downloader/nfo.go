package downloader

import (
	"encoding/xml"
	"fmt"
	"io"
	"iwaradl/api"
	"iwaradl/util"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// WriteNfoToPath get video detail info and write nfo to path
func WriteNfoToPath(vi api.VideoInfo, path string) (title string, outPath string, err error) {
	util.DebugLog("Getting detailed info for video: %s", vi.Id)
	detailInfo, err := api.GetDetailInfo(vi)
	if err != nil {
		return "", "", err
	}

	// add <br> to description
	detailInfo.Description = strings.ReplaceAll(detailInfo.Description, "\n", "<br/>\n")

	util.DebugLog("Writing NFO file: %s", path)
	f, err := os.Create(path)
	if err != nil {
		println(err.Error())
		return "", "", err
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)
	// write xml header
	_, err = f.WriteString(xml.Header)
	if err != nil {
		return "", "", err
	}
	// marshal
	b, err := xml.MarshalIndent(detailInfo, "", "  ")
	if err != nil {
		println(err.Error())
		return "", "", err
	}
	_, err = f.Write(b)
	if err != nil {
		return "", "", err
	}

	return detailInfo.VideoName, path, nil
}

// UpdateNfoFiles Update all nfo files in a directory
func UpdateNfoFiles(rootDir string, delay int) {
	util.DebugLog("Start updating nfo files in %s", rootDir)
	nfoFiles := []string{}
	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".nfo") {
			nfoFiles = append(nfoFiles, path)
		}
		return nil
	})

	if err != nil {
		util.DebugLog("Error walking the path %s: %v", rootDir, err)
		println("Error walking the path " + rootDir + ": " + err.Error())
		return
	}

	total := len(nfoFiles)
	util.DebugLog("Found %d nfo files to update.", total)

	for i, nfoPath := range nfoFiles {
		baseName := strings.TrimSuffix(filepath.Base(nfoPath), ".nfo")
		parts := strings.Split(baseName, "-")
		if len(parts) < 2 {
			util.DebugLog("Invalid nfo filename format, skipping: %s", filepath.Base(nfoPath))
			continue
		}
		vid := parts[len(parts)-1]

		fmt.Printf("Updating [%d/%d]: %s (ID: %s)\n", i+1, total, baseName, vid)

		// 1. read and parse nfo file
		xmlFile, err := os.Open(nfoPath)
		if err != nil {
			util.DebugLog("Failed to open nfo file %s: %v", nfoPath, err)
			println("Error: " + err.Error())
			continue
		}

		xmlData, _ := io.ReadAll(xmlFile)
		xmlFile.Close()

		var nfoData api.JellyfinNfo
		err = xml.Unmarshal(xmlData, &nfoData)
		if err != nil {
			util.DebugLog("Failed to unmarshal nfo file %s: %v", nfoPath, err)
			println("Error parsing " + baseName + ": " + err.Error())
			continue
		}

		// 2. Get new video info
		videoInfo, err := api.GetVideoInfo(vid)
		if err != nil {
			util.DebugLog("Failed to get video info for %s: %v", vid, err)
			println("Error: " + err.Error())
			continue
		}

		// 3. Update info in nfo
		nfoData.Title = videoInfo.Title
		nfoData.Director = videoInfo.User.Name
		nfoData.Plot = strings.ReplaceAll(videoInfo.Body, "\n", "<br/>\n")
		nfoData.ReleaseDate = videoInfo.CreatedAt.Format("2006-01-02")
		nfoData.Premiered = nfoData.ReleaseDate
		nfoData.Year = videoInfo.CreatedAt.Format("2006")
		var categories []string
		for _, v := range videoInfo.Tags {
			categories = append(categories, v.Id)
		}
		nfoData.Genre = categories

		// 4. write updated nfo
		updatedXml, err := xml.MarshalIndent(nfoData, "", "  ")
		if err != nil {
			util.DebugLog("Failed to marshal updated nfo for %s: %v", vid, err)
			println("Error: " + err.Error())
			continue
		}

		err = os.WriteFile(nfoPath, []byte(xml.Header+string(updatedXml)), 0644)
		if err != nil {
			util.DebugLog("Failed to write updated nfo for %s: %v", vid, err)
			println("Error: " + err.Error())
			continue
		}

		util.DebugLog("Successfully updated nfo for video ID: %s", vid)

		// 5. Rate Limit
		if i < total-1 {
			util.DebugLog("Waiting for %d seconds before next request...", delay)
			time.Sleep(time.Duration(delay) * time.Second)
		}
	}
	println("NFO update process finished.")
}
