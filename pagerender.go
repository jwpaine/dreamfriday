package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	Models "dreamfriday/models"
)

type GenericComponent struct {
	Text       string
	Type       string // "P", "H1", "H2", etc.
	Attributes map[string]string
	Children   []Component
	styling    string
}

func generateRandomClassName(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	rng := rand.New(rand.NewSource(time.Now().UnixNano())) // Use a new random generator with a unique seed
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rng.Intn(len(letters))]
	}
	return string(b)
}

func GenerateCSS(className string, cssProperties map[string]string, mqType string, target string) string {
	if len(cssProperties) == 0 {
		return ""
	}

	var cssContent string
	for property, value := range cssProperties {
		cssContent += fmt.Sprintf(" %s: %s;", property, value)
	}

	// Wrap in media query if needed
	if mqType != "" && target != "" {
		return fmt.Sprintf("@media only screen and (%s: %s) { .%s {%s } }", mqType, target, className, cssContent)
	}

	return fmt.Sprintf(".%s {%s }", className, cssContent)
}

func (g *GenericComponent) Render(ctx context.Context, w io.Writer) error {
	var attrs []string
	for attr, value := range g.Attributes {
		attrs = append(attrs, fmt.Sprintf("%s=\"%s\"", attr, value))
	}

	// Render the opening <p> tag with attributes
	_, err := w.Write([]byte(fmt.Sprintf("<%s %s>", g.Type, strings.Join(attrs, " "))))
	if err != nil {
		return err
	}

	// Render the text content, if present
	if g.Text != "" {
		_, err = w.Write([]byte(g.Text))
		if err != nil {
			return err
		}
	}

	// Render child components, if any (e.g., <a> tags within the paragraph)
	for _, child := range g.Children {
		err := child.Render(ctx, w)
		if err != nil {
			return err
		}
	}

	// Render the closing </p> tag
	_, err = w.Write([]byte(fmt.Sprintf("</%s>", g.Type)))
	return err
}

// Component interface for rendering elements
type Component interface {
	Render(ctx context.Context, w io.Writer) error
	Styling() string
}

func (g *GenericComponent) Styling() string {
	return g.styling
}

func extractStyles(styleAttr interface{}) (map[string]string, map[string]map[string]map[string]string) {
	cssProps := map[string]string{}
	mediaQueries := map[string]map[string]map[string]string{}

	if styleMap, ok := styleAttr.(map[string]interface{}); ok {
		for key, value := range styleMap {
			switch v := value.(type) {
			case string:
				// Standard CSS property
				cssProps[key] = v
			case map[string]interface{}:
				// Handle media queries
				if key == "media" {
					for mqType, mqSettings := range v {
						if _, exists := mediaQueries[mqType]; !exists {
							mediaQueries[mqType] = map[string]map[string]string{}
						}
						for target, styleMap := range mqSettings.(map[string]interface{}) {
							queryProps := map[string]string{}
							if styleProps, ok := styleMap.(map[string]interface{}); ok {
								for prop, val := range styleProps {
									if styleValue, ok := val.(string); ok {
										queryProps[prop] = styleValue
									}
								}
							}
							// Store the queryProps under the specific target
							mediaQueries[mqType][target] = queryProps
						}
					}
				}
			}
		}
	}

	return cssProps, mediaQueries
}

func CreateComponent(componentType string, element Models.PageElement, children []Component) (Component, error) {
	attr := map[string]string{}

	// Flatten `props` and add to `attr`
	for key, value := range element.Attributes.Props {
		attr[key] = value
	}

	className := ""
	// Generate a random class name and append user-supplied classes if any
	if element.Attributes.Style != nil {
		className = fmt.Sprintf("%s_%s", componentType, generateRandomClassName(6))
	}
	if existingClass, exists := attr["class"]; exists && existingClass != "" {
		attr["class"] = className + " " + existingClass
	} else {
		attr["class"] = className
	}

	// Process Style for CSS properties
	cssProps, mediaQueries := extractStyles(element.Attributes.Style)

	// Generate base CSS and media query CSS
	styling := GenerateCSS(className, cssProps, "", "")
	for mqType, targets := range mediaQueries {
		for target, styles := range targets {
			styling += GenerateCSS(className, styles, mqType, target)
		}
	}
	// fmt.Println("rendering type: ", componentType)
	// if element.Type == "" {
	// 	return nil, fmt.Errorf("Element type is required")
	// }
	return &GenericComponent{Type: element.Type, Text: element.Text, Attributes: attr, Children: children, styling: styling}, nil
}

func RenderPageContent(ctx context.Context, components map[string]Models.Page, elements []Models.PageElement, w io.Writer) ([]Component, string, error) {
	var renderedComponents []Component
	var allCSS string

	for _, element := range elements {
		elementType := element.Type
		importComponent := element.Import
		children := []Component{}

		// Process the importComponent, if specified and present in the components map
		if importComponent != "" {
			if importedPage, exists := components[importComponent]; exists {
				// Recursively render elements of the imported component
				importedChildren, importedCSS, err := RenderPageContent(ctx, components, importedPage.Elements, w)
				if err != nil {
					return nil, "", err
				}
				// Append all imported children directly to the current children list
				children = append(children, importedChildren...)
				// Accumulate CSS from the imported component
				allCSS += importedCSS
			} else {
				fmt.Println("Component not found for import:", importComponent)
				continue
			}
		}

		// Skip rendering if the current element has a blank type and isn’t an imported component
		if elementType == "" && importComponent == "" {
			fmt.Println("Skipping element with blank type")
			continue
		}

		// Recursively process nested elements if any, skipping elements with blank types
		if len(element.Elements) > 0 {
			nestedChildren, nestedCSS, err := RenderPageContent(ctx, components, element.Elements, w)
			if err != nil {
				return nil, "", err
			}
			children = append(children, nestedChildren...)
			allCSS += nestedCSS
		}

		// Create the component with its children, only if it has a non-blank type
		if elementType != "" {
			component, err := CreateComponent(elementType, element, children)
			if err != nil {
				return nil, "", err
			}

			// Collect CSS from each component
			allCSS += component.Styling()

			// Add the component to the rendered components list
			renderedComponents = append(renderedComponents, component)

			// Render HTML for each component
			if err := component.Render(ctx, w); err != nil {
				return nil, "", err
			}
		} else {
			// If the main element has a blank type but includes imported children, add them directly
			renderedComponents = append(renderedComponents, children...)
		}
	}

	return renderedComponents, allCSS, nil
}

func RenderJSONContent(c echo.Context, reusable interface{}, jsonContent interface{}, previewMode bool) error {
	ctx := c.Request().Context()

	// Check that jsonContent is a slice of PageElement
	pageContent, ok := jsonContent.([]Models.PageElement)
	if !ok {
		return c.String(http.StatusBadRequest, "Invalid content structure, expected []PageElement")
	}

	components, ok := reusable.(map[string]Models.Page)
	// interate through each component and print the key:
	// for key, value := range components {
	// 	fmt.Println("component Key:", key, "Value:", value)
	// }

	buffer := new(bytes.Buffer)

	// Call RenderPageContent to generate components and accumulate CSS
	renderedComponents, allCSS, err := RenderPageContent(ctx, components, pageContent, buffer)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Error rendering page content: "+err.Error())
	}

	// Prepare HTML with the accumulated CSS
	var renderedHTML strings.Builder
	renderedHTML.WriteString("<!DOCTYPE html>\n")

	// hard-coded global defaults
	globalDefaults := `
		<meta name="viewport" content="width=device-width, initial-scale=1">
	`

	renderedHTML.WriteString("<head>\n")
	renderedHTML.WriteString(globalDefaults)
	renderedHTML.WriteString(fmt.Sprintf("<style>%s</style>\n", allCSS)) // Write all accumulated CSS

	// Conditionally add the preview mode div within the body
	if previewMode {
		renderedHTML.WriteString("<div id='preview'><span>Preview Mode Enabled</span><a href='/preview'>Disable</a></div>\n")
	}

	// Render all components to HTML content
	for _, component := range renderedComponents {
		if err := component.Render(ctx, &renderedHTML); err != nil {
			return c.String(http.StatusInternalServerError, "Error rendering component: "+err.Error())
		}
	}

	renderedHTML.WriteString("\n</body>\n</html>")
	return c.HTML(http.StatusOK, renderedHTML.String())
}
