package main

import (
	"flag"
	"fmt"
	"regexp"
	"time"

	"github.com/bndr/gojenkins"
	ui "github.com/gizak/termui"
	"github.com/golang/glog"
	tm "github.com/nsf/termbox-go"
)

func init() {
	flag.Parse()
}

var sampleInterval = flag.Duration("interval", 5*time.Second, "Interval between sampling (default:5s)")
var jenkinsUrl = flag.String("jenkinsUrl", "", "Jenkins Url")
var filter = flag.String("filter", "", "Filter job")

var filterBuildName *regexp.Regexp

func main() {
	defer glog.Flush()
	flag.Parse()

	err := ui.Init()
	if err != nil {
		panic(err)
	}
	defer ui.Close()

	jenkins, err := gojenkins.CreateJenkins(*jenkinsUrl).Init()
	if err != nil {
		panic(err)
	}
	ls, infobox, redbox, yellowbox, greenbox := initWidgets()

	if *filter != "" {
		filterBuildName = regexp.MustCompile(*filter)
	}

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
			// alway resize, strange behaviour with tm.EventResize
			resizeUI(ls)
			ls.Items = ls.Items[:0]
			resetBox(infobox, redbox, yellowbox, greenbox)
			jenkinsPoll(jenkins, infobox, ls, redbox, yellowbox, greenbox)
			ui.Render(ui.Body)
		}
	}
}

func resizeUI(ls *ui.List) {
	w, h := tm.Size()
	ui.Body.Width = w
	ls.Height = h - 6
	ui.Body.Align()
}

func jenkinsPoll(jenkins *gojenkins.Jenkins, infobox *ui.Par, ls *ui.List, redbox *ui.Par, yellowbox *ui.Par, greenbox *ui.Par) {
	defer func() {
		if r := recover(); r != nil {
			infobox.Border.FgColor = ui.ColorRed
			//err := fmt.Errorf("%v", r)
			infobox.Text += " : /!\\ Jenkins is currently unreachable"
		}
	}()
	const layout = "Mon Jan 2 15:04:05"
	infobox.Border.FgColor = ui.ColorWhite
	infobox.Text = "Refresh at " + time.Now().Format(layout)
	jenkins.Poll()
	jobs, err := jenkins.GetAllJobs()
	if err != nil {
		infobox.Text = "Error with getAllJobs " + fmt.Sprintf("%s", err)
	}
	for _, k := range jobs {
		addJob(ls, k, redbox, yellowbox, greenbox)
	}
}

func resetBox(infobox *ui.Par, redbox *ui.Par, yellowbox *ui.Par, greenbox *ui.Par) {
	redbox.BgColor = ui.ColorBlack
	yellowbox.BgColor = ui.ColorBlack
	greenbox.BgColor = ui.ColorBlack
}

func addJob(list *ui.List, job *gojenkins.Job, redbox *ui.Par, yellowbox *ui.Par, greenbox *ui.Par) {
	if filterBuildName == nil || (filterBuildName != nil && filterBuildName.MatchString(job.GetName())) {
		str := job.GetName()
		lastBuild, _ := job.GetLastBuild()
		if lastBuild != nil {
			isRunning, _ := job.IsRunning()
			if isRunning {
				str = "...building " + str
			}
			str += " " + " " + lastBuild.GetResult()
			switch lastBuild.GetResult() {
			case "SUCCESS":
				greenbox.BgColor = ui.ColorGreen
			case "UNSTABLE":
				yellowbox.BgColor = ui.ColorYellow
			case "FAILURE":
				redbox.BgColor = ui.ColorRed
			}
		}
		list.Items = append(list.Items, str)
	}
}

// TODO make new widget traffic light
// Waiting for canvas from termui
func initWidgets() (*ui.List, *ui.Par, *ui.Par, *ui.Par, *ui.Par) {
	ui.UseTheme("Jenkins Term UI")

	title := "q to quit - " + *jenkinsUrl
	if *filter != "" {
		title += " filter on " + *filter
	}
	p := ui.NewPar(title)
	_, h := tm.Size()
	p.Height = 3
	p.TextFgColor = ui.ColorWhite
	p.Border.Label = "Go Jenkins Dashboard"
	p.Border.FgColor = ui.ColorCyan

	info := ui.NewPar("")
	info.Height = 3
	info.Y = h - 3
	info.TextFgColor = ui.ColorWhite
	info.Border.FgColor = ui.ColorWhite

	ls := ui.NewList()
	ls.ItemFgColor = ui.ColorYellow
	ls.Border.Label = "Jobs"
	ls.Y = 3
	ls.Height = h - 6

	width, height := 4, 5
	redbox, yellowbox, greenbox := ui.NewPar(""), ui.NewPar(""), ui.NewPar("")
	redbox.HasBorder, yellowbox.HasBorder, greenbox.HasBorder = false, false, false
	redbox.Height, yellowbox.Height, greenbox.Height = height, height, height
	redbox.Width, yellowbox.Width, greenbox.Width = width, width, width
	redbox.BgColor = ui.ColorRed
	yellowbox.BgColor = ui.ColorYellow
	greenbox.BgColor = ui.ColorGreen

	ui.Body.AddRows(
		ui.NewRow(
			ui.NewCol(12, 0, p),
		),
		ui.NewRow(
			ui.NewCol(10, 0, ls),
			ui.NewCol(2, 0, redbox, yellowbox, greenbox),
		),
		ui.NewRow(
			ui.NewCol(12, 0, info),
		),
	)
	ui.Body.Align()
	ui.Render(ui.Body)
	return ls, info, redbox, yellowbox, greenbox
}
