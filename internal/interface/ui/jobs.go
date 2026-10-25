package ui

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"downloader/internal/domain"
	"downloader/internal/usecase"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type jobActions struct {
	cancel  func(id int)
	retry   func(job domain.Job)
	reveal  func(path string)
	open    func(path string)
	details func(err error)
}

// jobList shows the queue: one card per job, newest at the bottom.
type jobList struct {
	items    []usecase.JobStatus
	index    map[int]int // job ID -> position in items
	actions  jobActions
	onChange func() // after any change, e.g. to update the summary line

	list  *widget.List
	empty fyne.CanvasObject
	root  fyne.CanvasObject
}

func newJobList(a jobActions) *jobList {

	l := &jobList{index: map[int]int{}, actions: a}

	l.list = widget.NewList(
		func() int { return len(l.items) },
		func() fyne.CanvasObject { return newJobRow() },
		func(i widget.ListItemID, o fyne.CanvasObject) { o.(*jobRow).set(l.items[i], l.actions) },
	)
	l.list.HideSeparators = true
	l.list.OnSelected = func(widget.ListItemID) { l.list.UnselectAll() } // rows are not selectable

	icon := widget.NewIcon(theme.NewDisabledResource(theme.DownloadIcon()))
	l.empty = container.NewCenter(container.NewVBox(
		container.NewCenter(container.NewGridWrap(fyne.NewSize(56, 56), icon)),
		widget.NewLabelWithStyle("Nothing here yet", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		tight(lowCenter("Your downloads and texts will show up here.")),
		tight(lowCenter("You can drop files anywhere on the window.")),
	))

	l.root = container.NewStack(l.list, l.empty)
	return l
}

func lowCenter(text string) *widget.Label {
	lbl := widget.NewLabelWithStyle(text, fyne.TextAlignCenter, fyne.TextStyle{})
	lbl.Importance = widget.LowImportance
	return lbl
}

func (l *jobList) object() fyne.CanvasObject { return l.root }

func (l *jobList) update(st usecase.JobStatus) {

	if i, ok := l.index[st.ID]; ok {
		l.items[i] = st
		l.list.RefreshItem(i)
	} else {
		l.index[st.ID] = len(l.items)
		l.items = append(l.items, st)
		l.list.Refresh()
		l.list.ScrollToBottom()
	}

	l.changed()
}

func (l *jobList) clearFinished() {

	kept := l.items[:0]
	for _, st := range l.items {
		if !st.State.Finished() {
			kept = append(kept, st)
		}
	}
	l.items = kept

	l.index = map[int]int{}
	for i, st := range l.items {
		l.index[st.ID] = i
	}

	l.list.Refresh()
	l.changed()
}

func (l *jobList) changed() {
	if len(l.items) == 0 {
		l.empty.Show()
	} else {
		l.empty.Hide()
	}
	if l.onChange != nil {
		l.onChange()
	}
}

// summary is the line under the queue title, e.g. "1 working · 2 waiting · 3 done".
func (l *jobList) summary() string {

	var working, waiting, done, failed int
	for _, st := range l.items {
		switch st.State {
		case usecase.Downloading, usecase.Transcribing:
			working++
		case usecase.Waiting:
			waiting++
		case usecase.Done:
			done++
		case usecase.Failed:
			failed++
		}
	}

	var parts []string
	add := func(n int, word string) {
		if n > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", n, word))
		}
	}
	add(working, "working")
	add(waiting, "waiting")
	add(done, "done")
	add(failed, "failed")

	if len(parts) == 0 {
		return "Jobs run one after another"
	}
	return strings.Join(parts, " · ")
}

type jobRow struct {
	widget.BaseWidget

	badge      *iconBadge
	title      *widget.Label
	detail     *widget.Label
	chip       *chip
	bar        *thinBar
	cancelBtn  *widget.Button
	retryBtn   *widget.Button
	revealBtn  *widget.Button
	openBtn    *widget.Button
	detailsBtn *widget.Button

	row     *fyne.Container
	content fyne.CanvasObject
}

func newJobRow() *jobRow {

	r := &jobRow{
		badge:      newIconBadge(38, 20),
		title:      label("", theme.SizeNameText, true, widget.MediumImportance),
		detail:     label("", theme.SizeNameCaptionText, false, widget.LowImportance),
		chip:       newChip(),
		bar:        newThinBar(),
		cancelBtn:  smallButton("Cancel", theme.CancelIcon()),
		retryBtn:   smallButton("Retry", theme.ViewRefreshIcon()),
		revealBtn:  smallButton("Show", theme.FolderOpenIcon()),
		openBtn:    smallButton("Open text", theme.DocumentIcon()),
		detailsBtn: smallButton("Why?", theme.QuestionIcon()),
	}
	r.title.Truncation = fyne.TextTruncateEllipsis
	r.detail.Truncation = fyne.TextTruncateEllipsis

	// line 1: title ........ chip
	// line 2: detail ....... buttons
	// line 3: progress
	buttons := container.NewHBox(r.detailsBtn, r.retryBtn, r.openBtn, r.revealBtn, r.cancelBtn)
	line1 := container.NewBorder(nil, nil, nil, container.NewCenter(r.chip), tight(r.title))
	line2 := container.NewBorder(nil, nil, nil, buttons, tight(r.detail))
	text := container.NewVBox(line1, line2, container.New(layout.NewCustomPaddedLayout(2, 0, 0, 0), r.bar))

	r.row = container.NewBorder(nil, nil, container.NewCenter(r.badge), nil,
		container.New(layout.NewCustomPaddedLayout(0, 0, 4, 0), text))
	box := container.NewStack(
		newPanel(named(colorRaised), named(colorSurfaceBorder), 12),
		container.New(layout.NewCustomPaddedLayout(8, 8, 12, 10), r.row),
	)
	r.content = container.New(layout.NewCustomPaddedLayout(3, 3, 0, 4), box)

	r.ExtendBaseWidget(r)
	return r
}

func smallButton(text string, icon fyne.Resource) *widget.Button {
	b := widget.NewButtonWithIcon(text, icon, nil)
	b.Importance = widget.LowImportance
	return b
}

func (r *jobRow) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(r.content)
}

func (r *jobRow) set(st usecase.JobStatus, a jobActions) {

	look := lookFor(st)

	r.badge.set(look.icon, look.color)
	r.title.SetText(titleFor(st))
	r.detail.SetText(look.detail)
	r.chip.set(look.chip, look.color, look.importance)

	switch st.State {
	case usecase.Downloading, usecase.Transcribing:
		r.bar.SetValue(st.Percent / 100) // Unknown (-1) animates
	default:
		r.bar.SetIdle()
	}

	showIf(r.cancelBtn, !st.State.Finished())
	showIf(r.retryBtn, st.State == usecase.Failed || st.State == usecase.Canceled)
	showIf(r.detailsBtn, st.State == usecase.Failed)
	showIf(r.openBtn, st.State == usecase.Done && st.Result.TextPath != "")
	showIf(r.revealBtn, st.State == usecase.Done && st.Result.MediaPath != "")

	r.cancelBtn.OnTapped = func() { a.cancel(st.ID) }
	r.retryBtn.OnTapped = func() { a.retry(st.Job) }
	r.detailsBtn.OnTapped = func() { a.details(st.Err) }
	r.openBtn.OnTapped = func() { a.open(st.Result.TextPath) }
	r.revealBtn.OnTapped = func() { a.reveal(st.Result.MediaPath) }

	// buttons were shown or hidden: lay the row out again
	r.row.Refresh()
}

// look is how one state is shown: icon, color, chip text and the detail line.
type look struct {
	icon       fyne.Resource
	color      fyne.ThemeColorName
	importance widget.Importance
	chip       string
	detail     string
}

func lookFor(st usecase.JobStatus) look {

	switch st.State {
	case usecase.Waiting:
		return look{theme.HistoryIcon(), theme.ColorNamePlaceHolder, widget.LowImportance,
			"Waiting", "Starts after the jobs above"}

	case usecase.Downloading:
		return look{theme.NewPrimaryThemedResource(theme.DownloadIcon()), theme.ColorNamePrimary, widget.HighImportance,
			"Downloading" + percent(st.Percent), "Getting the file from " + host(st.Job.URL)}

	case usecase.Transcribing:
		detail := "Turning speech into text"
		if st.Job.Options.Translate {
			detail += ", then into English"
		}
		return look{theme.NewPrimaryThemedResource(theme.DocumentCreateIcon()), theme.ColorNamePrimary, widget.HighImportance,
			"Writing text" + percent(st.Percent), detail}

	case usecase.Done:
		detail := "Saved in " + shortPath(filepath.Dir(st.Result.MediaPath))
		if st.Result.TextPath != "" {
			detail = "Text saved: " + filepath.Base(st.Result.TextPath)
		}
		return look{theme.NewSuccessThemedResource(theme.ConfirmIcon()), theme.ColorNameSuccess, widget.SuccessImportance,
			"Done", detail}

	case usecase.Failed:
		return look{theme.NewErrorThemedResource(theme.ErrorIcon()), theme.ColorNameError, widget.DangerImportance,
			"Failed", friendlyError(st.Err)}

	default:
		return look{theme.NewDisabledResource(theme.CancelIcon()), theme.ColorNamePlaceHolder, widget.LowImportance,
			"Stopped", "You stopped this job"}
	}
}

func titleFor(st usecase.JobStatus) string {
	if st.Result.MediaPath != "" {
		return filepath.Base(st.Result.MediaPath)
	}
	if st.Job.FilePath != "" {
		return filepath.Base(st.Job.FilePath)
	}
	return prettyLink(st.Job.URL)
}

// prettyLink: "https://www.youtube.com/watch?v=abc" -> "youtube.com · abc".
func prettyLink(link string) string {
	u, err := url.Parse(link)
	if err != nil || u.Host == "" {
		return link
	}
	if id := u.Query().Get("v"); id != "" {
		return host(link) + " · " + id
	}
	if u.Path != "" && u.Path != "/" {
		return host(link) + " · " + u.Path
	}
	return host(link)
}

func host(link string) string {
	u, err := url.Parse(link)
	if err != nil || u.Host == "" {
		return "the site"
	}
	return strings.TrimPrefix(u.Host, "www.")
}

func percent(p float64) string {
	if p < 0 {
		return "…"
	}
	return fmt.Sprintf(" %.0f%%", p)
}

func showIf(o fyne.CanvasObject, show bool) {
	if show {
		o.Show()
	} else {
		o.Hide()
	}
}
