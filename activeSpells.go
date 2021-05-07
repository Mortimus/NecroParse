package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	everquest "github.com/Mortimus/goEverquest"
)

var active map[string]*[]ActiveSpell

func init() {
	active = make(map[string]*[]ActiveSpell)
}

type ActiveSpell struct {
	id       int
	name     string
	ticks    int
	maxTicks int
	ui       *fyne.Container
	target   string
}

func (s *ActiveSpell) set(ui *fyne.Container, spell everquest.Spell) {
	s.id = spell.Id
	s.name = spell.Name
	s.maxTicks = spell.Duration
	if canFocus(configuration.Everquest.PlayerClass, spell) { // Increase the ticks if we are focusing the duration.
		s.maxTicks = int(float64(s.maxTicks) * configuration.Everquest.ExtendDotsPercent)
	}
	s.ui = ui
	s.target = "Unknown"
	index := s.findIndex()
	if index < 0 {
		ui.Add(container.NewHBox(
			widget.NewLabel(spell.Name),
			widget.NewProgressBar(),
		),
		)
		ui.Objects[s.findIndex()].(*fyne.Container).Objects[1].(*widget.ProgressBar).Max = float64(spell.Duration)
		if !configuration.Main.ShowUnknown {
			ui.Objects[s.findIndex()].(*fyne.Container).Hide()
		}
		ui.Refresh()
	} else { // UI Element already exists, recycle it
		if configuration.Main.ShowUnknown {
			ui.Objects[index].(*fyne.Container).Show()
		}
		ui.Objects[index].(*fyne.Container).Objects[1].(*widget.ProgressBar).SetValue(0)
	}
}

func (s *ActiveSpell) tick() {
	index := s.findIndex()
	s.ui.Objects[index].(*fyne.Container).Show()
	s.ticks++
	Debug.Printf("Adding a tick to %s, we are at %d ticks\n", s.name, s.ticks)
	if index < 0 {
		return
	}
	s.ui.Objects[index].(*fyne.Container).Objects[1].(*widget.ProgressBar).SetValue(float64(s.ticks))
	s.ui.Refresh()
	if s.isWarning() {
		if configuration.Main.UseTTS {
			playAudio(s.name)
		}
		Debug.Printf("Sending tick warning for %s at %d ticks", s.name, s.ticks)
	}
	if s.isComplete() {
		Debug.Printf("Removing ui elements for %s\n", s.name)
		s.cleanup()
	}
}

func (s *ActiveSpell) cleanup() {
	index := s.findIndex()
	Debug.Printf("Cleaning up %#+v with index %d\n", s, index)
	if index < 0 {
		return
	}
	s.ui.Objects[index].(*fyne.Container).Hide()
	s.ui.Refresh()
	// s.ui.Remove(s.ui.Objects[index])
}

func (s *ActiveSpell) isComplete() bool {
	if s.ticks >= SpellDB.GetSpellByID(s.id).Duration {
		if s.ticks > SpellDB.GetSpellByID(s.id).Duration {
			Err.Printf("%d:%s has overshot it's max tick currently %d :: max %d", s.id, s.name, s.ticks, SpellDB.GetSpellByID(s.id).Duration)
		}
		Debug.Printf("%d:%s has reached it's max tick of %d", s.id, s.name, s.ticks)
		return true
	}
	return false
}

func (s *ActiveSpell) isWarning() bool {
	if (SpellDB.GetSpellByID(s.id).Duration - s.ticks) == configuration.Main.RemainingTicksWarning { // Only deal with exact, otherwise we can get duplicates
		Debug.Printf("%d:%s has reached the warning level of ticks remaining, currently at %d ticks.", s.id, s.name, s.ticks)
		return true
	}
	return false
}

func (s *ActiveSpell) findIndex() int {
	if len(s.ui.Objects) <= 1 { // make sure there is an active spell
		return -2
	}
	for i, e := range s.ui.Objects {
		if i == 0 { // Skip mob name
			continue
		}
		// content.Objects[1].(*fyne.Container).Objects[1].(*widget.ProgressBar).SetValue(0.75)
		if e.(*fyne.Container).Objects[0].(*widget.Label).Text == s.name {
			return i
		}
	}
	return -1
}

func canFocus(class string, spell everquest.Spell) bool {
	for _, usable := range spell.GetClasses() {
		if usable == class {
			// TODO: Check if level can be focused
			Debug.Printf("%s can use and therefore focus %s", class, spell.Name)
			return true
		}
	}
	return false
}

func findSpellInstance(mob, spell string) int {
	for i, s := range *active[spell] {
		if s.name == spell && s.target == mob {
			return i
		}
	}
	for i, s := range *active[spell] {
		if s.name == spell && s.target == "Unknown" { // Spell not assigned a mob
			return i
		}
	}
	Debug.Printf("Cannot find spell %s instance for %s", spell, mob)
	return -1
}

func removeInstance(instance int, spellName string) {
	// Remove the element at index i from a.
	(*active[spellName])[instance] = (*active[spellName])[len((*active[spellName]))-1] // Copy last element to index i.
	(*active[spellName])[len((*active[spellName]))-1] = ActiveSpell{}                  // Erase last element (write zero value).
	(*active[spellName]) = (*active[spellName])[:len((*active[spellName]))-1]          // Truncate slice.
}

func cleanupMob(mob string) int {
	var found int
	for i, s := range active {
		for h, ms := range *s {
			if ms.target == mob {
				ms.cleanup()
				removeInstance(h, i)
				found++
			}
		}
	}
	return found
}

func cleanupAll() {
	for _, s := range active {
		for _, ms := range *s {
			ms.cleanup()
		}
	}
	active = make(map[string]*[]ActiveSpell)
}
