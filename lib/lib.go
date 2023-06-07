package lib

import (
	"errors"
)

type DownloadedTiktokInfo struct {
	Filename  string
	Username  string
	Timestamp int64
	Source    string
}

var TikwmFailed = errors.New("tikwm failed for some reason")

func DownloadVideo(link string, directory ...string) (*DownloadedTiktokInfo, error) {
	info, tikwmErr := DownloadTikwm(link, directory...)
	if tikwmErr == nil {
		return info, nil
	}

	info, err := DownloadYtDlp(link, directory...)
	if err != nil {
		return nil, err
	}
	return info, TikwmFailed
}
