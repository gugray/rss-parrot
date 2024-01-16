package logic

import (
	"bufio"
	"os"
	"rss_parrot/shared"
	"strings"
)

type IBlockedFeeds interface {
	IsBlocked(feedUrl string) (bool, error)
}

type blockedFeeds struct {
	cfg *shared.Config
}

func NewBlockedFeeds(cfg *shared.Config) IBlockedFeeds {
	return &blockedFeeds{cfg}
}

func (bf *blockedFeeds) IsBlocked(feedUrl string) (bool, error) {

	feedUrl = strings.ToLower(feedUrl)
	feedUrl = strings.TrimPrefix(feedUrl, "https://")
	feedUrl = strings.TrimPrefix(feedUrl, "http://")
	readFile, err := os.Open(bf.cfg.BlockedFeedsFile)
	if err != nil {
		return false, err
	}
	defer readFile.Close()
	fileScanner := bufio.NewScanner(readFile)
	fileScanner.Split(bufio.ScanLines)

	for fileScanner.Scan() {
		line := fileScanner.Text()
		if feedUrl == line {
			return true, nil
		}
	}
	return false, nil
}
