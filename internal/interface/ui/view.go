package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"

	"downloader/internal/domain"
	"downloader/internal/usecase"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// WindowSize fits a 13" MacBook screen: inputs on the left, queue on the right.
var WindowSize = fyne.NewSize(1080, 760)

// Deps is what the UI needs from the rest of the app.
type Deps struct {
	Process      *usecase.ProcessUseCase
	ListModels   func() []domain.Model
	ModelsDir    string
	MissingTools []string
	InstallHint  string // e.g. "brew install whisper-cpp"
	Reveal       func(path string) error
	Open         func(path string) error
}

// preference keys, so the app remembers the last choices
const (
	prefMode      = "mode"
	prefSavePath  = "savePath"
	prefLanguage  = "language"
	prefModel     = "model"
	prefTranslate = "translate"
	prefFileDir   = "fileDir"

	prefSettingsOpen = "settingsOpen"
)

// mode is what the user wants from a link.
type mode int

const (
	modeAudio mode = iota
	modeVideo
	modeText // audio + transcription
)

var languages = []struct{ label, code string }{
	{"Auto", domain.LangAuto},
	{"Persian", domain.LangPersian},
	{"English", domain.LangEnglish},
}

var mediaExtensions = []string{
	".mp3", ".m4a", ".wav", ".ogg", ".opus", ".flac", ".aac",
	".mp4", ".mkv", ".webm", ".mov",
}

type View struct {
	app     fyne.App
	win     fyne.Window
	deps    Deps
	prefs   fyne.Preferences
	queue   *usecase.Queue
	closing atomic.Bool
	content fyne.CanvasObject

	// link card
	mode          mode
	tiles         []*modeTile
	links         *widget.Entry
	linksHint     *widget.Label
	savePath      string
	savePathLabel *widget.Label
	startBtn      *widget.Button

	// text settings
	language  *segmented
	model     *widget.Select
	models    []domain.Model
	translate *widget.Check
	modelHint *widget.Label

	settingsBody    *fyne.Container
	settingsToggle  *widget.Button
	settingsSummary *widget.Label

	// queue
	summary *widget.Label
	jobs    *jobList
}

func New(a fyne.App, w fyne.Window, deps Deps) *View {

	v := &View{app: a, win: w, deps: deps, prefs: a.Preferences()}

	v.jobs = newJobList(jobActions{
		cancel:  func(id int) { v.queue.Cancel(id) },
		retry:   func(job domain.Job) { v.add(job) },
		reveal:  func(path string) { v.run(v.deps.Reveal, path) },
		open:    func(path string) { v.run(v.deps.Open, path) },
		details: func(err error) { v.showDetails(err) },
	})
	v.queue = usecase.NewQueue(deps.Process, v.onJobChange)
	v.content = v.build()

	w.SetOnDropped(v.onDrop)
	w.SetCloseIntercept(v.confirmClose)

	return v
}

func (v *View) Content() fyne.CanvasObject { return v.content }

// Shutdown stops the running job (and its programs) before the app exits.
func (v *View) Shutdown() {
	v.closing.Store(true)
	v.queue.Close()
}

func (v *View) build() fyne.CanvasObject {

	left := container.NewVBox(v.buildHeader())
	if banner := v.buildToolsBanner(); banner != nil {
		left.Add(banner)
	}
	left.Add(v.buildLinkCard())
	left.Add(v.buildFileCard())
	left.Add(v.buildSettingsCard())

	v.reloadModels()
	v.setMode(mode(v.prefs.IntWithFallback(prefMode, int(modeAudio))))

	leftCol := container.NewVScroll(container.New(layout.NewCustomPaddedLayout(14, 14, 20, 10), left))
	rightCol := container.New(layout.NewCustomPaddedLayout(14, 14, 10, 20), v.buildQueueCard())

	return container.NewGridWithColumns(2, leftCol, rightCol)
}

func (v *View) buildHeader() fyne.CanvasObject {

	badge := newIconBadge(46, 26)
	badge.set(theme.NewPrimaryThemedResource(theme.DownloadIcon()), theme.ColorNamePrimary)

	title := label("Downloader", theme.SizeNameHeadingText, true, widget.MediumImportance)
	sub := label("Save audio or video from a link, and turn speech into text.", theme.SizeNameText, false, widget.LowImportance)
	sub.Wrapping = fyne.TextWrapWord

	return container.New(layout.NewCustomPaddedLayout(0, 2, 0, 0),
		container.NewBorder(nil, nil, container.NewCenter(badge), nil, container.NewVBox(tight(title), tight(sub))))
}

func (v *View) buildLinkCard() fyne.CanvasObject {

	v.tiles = []*modeTile{
		newModeTile(theme.MediaMusicIcon(), "Audio", "mp3 file", func() { v.setMode(modeAudio) }),
		newModeTile(theme.MediaVideoIcon(), "Video", "video file", func() { v.setMode(modeVideo) }),
		newModeTile(theme.DocumentCreateIcon(), "Text", "audio + its text", func() { v.setMode(modeText) }),
	}
	tiles := container.NewGridWithColumns(3, v.tiles[0], v.tiles[1], v.tiles[2])

	v.links = widget.NewMultiLineEntry()
	v.links.SetPlaceHolder("One link per line — YouTube, SoundCloud and 1000+ sites")
	v.links.SetMinRowsVisible(3)
	v.links.OnChanged = func(string) { v.refreshState() }

	v.linksHint = label("", theme.SizeNameCaptionText, false, widget.LowImportance)
	v.linksHint.Truncation = fyne.TextTruncateEllipsis

	v.savePath = v.prefs.StringWithFallback(prefSavePath, defaultSavePath())
	v.savePathLabel = label(shortPath(v.savePath), theme.SizeNameText, true, widget.MediumImportance)
	v.savePathLabel.Truncation = fyne.TextTruncateEllipsis
	changeBtn := widget.NewButtonWithIcon("Change", theme.FolderOpenIcon(), v.chooseFolder)
	changeBtn.Importance = widget.LowImportance
	saveRow := container.NewBorder(nil, nil,
		label("Save to", theme.SizeNameText, false, widget.LowImportance), changeBtn, v.savePathLabel)

	v.startBtn = widget.NewButtonWithIcon("Download", theme.DownloadIcon(), v.startDownload)
	v.startBtn.Importance = widget.HighImportance

	return card(container.NewVBox(
		stepTitle("1", "What do you want?"),
		tiles,
		stepTitle("2", "Paste links"),
		v.links,
		tight(v.linksHint),
		saveRow,
		v.startBtn,
	))
}

func (v *View) buildFileCard() fyne.CanvasObject {

	icon := newIconBadge(42, 24)
	icon.set(theme.NewPrimaryThemedResource(theme.UploadIcon()), theme.ColorNamePrimary)

	title := label("Already have a file?", theme.SizeNameText, true, widget.MediumImportance)
	sub := label("Drop it anywhere on this window to get its text.", theme.SizeNameCaptionText, false, widget.LowImportance)
	sub.Truncation = fyne.TextTruncateEllipsis

	chooseBtn := widget.NewButtonWithIcon("Choose file", theme.FolderOpenIcon(), v.chooseFile)

	body := container.NewBorder(nil, nil, container.NewCenter(icon), container.NewCenter(chooseBtn),
		container.NewVBox(tight(title), tight(sub)))

	return container.NewStack(
		newPanel(tint(theme.ColorNamePrimary, 0x12), tint(theme.ColorNamePrimary, 0x70), 16),
		container.New(layout.NewCustomPaddedLayout(12, 12, 16, 16), body),
	)
}

func (v *View) buildSettingsCard() fyne.CanvasObject {

	labels := make([]string, len(languages))
	for i, l := range languages {
		labels[i] = l.label
	}
	v.language = newSegmented(labels)
	v.language.SetSelected(v.prefs.StringWithFallback(prefLanguage, languages[0].label))
	v.language.OnChanged = func(s string) {
		v.prefs.SetString(prefLanguage, s)
		v.updateSettingsSummary()
	}

	v.model = widget.NewSelect(nil, func(s string) {
		v.prefs.SetString(prefModel, s)
		v.refreshState()
	})
	refreshBtn := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), v.reloadModels)
	refreshBtn.Importance = widget.LowImportance
	folderBtn := widget.NewButtonWithIcon("", theme.FolderOpenIcon(), v.openModelsFolder)
	folderBtn.Importance = widget.LowImportance

	v.translate = widget.NewCheck("Also make an English version", nil)
	v.translate.SetChecked(v.prefs.Bool(prefTranslate))
	v.translate.OnChanged = func(on bool) {
		v.prefs.SetBool(prefTranslate, on)
		v.updateSettingsSummary()
	}

	v.modelHint = label("No model found. Put a ggml-*.bin file in the models folder (folder button), then press refresh.",
		theme.SizeNameCaptionText, false, widget.DangerImportance)
	v.modelHint.Wrapping = fyne.TextWrapWord
	v.modelHint.Hide()

	title := label("Text settings", theme.SizeNameSubHeadingText, true, widget.MediumImportance)
	v.settingsSummary = label("", theme.SizeNameCaptionText, false, widget.LowImportance)
	v.settingsSummary.Truncation = fyne.TextTruncateEllipsis

	sub := label("Used for “Text” and for files. Text is saved next to the file (.txt, .srt).",
		theme.SizeNameCaptionText, false, widget.LowImportance)
	sub.Wrapping = fyne.TextWrapWord

	form := widget.NewForm(
		widget.NewFormItem("Language", v.language),
		widget.NewFormItem("Model", container.NewBorder(nil, nil, nil, container.NewHBox(refreshBtn, folderBtn), v.model)),
		widget.NewFormItem("", v.translate),
	)

	// closed by default: a one-line summary is enough most of the time
	v.settingsBody = container.NewVBox(sub, form, v.modelHint)
	v.settingsToggle = widget.NewButton("", func() { v.openSettings(!v.settingsBody.Visible()) })
	v.openSettings(v.prefs.Bool(prefSettingsOpen))

	header := container.NewBorder(nil, nil, nil, container.NewCenter(v.settingsToggle),
		container.NewVBox(tight(title), tight(v.settingsSummary)))

	return card(container.NewVBox(header, v.settingsBody))
}

func (v *View) openSettings(open bool) {
	showIf(v.settingsBody, open)
	if open {
		v.settingsToggle.SetText("Done")
		v.settingsToggle.SetIcon(theme.MenuDropUpIcon())
	} else {
		v.settingsToggle.SetText("Change")
		v.settingsToggle.SetIcon(theme.MenuDropDownIcon())
	}
	v.prefs.SetBool(prefSettingsOpen, open)
}

func (v *View) updateSettingsSummary() {
	model := v.model.Selected
	if model == "" {
		model = "none"
	}
	english := "off"
	if v.translate.Checked {
		english = "on"
	}
	v.settingsSummary.SetText(fmt.Sprintf("Language %s · Model %s · English version %s", v.language.selected, model, english))
}

func (v *View) buildQueueCard() fyne.CanvasObject {

	title := label("Queue", theme.SizeNameSubHeadingText, true, widget.MediumImportance)
	v.summary = label("", theme.SizeNameCaptionText, false, widget.LowImportance)
	v.jobs.onChange = func() { v.summary.SetText(v.jobs.summary()) }
	v.summary.SetText(v.jobs.summary())

	clearBtn := smallButton("Clear finished", theme.ContentClearIcon())
	clearBtn.OnTapped = v.jobs.clearFinished
	stopBtn := smallButton("Stop all", theme.MediaStopIcon())
	stopBtn.OnTapped = v.queue.CancelAll

	header := container.NewBorder(nil, nil, container.NewVBox(tight(title), tight(v.summary)),
		container.NewCenter(container.NewHBox(clearBtn, stopBtn)))

	return card(container.NewBorder(container.New(layout.NewCustomPaddedLayout(0, 8, 0, 0), header), nil, nil, nil, v.jobs.object()))
}

func (v *View) buildToolsBanner() fyne.CanvasObject {

	if len(v.deps.MissingTools) == 0 {
		return nil
	}

	msg := label(fmt.Sprintf("Missing: %s. Install in Terminal, then reopen the app:", strings.Join(v.deps.MissingTools, ", ")),
		theme.SizeNameText, true, widget.WarningImportance)
	msg.Wrapping = fyne.TextWrapWord

	cmd := widget.NewLabelWithStyle(v.deps.InstallHint, fyne.TextAlignLeading, fyne.TextStyle{Monospace: true})
	cmd.Selectable = true
	copyBtn := widget.NewButtonWithIcon("Copy", theme.ContentCopyIcon(), func() {
		v.app.Clipboard().SetContent(v.deps.InstallHint)
	})

	icon := newIconBadge(36, 20)
	icon.set(theme.NewWarningThemedResource(theme.WarningIcon()), theme.ColorNameWarning)

	body := container.NewBorder(nil, nil, container.NewCenter(icon), nil,
		container.NewVBox(msg, container.NewBorder(nil, nil, nil, copyBtn, cmd)))

	return container.NewStack(
		newPanel(tint(theme.ColorNameWarning, 0x1c), tint(theme.ColorNameWarning, 0x80), 16),
		container.New(layout.NewCustomPaddedLayout(10, 10, 14, 14), body),
	)
}

func (v *View) setMode(m mode) {
	if m < modeAudio || m > modeText {
		m = modeAudio
	}
	v.mode = m
	v.prefs.SetInt(prefMode, int(m))
	for i, t := range v.tiles {
		t.setSelected(mode(i) == m)
	}
	v.refreshState()
}

// refreshState updates hints, the start button and the model warning after any change.
func (v *View) refreshState() {

	if v.startBtn == nil || v.model == nil {
		return // still building
	}

	hasModels := len(v.models) > 0
	showIf(v.modelHint, !hasModels)
	if hasModels {
		v.model.Enable()
	} else {
		v.model.Disable()
		v.openSettings(true) // show the warning, do not hide it in a closed card
	}
	v.updateSettingsSummary()

	links, invalid := usecase.ParseLinks(v.links.Text)
	needModel := v.mode == modeText && v.selectedModel() == nil

	switch {
	case len(invalid) > 0:
		v.setHint("This is not a link: "+invalid[0], widget.DangerImportance)
	case len(links) == 0:
		v.setHint("Tip: paste many links at once. They download one after another.", widget.LowImportance)
	case needModel:
		v.setHint("“Text” needs a model. See Text settings below.", widget.WarningImportance)
	case len(links) == 1:
		v.setHint("1 link ready", widget.SuccessImportance)
	default:
		v.setHint(fmt.Sprintf("%d links ready", len(links)), widget.SuccessImportance)
	}

	v.startBtn.SetText(startLabel(v.mode, len(links)))
	if len(links) > 0 && len(invalid) == 0 && !needModel {
		v.startBtn.Enable()
	} else {
		v.startBtn.Disable()
	}
}

func startLabel(m mode, n int) string {
	switch {
	case m == modeVideo && n > 1:
		return fmt.Sprintf("Download %d videos", n)
	case m == modeVideo:
		return "Download video"
	case m == modeText && n > 1:
		return fmt.Sprintf("Download %d and get text", n)
	case m == modeText:
		return "Download and get text"
	case n > 1:
		return fmt.Sprintf("Download %d audio files", n)
	default:
		return "Download audio"
	}
}

func (v *View) setHint(text string, imp widget.Importance) {
	v.linksHint.Text = text
	v.linksHint.Importance = imp
	v.linksHint.Refresh()
}

func (v *View) reloadModels() {

	v.models = v.deps.ListModels()

	names := make([]string, len(v.models))
	for i, m := range v.models {
		names[i] = m.Name
	}
	v.model.SetOptions(names)

	switch {
	case len(names) == 0:
		v.model.PlaceHolder = "No model found"
		v.model.ClearSelected()
	case slices.Contains(names, v.prefs.String(prefModel)):
		v.model.SetSelected(v.prefs.String(prefModel))
	default:
		v.model.SetSelected(names[0])
	}

	v.refreshState()
}

func (v *View) selectedModel() *domain.Model {
	for i := range v.models {
		if v.models[i].Name == v.model.Selected {
			return &v.models[i]
		}
	}
	return nil
}

func (v *View) transcribeOptions() (domain.TranscribeOptions, bool) {

	m := v.selectedModel()
	if m == nil {
		return domain.TranscribeOptions{}, false
	}

	lang := domain.LangAuto
	for _, l := range languages {
		if l.label == v.language.selected {
			lang = l.code
		}
	}

	return domain.TranscribeOptions{Language: lang, ModelPath: m.Path, Translate: v.translate.Checked}, true
}

func (v *View) startDownload() {

	links, invalid := usecase.ParseLinks(v.links.Text)
	if len(links) == 0 || len(invalid) > 0 {
		return
	}

	job := domain.Job{SavePath: v.savePath, Format: domain.Audio}
	switch v.mode {
	case modeVideo:
		job.Format = domain.Video
	case modeText:
		opts, ok := v.transcribeOptions()
		if !ok {
			v.showNoModel()
			return
		}
		job.Transcribe = true
		job.Options = opts
	}

	for _, link := range links {
		job.URL = link
		if !v.add(job) {
			return
		}
	}

	v.links.SetText("")
}

func (v *View) addFiles(paths []string) {

	opts, ok := v.transcribeOptions()
	if !ok {
		v.showNoModel()
		return
	}

	for _, p := range paths {
		if !v.add(domain.Job{FilePath: p, Transcribe: true, Options: opts}) {
			return
		}
	}
}

func (v *View) add(job domain.Job) bool {
	if _, err := v.queue.Add(job); err != nil {
		dialog.ShowError(err, v.win)
		return false
	}
	return true
}

func (v *View) chooseFolder() {

	d := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
		if err != nil {
			dialog.ShowError(err, v.win)
			return
		}
		if uri == nil {
			return
		}
		v.savePath = uri.Path()
		v.prefs.SetString(prefSavePath, v.savePath)
		v.savePathLabel.SetText(shortPath(v.savePath))
	}, v.win)

	if dir, err := storage.ListerForURI(storage.NewFileURI(v.savePath)); err == nil {
		d.SetLocation(dir)
	}
	d.Resize(fyne.NewSize(720, 520))
	d.Show()
}

func (v *View) chooseFile() {

	d := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil {
			dialog.ShowError(err, v.win)
			return
		}
		if reader == nil {
			return
		}
		path := reader.URI().Path()
		reader.Close()

		v.prefs.SetString(prefFileDir, filepath.Dir(path))
		v.addFiles([]string{path})
	}, v.win)

	d.SetFilter(storage.NewExtensionFileFilter(mediaExtensions))
	if dir, err := storage.ListerForURI(storage.NewFileURI(v.prefs.StringWithFallback(prefFileDir, defaultSavePath()))); err == nil {
		d.SetLocation(dir)
	}
	d.Resize(fyne.NewSize(720, 520))
	d.Show()
}

// onDrop takes files dropped anywhere on the window.
func (v *View) onDrop(_ fyne.Position, uris []fyne.URI) {

	var files []string
	skipped := 0
	for _, u := range uris {
		if u.Scheme() == "file" && isMedia(u.Path()) {
			files = append(files, u.Path())
		} else {
			skipped++
		}
	}

	if len(files) > 0 {
		v.addFiles(files)
	}
	if skipped > 0 {
		dialog.ShowInformation("Some items skipped",
			fmt.Sprintf("%d item(s) are not audio or video files.", skipped), v.win)
	}
}

func (v *View) onJobChange(st usecase.JobStatus) {

	if v.closing.Load() {
		return
	}

	// the queue calls this from its own goroutine
	fyne.Do(func() {
		v.jobs.update(st)

		switch st.State {
		case usecase.Done:
			v.app.SendNotification(fyne.NewNotification("Done", titleFor(st)))
		case usecase.Failed:
			v.app.SendNotification(fyne.NewNotification("Failed", titleFor(st)+": "+friendlyError(st.Err)))
		}
	})
}

func (v *View) confirmClose() {

	if !v.queue.Busy() {
		v.win.Close()
		return
	}

	dialog.ShowConfirm("Quit Downloader?", "A job is still running. Quitting will stop it.",
		func(quit bool) {
			if quit {
				v.win.Close()
			}
		}, v.win)
}

func (v *View) showDetails(err error) {
	if err == nil {
		return
	}
	text := widget.NewLabel(err.Error())
	text.Wrapping = fyne.TextWrapWord
	text.Selectable = true

	scroll := container.NewVScroll(text)
	scroll.SetMinSize(fyne.NewSize(560, 220))

	d := dialog.NewCustom("What went wrong", "Close",
		container.NewBorder(tight(label(friendlyError(err), theme.SizeNameText, true, widget.DangerImportance)), nil, nil, nil, scroll),
		v.win)
	d.Show()
}

func (v *View) showNoModel() {

	d := dialog.NewConfirm("No text model",
		"Put a ggml-*.bin model in the models folder, then press refresh next to Model.",
		func(open bool) {
			if open {
				v.openModelsFolder()
			}
		}, v.win)
	d.SetConfirmText("Open models folder")
	d.SetDismissText("Close")
	d.Show()
}

func (v *View) openModelsFolder() {
	os.MkdirAll(v.deps.ModelsDir, 0o755)
	v.run(v.deps.Open, v.deps.ModelsDir)
}

func (v *View) run(fn func(string) error, path string) {
	if err := fn(path); err != nil {
		dialog.ShowError(err, v.win)
	}
}

func isMedia(path string) bool {
	return slices.Contains(mediaExtensions, strings.ToLower(filepath.Ext(path)))
}

func defaultSavePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, "Downloads")
}

// shortPath shows the home folder as ~.
func shortPath(path string) string {
	home, err := os.UserHomeDir()
	if err == nil && strings.HasPrefix(path, home) {
		return "~" + strings.TrimPrefix(path, home)
	}
	return path
}
