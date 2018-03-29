package icon

import (
	"github.com/GeorgeMac/idicon/colour"
	"github.com/GeorgeMac/idicon/icon"
	"strings"
)

var (
	generator *icon.Generator
)

func init() {
	var err error

	props := icon.DefaultProps()
	props.BaseColour = colour.NewColour(0xf2, 0xf2, 0xf2)

	generator, err = icon.NewGenerator(5, 5, icon.With(props))
	if err != nil {
		panic(err)
	}
}

// Generate identiconを生成し、そのsvgを返します
func Generate(salt string) string {
	return strings.Replace(generator.Generate([]byte(salt)).String(), `<svg`, `<svg xmlns="http://www.w3.org/2000/svg"`, 1)
}
