package lib

import (
	"fmt"
	tele "github.com/mehanon/telebot"
	"path"
	"regexp"
	"strconv"
	"time"
)

type TtTgConfig struct {
	Token       string  `json:"token"`
	AdminList   []int64 `json:"admin-list"`
	DataDirPath string  `json:"data-dir-path"`
	TgURL       string  `json:"tg-url,omitempty"`
	IsLocal     bool    `json:"is-local,omitempty"`
}

type TtTg struct {
	Cfg TtTgConfig
	Bot *tele.Bot
}

func NewTtTg(cfg TtTgConfig, preferences ...tele.Settings) (*TtTg, error) {
	var pref tele.Settings
	if len(preferences) > 0 {
		pref = preferences[0]
	} else {
		pref = tele.Settings{
			Token:   cfg.Token,
			Poller:  &tele.LongPoller{Timeout: 30 * time.Second},
			URL:     cfg.TgURL,
			Verbose: Debug,
			OnError: func(err error, ctx tele.Context) {
				for _, admin := range cfg.AdminList {
					_, _ = ctx.Bot().Send(&tele.Chat{ID: admin},
						fmt.Sprintf("Error :c\n\n	%s\n\nAt chat: '%s' [%d]", err.Error(), ctx.Chat().Title, ctx.Chat().ID))
				}
			},
			Local: cfg.IsLocal,
		}
	}

	err := CreateDirIfNotFound(cfg.DataDirPath)
	if err != nil {
		return nil, err
	}

	bot, err := tele.NewBot(pref)
	if err != nil {
		return nil, err
	}

	urlRegex := regexp.MustCompile("[-a-zA-Z0-9@:%._+~#=]{1,256}\\.[a-zA-Z0-9()]{1,6}\\b([-a-zA-Z0-9()@:%_+.~#?&/=]*)")
	bot.Handle(tele.OnText, func(ctx tele.Context) error {
		progressMessage, err := bot.Reply(ctx.Message(), "ok, wait a sec")
		if err != nil {
			return err
		}
		links := urlRegex.FindAll([]byte(ctx.Message().Text), -1)
		if len(links) == 0 {
			id, err := strconv.ParseInt(ctx.Message().Text, 10, 64)
			if err == nil {
				links = append(links, []byte(fmt.Sprintf("https:/tiktok.com/@share/video/%s", id)))
			}
		}
		println(len(links))
		for _, link := range links {
			ttFileInfo, err := DownloadVideo(string(link), cfg.DataDirPath)
			if err != nil {
				return err
			}
			progressMessage, err = bot.Edit(progressMessage, "downloaded, converting...")
			if err != nil {
				bot.OnError(err, ctx)
			}

			if !Debug {
				DelayFileDeletion(ttFileInfo.Filename, time.Minute*5)
			}

			h264Filename, err := ConvertH265toH264(ttFileInfo.Filename)
			if err != nil {
				bot.OnError(err, ctx)
				h264Filename = ttFileInfo.Filename
				progressMessage, err = bot.Edit(progressMessage, "converting failed, the video will be uploaded as it is")
				if err != nil {
					bot.OnError(err, ctx)
				}
			} else {
				if !Debug {
					DelayFileDeletion(h264Filename, time.Minute*5)
				}
				progressMessage, err = bot.Edit(progressMessage, "converted, uploading...")
				if err != nil {
					bot.OnError(err, ctx)
				}
			}

			err = SendVideoGracefully(ctx, h264Filename,
				fmt.Sprintf("@%s_%s.mp4", ttFileInfo.Username, time.Unix(ttFileInfo.Timestamp, 0).Format("2006-01-02")), progressMessage)
			if err != nil {
				return err
			}
			if ttFileInfo.Source != TikwmSign {
				err := ctx.Reply("Tikwm failed")
				if err != nil {
					return err
				}
			}
		}

		err = bot.Delete(progressMessage)
		if err != nil {
			return err
		}

		return nil
	})

	bot.Handle(tele.OnVideo, func(ctx tele.Context) error {
		progressMessage, err := bot.Reply(ctx.Message(), "ok, wait a sec, downloading...")
		if err != nil {
			return err
		}

		localFilename := path.Join(cfg.DataDirPath, fmt.Sprintf("%d_%d_%s", ctx.Chat().ID, ctx.Message().ID, ctx.Message().Video.FileName))
		err = bot.Download(&ctx.Message().Video.File, localFilename)
		if err != nil {
			return err
		}

		progressMessage, err = bot.Edit(progressMessage, "downloaded, converting...")
		if err != nil {
			bot.OnError(err, ctx)
		}
		defer func() {
			if !Debug {
				DelayFileDeletion(localFilename, time.Minute*5)
			}
		}()
		time.Sleep(time.Millisecond * 100)
		h264Filename, err := ConvertH265toH264(localFilename)
		if err != nil {
			return err
		} else {
			if !Debug {
				DelayFileDeletion(h264Filename, time.Minute*5)
			}
			progressMessage, err = bot.Edit(progressMessage, "converted, uploading...")
			if err != nil {
				bot.OnError(err, ctx)
			}
		}

		err = SendVideoGracefully(ctx, h264Filename, ctx.Message().Video.FileName, progressMessage)
		if err != nil {
			return err
		}

		return bot.Delete(progressMessage)
	})

	return &TtTg{Cfg: cfg, Bot: bot}, nil
}

func (bot *TtTg) Start() {
	bot.Bot.Start()
}
