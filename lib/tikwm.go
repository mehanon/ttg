package lib

import (
	"github.com/mehanon/tikmeh/tikwm"
	"os"
	"path"
)

const TikwmSign = "tikwm"

func DownloadTikwm(link string, directory ...string) (*DownloadedTiktokInfo, error) {
	dir := "."
	if len(directory) > 0 {
		dir = directory[0]
	}

	info, filename, err := tikwm.DownloadTiktokVerbose(link)
	if err != nil {
		return nil, err
	}
	if dir != "." {
		err = os.Rename(filename, path.Join(dir, filename))
		if err != nil {
			return nil, err
		}
	}

	return &DownloadedTiktokInfo{
		Filename:  path.Join(dir, filename),
		Username:  info.Author.Username,
		Timestamp: info.CreateTime,
		Source:    TikwmSign,
	}, nil
}
