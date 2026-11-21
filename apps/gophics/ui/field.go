package ui

import (
	"github.com/doug/gophics/geom"
	"github.com/doug/gophics/shell"
	"github.com/doug/gophics/theme"
	"github.com/doug/gophics/widget"
)

// Field is theme.Field plus OnKeyPreview. theme.Field does not expose the
// underlying widget.TextField's OnKeyPreview, and keys reach only the focused
// widget, so without it no app shortcut works while a field has focus
// (fixpass issue 6). The look is theme.Field's own Build (theme/theme.go),
// with the same theme values; only OnKeyPreview is added.
type Field struct {
	Value        string
	Placeholder  string
	Multiline    bool
	Autofocus    bool
	OnChange     func(string)
	OnSubmit     func(string)
	OnKeyPreview func(shell.Key) bool
}

func (f Field) CreateState() widget.State { return &fieldState{} }

type fieldState struct {
	widget.StateBase[Field]
	focused bool
}

func (s *fieldState) Build(ctx widget.Ctx) widget.Widget {
	th := theme.Of(ctx)
	f := s.W()
	border, bw := th.Outline, float32(1)
	if s.focused {
		border, bw = th.Primary, 1.5
	}
	return widget.Decorated{
		Color: th.Surface, Radius: th.Radius, BorderColor: border, BorderWidth: bw,
		Child: widget.Padding{
			Insets: geom.InsetsSymmetric(12, 10),
			Child: widget.TextField{
				Autofocus:        f.Autofocus,
				Value:            f.Value,
				Placeholder:      f.Placeholder,
				Multiline:        f.Multiline,
				OnChange:         f.OnChange,
				OnSubmit:         f.OnSubmit,
				OnKeyPreview:     f.OnKeyPreview,
				OnFocus:          func(v bool) { s.SetState(func() { s.focused = v }) },
				TextColor:        th.Text,
				PlaceholderColor: th.Muted,
				CaretColor:       th.Primary,
				SelectionColor:   th.Selection,
			},
		},
	}
}
