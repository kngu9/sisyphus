// Copyright 2019 CanonicalLtd

package call

var (
	TimeNow = &timeNow
)

func RenderString(attr Attributes, str string) string {
	return attr.renderString(str)
}
