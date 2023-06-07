package lib

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path"
)

const YtDlpSign = "yt-dlp"
const YtDlpPath = "yt-dlp"

func DownloadYtDlp(url string, directory ...string) (*DownloadedTiktokInfo, error) {
	type TiktokResponse struct {
		Id                 string `json:"id"`
		Title              string `json:"title,omitempty"`
		Description        string `json:"description,omitempty"`
		Uploader           string `json:"uploader"`
		Creator            string `json:"creator"`
		Track              string `json:"track,omitempty"`
		Artist             string `json:"artist,omitempty"`
		Timestamp          int64  `json:"timestamp"`
		RequestedDownloads []struct {
			Filename string `json:"_filename"`
		} `json:"requested_downloads"`
	}

	cmd := exec.Command(YtDlpPath, "-J", "--no-simulate", "--no-progress", url)
	if len(directory) > 0 {
		cmd.Dir = directory[0]
	}
	output, err := cmd.Output()
	if err != nil {
		return nil, errors.New(fmt.Sprintf("%s\n%s", err.Error(), string(output)))
	}

	var resp TiktokResponse
	err = json.Unmarshal(output, &resp)
	if err != nil {
		return nil, err
	}

	if len(resp.RequestedDownloads) != 1 {
		return nil, errors.New(fmt.Sprintf("len(TiktokResponse.RequestedDownloads) != 1, but = %d", len(resp.RequestedDownloads)))
	}

	return &DownloadedTiktokInfo{
		Filename:  path.Join(cmd.Dir, resp.RequestedDownloads[0].Filename),
		Username:  resp.Creator,
		Timestamp: resp.Timestamp,
		Source:    YtDlpSign,
	}, nil
}
