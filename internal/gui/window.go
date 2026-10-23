package gui

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/qrave1/MusicPort/internal/spotify"
	"github.com/qrave1/MusicPort/internal/syncer"
)

// Window owns all UI state; background jobs publish updates through fyne.Do.
type Window struct {
	app                        fyne.App
	window                     fyne.Window
	service                    Service
	tracks, filtered           []syncer.Track
	collections                []spotify.Collection
	selected                   string
	cancel                     context.CancelFunc
	busy                       bool
	status                     *widget.Label
	list                       *widget.List
	search                     *widget.Entry
	selectCollection           *widget.Select
	login, refresh, load, stop *widget.Button
	link                       *widget.Hyperlink
	progress                   *widget.ProgressBarInfinite
}

func New(a fyne.App, service Service) *Window {
	g := &Window{app: a, window: a.NewWindow("MusicPort"), service: service, selected: "liked", collections: []spotify.Collection{{ID: "liked", Name: "Мне нравится"}}}
	g.status = widget.NewLabel("Войдите в Spotify или загрузите коллекцию с сохранённым входом.")
	g.status.Wrapping = fyne.TextWrapWord
	g.link = widget.NewHyperlink("Открыть страницу входа", nil)
	g.link.Hide()
	g.progress = widget.NewProgressBarInfinite()
	g.progress.Hide()
	g.list = widget.NewList(func() int { return len(g.filtered) }, func() fyne.CanvasObject { return widget.NewLabel("") }, func(i widget.ListItemID, o fyne.CanvasObject) {
		t := g.filtered[i]
		o.(*widget.Label).SetText(fmt.Sprintf("%d. %s   [%s]", i+1, t.Title, t.ID))
	})
	g.search = widget.NewEntry()
	g.search.SetPlaceHolder("Поиск по названию или Spotify ID")
	g.search.OnChanged = func(string) { g.filter() }
	g.selectCollection = widget.NewSelect([]string{"Мне нравится"}, func(label string) {
		for i, c := range g.collections {
			if g.label(i, c) == label {
				g.selected = c.ID
				break
			}
		}
	})
	g.selectCollection.SetSelected("Мне нравится")
	g.login = widget.NewButton("Войти в Spotify", g.authorize)
	g.refresh = widget.NewButton("Обновить плейлисты", g.refreshCollections)
	g.load = widget.NewButton("Загрузить треки", g.loadTracks)
	g.stop = widget.NewButton("Отмена", func() {
		if g.cancel != nil {
			g.cancel()
		}
	})
	g.stop.Disable()
	toolbar := container.NewVBox(container.NewHBox(g.login, g.refresh, g.load, g.stop), g.selectCollection, g.search, g.link, g.progress)
	g.window.SetContent(container.NewBorder(toolbar, container.NewVBox(widget.NewSeparator(), g.status, widget.NewLabel("Источник аудио пока не подключён. Здесь отображается коллекция Spotify.")), nil, nil, g.list))
	g.window.Resize(fyne.NewSize(950, 650))
	g.window.SetOnClosed(func() {
		if g.cancel != nil {
			g.cancel()
		}
	})
	return g
}
func (g *Window) ShowAndRun() { g.window.ShowAndRun() }
func (g *Window) label(i int, c spotify.Collection) string {
	if c.ID == "liked" {
		return "Мне нравится"
	}
	return fmt.Sprintf("%s [%s]", c.Name, c.ID)
}
func (g *Window) filter() {
	q := strings.ToLower(g.search.Text)
	g.filtered = nil
	for _, t := range g.tracks {
		if strings.Contains(strings.ToLower(t.Title+" "+t.ID), q) {
			g.filtered = append(g.filtered, t)
		}
	}
	g.list.Refresh()
}
func (g *Window) start(message string, work func(context.Context) (func(), error)) {
	if g.busy {
		return
	}
	g.busy = true
	ctx, cancel := context.WithCancel(context.Background())
	g.cancel = cancel
	g.login.Disable()
	g.refresh.Disable()
	g.load.Disable()
	g.selectCollection.Disable()
	g.stop.Enable()
	g.progress.Show()
	g.progress.Start()
	g.status.SetText(message)
	go func() {
		apply, err := work(ctx)
		fyne.Do(func() {
			cancel()
			g.cancel = nil
			g.busy = false
			g.login.Enable()
			g.refresh.Enable()
			g.load.Enable()
			g.selectCollection.Enable()
			g.stop.Disable()
			g.progress.Stop()
			g.progress.Hide()
			if err != nil {
				g.status.SetText("Ошибка: " + err.Error())
				return
			}
			if apply != nil {
				apply()
			}
		})
	}()
}
func (g *Window) authorize() {
	g.start("Ожидаем подтверждение в браузере…", func(ctx context.Context) (func(), error) {
		err := g.service.Login(ctx, func(raw string) {
			fyne.Do(func() {
				u, e := url.Parse(raw)
				if e != nil {
					return
				}
				g.link.SetURL(u)
				g.link.Show()
				if e = g.app.OpenURL(u); e != nil {
					g.status.SetText("Откройте страницу входа по ссылке в окне.")
				}
			})
		})
		return func() {
			g.link.Hide()
			g.status.SetText("Вход выполнен. Нажмите «Загрузить треки» или обновите плейлисты.")
		}, err
	})
}
func (g *Window) refreshCollections() {
	g.start("Получаем плейлисты…", func(ctx context.Context) (func(), error) {
		items, err := g.service.Collections(ctx)
		return func() {
			g.collections = items
			labels := make([]string, len(items))
			for i, c := range items {
				labels[i] = g.label(i, c)
			}
			g.selectCollection.SetOptions(labels)
			g.selectCollection.SetSelected("Мне нравится")
			g.selected = "liked"
			g.status.SetText(fmt.Sprintf("Коллекций: %d", len(items)))
		}, err
	})
}
func (g *Window) loadTracks() {
	id := g.selected
	g.start("Получаем треки…", func(ctx context.Context) (func(), error) {
		tracks, err := g.service.Tracks(ctx, id)
		return func() {
			g.tracks = tracks
			g.filter()
			g.status.SetText(fmt.Sprintf("Загружено треков: %d", len(tracks)))
		}, err
	})
}
