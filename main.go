package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"runtime"

	"github.com/chromedp/chromedp"
)

const (
	maxNumberOfTabs       = 100
	maxNumberOfGoroutines = 5
	defaultBrowser        = "https://www.google.com/"
	dataFile              = "data.txt"
)

func main() {
	fmt.Println("start -> open -> save -> exit")

	var ctx context.Context
	for {
		var command string
		fmt.Scanf("%s", &command)
		switch command {
		case "start":
			ctx = start()
		case "open":
			run(ctx, open)
		case "save":
			run(ctx, save)
		case "exit":
			_ = exit(ctx)
			os.Exit(1)
		}
	}
}

func run(ctx context.Context, f func(ctx context.Context)) {
	if ctx != nil {
		f(ctx)
	}
}

func exit(ctx context.Context) error {
	if ctx != nil {
		err := chromedp.FromContext(ctx).Browser.Process().Kill()
		return err
	}
	return nil
}

func start() context.Context {

	user, err := user.Current()
	if err != nil {
		log.Fatalf(err.Error())
	}

	homeDir := user.HomeDir

	userDataDir := ""

	switch runtime.GOOS {
	case "windows":
		userDataDir = filepath.Dir(homeDir + "/AppData/Local/Google/Chrome/User Data/")
	case "darwin":
		userDataDir = filepath.Dir(homeDir + "/AppData/Local/Google/Chrome/User Data/")
	case "linux":
		userDataDir = ""
	}

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		//chromedp.UserDataDir("C:\\Users\\Administrator\\AppData\\Local\\Google\\Chrome\\User Data"),
		chromedp.UserDataDir(userDataDir),
		chromedp.Flag("headless", false),
		chromedp.UserAgent("Mozilla/5.0 (X11; Linux x8664) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/97.0.4692.71 Safari/537.36"),
		chromedp.Flag("enable-automation", false),
		chromedp.Flag("restore-on-startup", false),
		chromedp.Flag("password-store", "basic"),
		chromedp.Flag("remote-debugging-port", "9222"),
		chromedp.Flag("mute-audio", false),
		chromedp.Flag("new-window", true),
	)

	allocCtx, _ := chromedp.NewExecAllocator(context.Background(), opts...)
	//allocCtx, _ := chromedp.NewRemoteAllocator(context.Background(), opts...)
	parentCtx, _ := chromedp.NewContext(allocCtx)
	if err := chromedp.Run(parentCtx, chromedp.Navigate(defaultBrowser)); err != nil {
		log.Println(err)
	}
	return parentCtx
}

func open(parentCtx context.Context) {
	f, err := os.Open(dataFile)

	if err != nil {
		log.Println(err)
	}

	defer f.Close()

	scanner := bufio.NewScanner(f)

	ch := make(chan string, maxNumberOfTabs)

	for i := 0; i < maxNumberOfGoroutines; i++ {
		go func() {
			for v := range ch {
				if v != defaultBrowser {
					tabCtx, _ := chromedp.NewContext(parentCtx)
					if err := chromedp.Run(tabCtx, chromedp.Navigate(v)); err != nil {
						log.Println(err)
					}
				}
			}
		}()
	}

	for scanner.Scan() {
		ch <- scanner.Text()
	}

}

func save(parentCtx context.Context) {
	if err := os.Truncate(dataFile, 0); err != nil {
		log.Printf("Failed to truncate: %v", err)
	}

	infos, _ := chromedp.Targets(parentCtx)

	ch := make(chan string, maxNumberOfTabs)

	for i := 0; i < maxNumberOfGoroutines; i++ {
		go func() {
			for v := range ch {
				f, err := os.OpenFile(dataFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0660)
				if err != nil {
					log.Fatal(err)
				}
				_, err = f.WriteString(v + "\n")

				if err != nil {
					log.Println(err)
				}

				f.Close()
			}
		}()
	}

	for _, v := range infos {
		if v.Type == "page" {
			ch <- v.URL
		}
	}

}
