package lib

import (
	"encoding/json"
	"errors"
	"fmt"
	tele "github.com/mehanon/telebot"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

var Debug = false
var ConversionThreshold int64 = 20_000_000 // 20MB

type Metadata struct {
	Streams []struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"streams"`
	Format struct {
		Filename string `json:"filename"`
		Duration string `json:"duration"`
	} `json:"format"`
}

func GetMetadata(filename string) (*Metadata, error) {
	output, err := exec.Command("ffprobe", "-v", "error", "-select_streams", "v:0", "-show_entries", "stream=width,height",
		"-of", "json", "-show_format", filename).Output()
	if err != nil {
		return nil, errors.New(fmt.Sprintf("%v\n%s", err, string(output)))
	}

	var metadata Metadata
	err = json.Unmarshal(output, &metadata)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("%v\n%s", err, string(output)))
	}

	return &metadata, nil
}

func formatDuration(d time.Duration) string {
	trailingZeros := func(d time.Duration, zeros int) string {
		num := int64(d)
		s := fmt.Sprintf("%d", num)
		for len(s) < zeros {
			s = "0" + s
		}
		return s
	}

	return fmt.Sprintf("%s:%s:%s.%s",
		trailingZeros(d/time.Hour%24, 2), trailingZeros(d/time.Minute%60, 2),
		trailingZeros(d/time.Second%60, 2), trailingZeros(d/time.Millisecond%1000, 3))
}

func SendVideoGracefully(ctx tele.Context, filename string, telegramFilename string, progressMessage *tele.Message) error {
	metadata, err := GetMetadata(filename)
	if err != nil {
		return err
	}
	thumbnailBigFileName := fmt.Sprintf("%s_big.png", filename)
	duration, _ := strconv.ParseFloat(metadata.Format.Duration, 64)
	output, err := exec.Command("ffmpeg", "-i", filename, "-ss",
		formatDuration(time.Duration(duration*float64(time.Second/2))),
		"-vframes", "1", thumbnailBigFileName).Output()
	if err != nil {
		return errors.New(fmt.Sprintf("%v\n%s", err, string(output)))
	}
	thumbnailFileName := fmt.Sprintf("%s.jpg", filename)
	output, err = exec.Command("convert", thumbnailBigFileName, "-resize", "320x320", thumbnailFileName).Output()
	if err != nil {
		return errors.New(fmt.Sprintf("%v\n%s", err, string(output)))
	}

	for retryNumber := 0; retryNumber < 4; retryNumber++ {
		_, err = ctx.Bot().Send(
			ctx.Chat(),
			&tele.Video{
				File:     tele.FromDisk(filename),
				Width:    metadata.Streams[0].Width,
				Height:   metadata.Streams[0].Height,
				Duration: int(duration),
				Caption:  "",
				Thumbnail: &tele.Photo{
					File: tele.FromDisk(thumbnailFileName),
				},
				Streaming: true,
				MIME:      "video/mp4",
				FileName:  telegramFilename,
			},
			&tele.SendOptions{ReplyTo: ctx.Message(), ParseMode: "MarkdownV2"},
		)
		if err == nil || !strings.Contains(err.Error(), "connection reset by peer") {
			break
		}
		progressMessage, _ = ctx.Bot().Edit(progressMessage, fmt.Sprintf("while uploading an error occured, retry number: %d", retryNumber+1))
	}
	if !Debug {
		go func() {
			_ = os.Remove(thumbnailBigFileName)
			_ = os.Remove(thumbnailFileName)
		}()
	}

	return err
}

func DelayFileDeletion(filename string, delay time.Duration) {
	go func() {
		time.Sleep(delay)
		err := os.Remove(filename)
		if err != nil {
			log.Printf("when deleting %s, an error occured: %s", filename, err.Error())
		}
	}()
}

func CreateDirIfNotFound(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err = os.Mkdir(path, 0777)
		if err != nil {
			return err
		}
	}
	return nil
}

func ConvertH265toH264(filename string) (string, error) {
	stat, err := os.Stat(filename)
	if err != nil {
		return "", err
	} else if stat.Size() > ConversionThreshold {
		return filename, nil
	}

	h264Filename := fmt.Sprintf("%s.h264.mp4", filename)
	_, err = exec.Command("ffmpeg", "-i", filename, "-vcodec", "libx264", "-acodec", "aac", "-y",
		"-preset", "fast", "-metadata", "source_link=t.me/by_meh", h264Filename).Output()
	return h264Filename, err
}
