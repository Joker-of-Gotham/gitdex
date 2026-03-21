package views

import tea "charm.land/bubbletea/v2"

type Router struct {
	views  map[ID]View
	active ID
	order  []ID
	width  int
	height int
}

func NewRouter(initial ID, views ...View) *Router {
	r := &Router{
		views:  make(map[ID]View, len(views)),
		active: initial,
		order:  make([]ID, 0, len(views)),
	}
	for _, v := range views {
		r.views[v.ID()] = v
		r.order = append(r.order, v.ID())
	}
	return r
}

func (r *Router) Active() View {
	return r.views[r.active]
}

func (r *Router) ActiveID() ID {
	return r.active
}

func (r *Router) SwitchTo(id ID) {
	if _, ok := r.views[id]; ok {
		r.active = id
	}
}

func (r *Router) Order() []ID {
	return r.order
}

func (r *Router) ViewTitle(id ID) string {
	if v, ok := r.views[id]; ok {
		return v.Title()
	}
	return string(id)
}

func (r *Router) Init() tea.Cmd {
	var cmds []tea.Cmd
	for _, v := range r.views {
		if cmd := v.Init(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return tea.Batch(cmds...)
}

func (r *Router) Update(msg tea.Msg) tea.Cmd {
	if sw, ok := msg.(SwitchViewMsg); ok {
		r.SwitchTo(sw.Target)
		return nil
	}

	v := r.views[r.active]
	updated, cmd := v.Update(msg)
	r.views[r.active] = updated
	return cmd
}

func (r *Router) Render() string {
	return r.views[r.active].Render()
}

func (r *Router) SetSize(width, height int) {
	r.width = width
	r.height = height
	for id, v := range r.views {
		v.SetSize(width, height)
		r.views[id] = v
	}
}
