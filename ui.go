package main

import (
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/sirupsen/logrus"
)

func ShowSettingsWindow(notifier *Notifier) {
	minMedianRatingBindStr := binding.BindString(new(string))
	bf := binding.NewFloat()
	bf.Set(float64(notifier.Settings.MinMedianRating))
	bf.AddListener(binding.NewDataListener(func() {
		v, _ := bf.Get()
		notifier.Settings.MinMedianRating = v
		minMedianRatingBindStr.Set(strconv.Itoa(int(notifier.Settings.MinMedianRating)) +
			" " + ratingToRank(float64(notifier.Settings.MinMedianRating)))
	}))

	w := fyne.CurrentApp().NewWindow("Settings")
	w.Resize(fyne.Size{Width: 300, Height: 300})
	buttons := container.NewAdaptiveGrid(2,
		widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), func() {
			w.Hide()
		}),
		widget.NewButtonWithIcon("Save", theme.ConfirmIcon(), func() {
			if err := notifier.saveSettings(); err != nil {
				logrus.WithFields(logrus.Fields{
					"error": err,
				}).Error("could not save settings")
				dialog.ShowError(err, w)
			} else {
				w.Hide()
			}
		}),
	)

	w.SetContent(container.NewBorder(nil, buttons, nil, nil,
		container.NewAppTabs(
			container.NewTabItem("OGS", container.NewVBox(
				widget.NewLabel("Min median dan rating"),
				widget.NewSliderWithData(1900, 2700, bf),
				container.NewCenter(widget.NewLabelWithData(minMedianRatingBindStr)),
				widget.NewSeparator(),
				// widget.NewLabel("Custom match RegExp"),
				// widget.NewEntryWithData(binding.BindString(&customMatchRE)),
				widget.NewCheckWithData("Pro games", binding.BindBool(&notifier.Settings.ProGames)),
				widget.NewCheckWithData("Bot games", binding.BindBool(&notifier.Settings.BotGames)),
			)),
			container.NewTabItem("Other", container.NewVBox(
				widget.NewLabel("Not implemented"),
			)),
		),
	))
	notifier.OpenWindow <- w
}
