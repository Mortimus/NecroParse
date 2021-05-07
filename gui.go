package main

// import (
// 	"fyne.io/fyne"
// 	"fyne.io/fyne/widget"
// )

// var combatMobs CombatMobs

// func init() {
// 	combatMobs.init()
// }

// // show creates a new game and loads a table rendered in a new window.
// func show(app fyne.App) {
// 	w := app.NewWindow("NecroParse")
// 	w.SetPadded(false)
// 	w.SetContent(NewMobTable(combatMobs))
// 	w.Resize(fyne.NewSize(220, 140))

// 	w.Show()
// }

// var currentMob string

// type CombatMobs struct {
// 	widget.BaseWidget

// 	mobs map[string]ActiveSpell // Lookup by spell name
// }

// func (cm *CombatMobs) init() {
// 	cm.mobs = make(map[string]ActiveSpell)
// }

// // CreateRenderer gets the widget renderer for this table - internal use only
// func (cm *CombatMobs) CreateRenderer() fyne.WidgetRenderer {
// 	return newTableRender(cm)
// }

// // Table represents the rendering of a game in progress
// type Table struct {
// 	widget.BaseWidget

// 	game     *Game
// 	selected *Card
// }

// // NewTable creates a new table widget for the specified game
// func NewMobTable(g *Game) *CombatMobs {
// 	table := &Table{game: g}
// 	table.ExtendBaseWidget(table)
// 	return table
// }
