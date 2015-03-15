package main

import (
	"flag"
	"github.com/bndr/gojenkins"
	ui "github.com/gizak/termui"
	"github.com/golang/glog"
	tm "github.com/nsf/termbox-go"
	"time"
)

func init() {
	flag.Parse()
}

var sampleInterval = flag.Duration("interval", 5*time.Second, "Interval between sampling (default:5s)")
var jenkinsUrl = flag.String("jenkinsUrl", "", "Jenkins Url")

// TODO add new parameter to filter jobs

func main() {
	defer glog.Flush()
	flag.Parse()
	glog.Info("Starting Jenkins Term")

	err := ui.Init()
	if err != nil {
		panic(err)
	}
	defer ui.Close()

	jenkins := gojenkins.CreateJenkins(*jenkinsUrl).Init()
	ls, p, redbox, yellowbox, greenbox := initWidgets()

	evt := make(chan tm.Event)
	go func() {
		for {
			evt <- tm.PollEvent()
		}
	}()

	ticker := time.NewTicker(*sampleInterval).C
	for {
		select {
		case e := <-evt:
			if e.Type == tm.EventKey && e.Ch == 'q' {
				return
			}
		case <-ticker:
			jenkins.Poll()
			ls.Items = ls.Items[:0]
			// TODO remove reset to avoid blink
			resetBox(redbox, yellowbox, greenbox)
			for _, k := range jenkins.GetAllJobs() {
				addJob(ls, k, redbox, yellowbox, greenbox)
				computeSizes(ls, redbox, yellowbox, greenbox)
				ui.Render(ls, p, redbox, yellowbox, greenbox)
			}
		}
	}
}

func resetBox(redbox *ui.Par, yellowbox *ui.Par, greenbox *ui.Par) {
	redbox.BgColor = ui.ColorBlack
	yellowbox.BgColor = ui.ColorBlack
	greenbox.BgColor = ui.ColorBlack
}

func addJob(list *ui.List, job *gojenkins.Job, redbox *ui.Par, yellowbox *ui.Par, greenbox *ui.Par) {
	str := job.GetName() + " " + job.GetLastBuild().GetResult()
	switch job.GetLastBuild().GetResult() {
	case "SUCCESS":
		greenbox.BgColor = ui.ColorGreen
	case "WARNING":
		yellowbox.BgColor = ui.ColorYellow
	case "FAILURE":
		redbox.BgColor = ui.ColorRed

	}
	list.Items = append(list.Items, str)
}

func computeSizes(list *ui.List, redbox *ui.Par, yellowbox *ui.Par, greenbox *ui.Par) {
	w, h := tm.Size()
	list.Width = int(w * 75 / 100)
	list.Height = h - 3

	redbox.Height = 4
	redbox.Width = 15
	redbox.X = w - 15
	redbox.Y = 3

	yellowbox.Height = 4
	yellowbox.Width = 15
	yellowbox.X = w - 15
	yellowbox.Y = 8

	greenbox.Height = 4
	greenbox.Width = 15
	greenbox.X = w - 15
	greenbox.Y = 13

}

// TODO make new widget traffic light

func initWidgets() (*ui.List, *ui.Par, *ui.Par, *ui.Par, *ui.Par) {
	ui.UseTheme("Jenkins Term UI")

	p := ui.NewPar("q to quit - http://192.168.59.103:10000")
	w, _ := tm.Size()
	p.Height = 3
	p.Width = w
	p.TextFgColor = ui.ColorWhite
	p.Border.Label = "Go Jenkins Dashboard"
	p.Border.FgColor = ui.ColorCyan

	ls := ui.NewList()
	ls.ItemFgColor = ui.ColorYellow
	ls.Border.Label = "Jobs"
	ls.Y = 3

	redbox := ui.NewPar("")
	redbox.Border.Label = "Failure"
	redbox.BgColor = ui.ColorRed

	yellowbox := ui.NewPar("")
	yellowbox.Border.Label = "Warning"
	yellowbox.BgColor = ui.ColorYellow

	greenbox := ui.NewPar("")
	greenbox.Border.Label = "Success"
	greenbox.BgColor = ui.ColorGreen

	ui.Render(ls, p, redbox, yellowbox, greenbox)
	return ls, p, redbox, yellowbox, greenbox
}
