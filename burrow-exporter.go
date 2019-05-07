package main

import (
	"fmt"
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/urfave/cli"

	"context"
	"os/signal"
	"syscall"

	"github.com/jirwin/burrow_exporter/burrow_exporter"
)

var Version = "0.0.5"

func main() {
	app := cli.NewApp()
	app.Version = Version
	app.Name = "burrow-exporter"
	app.Flags = []cli.Flag{
		// burrow 地址
		cli.StringFlag{
			Name:   "burrow-addr",
			Usage:  "Address that burrow is listening on",
			EnvVar: "BURROW_ADDR",
		},
		// exporter 地址，供 prometheus pull metrics
		cli.StringFlag{
			Name:   "metrics-addr",
			Usage:  "Address to run prometheus on",
			EnvVar: "METRICS_ADDR",
		},
		// 抓取周期
		cli.IntFlag{
			Name:   "interval",
			Usage:  "The interval(seconds) specifies how often to scrape burrow.",
			EnvVar: "INTERVAL",
		},
		// API 版本，默认2，现在大部分应该都是3了
		cli.IntFlag{
			Name:   "api-version",
			Usage:  "Burrow API version to leverage",
			Value:  2,
			EnvVar: "API_VERSION",
		},
		// 是否跳过分区状态抓取
		cli.BoolFlag{
			Name:   "skip-partition-status",
			Usage:  "Skip exporting the per-partition status",
			EnvVar: "SKIP_PARTITION_STATUS",
		},
		// 是否跳过group状态抓取
		cli.BoolFlag{
			Name:   "skip-group-status",
			Usage:  "Skip exporting the per-group status",
			EnvVar: "SKIP_GROUP_STATUS",
		},
		cli.BoolFlag{
			Name:   "skip-partition-lag",
			Usage:  "Skip exporting the partition lag",
			EnvVar: "SKIP_PARTITION_LAG",
		},
		cli.BoolFlag{
			Name:   "skip-partition-current-offset",
			Usage:  "Skip exporting the current offset per partition",
			EnvVar: "SKIP_PARTITION_CURRENT_OFFSET",
		},
		cli.BoolFlag{
			Name:   "skip-partition-max-offset",
			Usage:  "Skip exporting the partition max offset",
			EnvVar: "SKIP_PARTITION_MAX_OFFSET",
		},
		cli.BoolFlag{
			Name:   "skip-total-lag",
			Usage:  "Skip exporting the total lag",
			EnvVar: "SKIP_TOTAL_LAG",
		},
		cli.BoolFlag{
			Name:   "skip-topic-partition-offset",
			Usage:  "Skip exporting topic partition offset",
			EnvVar: "SKIP_TOPIC_PARTITION_OFFSET",
		},
	}

	app.Action = func(c *cli.Context) error {
		if !c.IsSet("burrow-addr") {
			fmt.Println("A burrow address is required (e.g. --burrow-addr http://localhost:8000)")
			os.Exit(1)
		}

		if !c.IsSet("metrics-addr") {
			fmt.Println("An address to run prometheus on is required (e.g. --metrics-addr localhost:8080)")
			os.Exit(1)
		}

		if !c.IsSet("interval") {
			fmt.Println("A scrape interval is required (e.g. --interval 30)")
			os.Exit(1)
		}

		done := make(chan os.Signal, 1)

		// notify 监听 处理 SIGINT（中断）和 SIGTERM（终止）信号
		signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)

		// 使用context来关联goroutine
		// 一般创建context根节点使用context.Background(), 函数返回空的Context
		// 有了根节点，通过WithTimeout, WithCancel, WithDeadline, WithValue 这四个函数来创建子节点、孙节点...
		// ref: https://zhuyasen.com/post/golang_context.html
		ctx, cancel := context.WithCancel(context.Background())

		exporter := burrow_exporter.MakeBurrowExporter(
			c.String("burrow-addr"),
			c.Int("api-version"),
			c.String("metrics-addr"),
			c.Int("interval"),
			c.Bool("skip-partition-status"),
			c.Bool("skip-group-status"),
			c.Bool("skip-partition-lag"),
			c.Bool("skip-partition-current-offset"),
			c.Bool("skip-partition-max-offset"),
			c.Bool("skip-total-lag"),
			c.Bool("skip-topic-partition-offset"))
		go exporter.Start(ctx)

		fmt.Println(">>>> starting ......")
		<-done
		// 通过上面信号处理，实现程序优雅退出，下面做一些清理工作

		// 手动取消， 当Context 被 canceled 或是 times out 的时候，Done 返回一个被 closed 的channel
		// 关闭对应的c.done，也就是让它的后代goroutine退出，这里会终止爬取服务
		cancel()
		// 关闭 exporter
		exporter.Close()

		fmt.Println("!!! finished !!!!")
		return nil
	}

	err := app.Run(os.Args)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("error running burrow-exporter")
		os.Exit(1)
	}
}
